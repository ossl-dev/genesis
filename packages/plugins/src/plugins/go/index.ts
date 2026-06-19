import {
  type GenesisPlugin,
  type GenesisPluginInstance,
  type PluginRuntime,
  runCommand,
  getPlatform,
  createPackageManagerUpdateTask,
  createPackageInstallTask,
} from "@ossl/genesis-core";
import os from "node:os";
import path from "node:path";
import fs from "node:fs";

export interface GoOptions {
  version: string;
}

export function go(options: GoOptions): GenesisPluginInstance<GoOptions> {
  return {
    id: "go",
    category: "language",
    module: "@ossl/genesis-plugins/go",
    options,
  };
}

/**
 * Parse Go version from command output
 */
function parseGoVersion(output: string): string | undefined {
  const match = output.match(/go version go(\d+\.\d+\.\d+)/);
  return match?.[1];
}

/**
 * Get the Go installation directory
 */
function getGoDir(): string {
  return "/usr/local/go";
}

async function detectGo(runtime: PluginRuntime<GoOptions>) {
  const result = await runCommand("go", ["version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Go is not available on PATH",
    };
  }

  const version = parseGoVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Go version could not be determined",
    };
  }

  // Check if the installed version matches the requested version
  if (version.startsWith(runtime.options.version)) {
    return {
      ok: true,
      details: `Detected Go ${version}`,
    };
  }

  return {
    ok: false,
    details: `Detected Go ${version} but ${runtime.options.version} is requested`,
  };
}

/**
 * Download and extract Go on Linux/macOS
 */
async function installGoFromArchive(
  runtime: PluginRuntime<GoOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const { version } = runtime.options;
  const platform = getPlatform();
  const arch = os.arch();

  logger.info(`Installing Go ${version}...`);

  // Determine the correct archive URL
  const archMap: Record<string, string> = {
    x64: "amd64",
    arm64: "arm64",
  };
  const platformMap: Record<string, string> = {
    macos: "darwin",
    linux: "linux",
  };

  const archSuffix = archMap[arch] || "amd64";
  const platformSuffix = platformMap[platform] || "linux";

  const fileName = `go${version}.${platformSuffix}-${archSuffix}.tar.gz`;
  const downloadUrl = `https://go.dev/dl/${fileName}`;

  try {
    // Download the archive
    const tempDir = os.tmpdir();
    const archivePath = path.join(tempDir, fileName);

    logger.debug(`Downloading Go from ${downloadUrl}`);
    const downloadResult = await runCommand("curl", [
      "-L",
      "-o",
      archivePath,
      downloadUrl,
    ]);

    if (downloadResult.code !== 0) {
      throw new Error(`Failed to download Go: ${downloadResult.stderr}`);
    }

    // Remove any previous Go installation
    const goDir = getGoDir();
    if (fs.existsSync(goDir)) {
      logger.debug(`Removing previous Go installation at ${goDir}`);
      await fs.promises.rm(goDir, { recursive: true, force: true });
    }

    // Extract to /usr/local
    logger.debug(`Extracting Go to /usr/local`);
    const extractResult = await runCommand("tar", [
      "-C",
      "/usr/local",
      "-xzf",
      archivePath,
    ]);

    if (extractResult.code !== 0) {
      throw new Error(`Failed to extract Go archive: ${extractResult.stderr}`);
    }

    // Clean up
    await fs.promises.unlink(archivePath);

    logger.info(`Go ${version} installed to ${goDir}`);

    return {
      ok: true,
      details: `Go ${version} installed to ${goDir}`,
    };
  } catch (error) {
    logger.error(`Failed to install Go: ${error}`);
    return {
      ok: false,
      details: `Go installation failed: ${error}`,
    };
  }
}

/**
 * Log Windows installation guide for Go
 */
function logWindowsGoGuide(logger: any, version: string): void {
  logger.warn("Automatic Go installation on Windows requires manual steps");
  logger.info("");
  logger.info("=== Go for Windows Installation Guide ===");
  logger.info("");
  logger.info("1. Visit the Go downloads page:");
  logger.info("   https://go.dev/dl/");
  logger.info("");
  logger.info(`2. Download the Go installer for Windows:`);
  logger.info(`   - Look for: go${version}.windows-amd64.msi`);
  logger.info("");
  logger.info("3. Run the installer:");
  logger.info("   - Double-click the downloaded .msi file");
  logger.info("   - Follow the installation wizard");
  logger.info("   - By default, Go installs to C:\\Program Files\\Go");
  logger.info("");
  logger.info("4. Verify installation:");
  logger.info("   - Open a new Command Prompt or PowerShell");
  logger.info("   - Run: go version");
  logger.info("");
  logger.info("5. The installer should automatically add Go to your PATH");
  logger.info("   - If not, add C:\\Program Files\\Go\\bin to PATH");
  logger.info("");
  logger.info("===========================================");
  logger.info("");
}

export function createPlugin(
  instance: GenesisPluginInstance<GoOptions>,
): GenesisPlugin<GoOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      return detectGo(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const platform = getPlatform();

      // Skip task registration for Windows (use installer)
      if (platform === "windows") {
        return;
      }

      logger.debug("Registering system tasks for Go installation");

      // Register package manager update (will be deduplicated across plugins)
      const updateTask = createPackageManagerUpdateTask(
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(updateTask);

      // Register curl for downloading Go archives
      const curlTask = createPackageInstallTask(
        "curl",
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(curlTask);

      // Register tar for extracting archives
      const tarTask = createPackageInstallTask(
        "tar",
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(tarTask);

      logger.debug(
        "System tasks registered: package manager update, curl, tar installation",
      );
    },
    async apply(runtime) {
      const { logger } = runtime.context;
      const platform = getPlatform();

      // Check if Go is already installed and correct version
      const detectResult = await detectGo(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Go is already installed");
        return {
          ok: true,
          didChange: false,
          details: detectResult.details,
        };
      }

      // Handle Windows separately
      if (platform === "windows") {
        logWindowsGoGuide(logger, runtime.options.version);
        return {
          ok: false,
          didChange: false,
          details:
            "Automatic installation not supported on Windows. See installation guide in logs.",
        };
      }

      // macOS/Linux installation
      logger.info("Installing Go from archive...");

      const installResult = await installGoFromArchive(runtime);

      if (!installResult.ok) {
        logger.error("Failed to install Go");
        return {
          ok: false,
          didChange: false,
          details: installResult.details,
        };
      }

      // Set up PATH
      const goDir = getGoDir();
      const goBin = path.join(goDir, "bin");

      logger.info(`Adding ${goBin} to PATH`);

      // Note: In a real implementation, we would update shell profiles
      // For now, we'll just provide instructions
      logger.info("");
      logger.info("=== Go Environment Setup ===");
      logger.info(
        "Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):",
      );
      logger.info(`export PATH="${goBin}:$PATH"`);
      logger.info("");
      logger.info("Then restart your shell or run: source ~/.bashrc");
      logger.info("================================");

      return {
        ok: true,
        didChange: true,
        details: installResult.details,
      };
    },
    async validate(runtime) {
      const detectResult = await detectGo(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
