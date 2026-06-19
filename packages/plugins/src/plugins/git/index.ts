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

export interface GitOptions {
  version?: string;
  install_method?: "package" | "source" | "binary";
}

export function git(
  options: GitOptions = {},
): GenesisPluginInstance<GitOptions> {
  return {
    id: "git",
    category: "tool",
    module: "@ossl/genesis-plugins/git",
    options: {
      version: options.version ?? "latest",
      install_method: options.install_method ?? "package",
    },
  };
}

/**
 * Parse Git version from command output
 */
function parseGitVersion(output: string): string | undefined {
  const match = output.match(/git version (\d+\.\d+\.\d+)/);
  return match?.[1];
}

/**
 * Check if Git is available and get version
 */
async function detectGit(runtime: PluginRuntime<GitOptions>) {
  const result = await runCommand("git", ["--version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Git is not available on PATH",
    };
  }

  const version = parseGitVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Git version could not be determined",
    };
  }

  // If specific version requested, check if it matches
  if (runtime.options.version && runtime.options.version !== "latest") {
    if (version.startsWith(runtime.options.version)) {
      return {
        ok: true,
        details: `Detected Git ${version}`,
      };
    }
    return {
      ok: false,
      details: `Detected Git ${version} but ${runtime.options.version} is requested`,
    };
  }

  return {
    ok: true,
    details: `Detected Git ${version}`,
  };
}

/**
 * Install Git from source on Linux/macOS
 */
async function installGitFromSource(
  runtime: PluginRuntime<GitOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const platform = getPlatform();

  logger.info("Installing Git from source...");

  try {
    const tempDir = os.tmpdir();
    const gitSourceDir = path.join(tempDir, "git-source");

    // Clean up any previous source directory
    if (fs.existsSync(gitSourceDir)) {
      await fs.promises.rm(gitSourceDir, { recursive: true, force: true });
    }

    // Clone Git source repository
    logger.debug("Cloning Git source repository");
    const cloneResult = await runCommand("git", [
      "clone",
      "https://github.com/git/git.git",
      gitSourceDir,
    ]);

    if (cloneResult.code !== 0) {
      throw new Error(`Failed to clone Git repository: ${cloneResult.stderr}`);
    }

    // Install dependencies based on platform
    const dependencies =
      platform === "macos"
        ? ["gettext", "openssl", "zlib"]
        : ["libssl-dev", "libcurl4-openssl-dev", "zlib1g-dev", "libexpat1-dev"];

    for (const dep of dependencies) {
      const depResult = await runCommand(
        platform === "macos" ? "brew" : "sudo apt-get",
        ["install", "-y", dep],
      );
      if (depResult.code !== 0) {
        logger.warn(`Failed to install ${dep}: ${depResult.stderr}`);
      }
    }

    // Build and install Git
    logger.debug("Building Git from source");
    const buildCommands = ["make all", "sudo make install"];

    for (const cmd of buildCommands) {
      const [command, ...args] = cmd.split(" ");
      const buildResult = await runCommand(command, args, {
        cwd: gitSourceDir,
      });

      if (buildResult.code !== 0) {
        throw new Error(`Failed to build Git: ${buildResult.stderr}`);
      }
    }

    // Clean up
    await fs.promises.rm(gitSourceDir, { recursive: true, force: true });

    logger.info("Git installed from source successfully");

    return {
      ok: true,
      details: "Git installed from source",
    };
  } catch (error) {
    logger.error(`Failed to install Git from source: ${error}`);
    return {
      ok: false,
      details: `Git source installation failed: ${error}`,
    };
  }
}

/**
 * Download and install Git binary on Linux/macOS
 */
async function installGitFromBinary(
  runtime: PluginRuntime<GitOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const platform = getPlatform();
  const arch = os.arch();

  logger.info("Installing Git from binary...");

  // Determine the correct binary URL
  const archMap: Record<string, string> = {
    x64: "x86_64",
    arm64: "arm64",
  };
  const platformMap: Record<string, string> = {
    macos: "darwin",
    linux: "linux",
  };

  const archSuffix = archMap[arch] || "x86_64";
  const platformSuffix = platformMap[platform] || "linux";

  // Use GitHub releases for Git binaries
  const fileName = `git-${platformSuffix}-${archSuffix}.tar.gz`;
  const downloadUrl = `https://github.com/git/git/releases/download/v2.43.0/${fileName}`;

  try {
    // Download the binary
    const tempDir = os.tmpdir();
    const archivePath = path.join(tempDir, fileName);

    logger.debug(`Downloading Git from ${downloadUrl}`);
    const downloadResult = await runCommand("curl", [
      "-L",
      "-o",
      archivePath,
      downloadUrl,
    ]);

    if (downloadResult.code !== 0) {
      throw new Error(`Failed to download Git: ${downloadResult.stderr}`);
    }

    // Extract to /usr/local
    logger.debug(`Extracting Git to /usr/local`);
    const extractResult = await runCommand("tar", [
      "-C",
      "/usr/local",
      "-xzf",
      archivePath,
    ]);

    if (extractResult.code !== 0) {
      throw new Error(`Failed to extract Git binary: ${extractResult.stderr}`);
    }

    // Clean up
    await fs.promises.unlink(archivePath);

    logger.info("Git binary installed successfully");

    return {
      ok: true,
      details: "Git installed from binary",
    };
  } catch (error) {
    logger.error(`Failed to install Git binary: ${error}`);
    return {
      ok: false,
      details: `Git binary installation failed: ${error}`,
    };
  }
}

/**
 * Log Windows installation guide for Git
 */
function logWindowsGitGuide(logger: any): void {
  logger.warn("Automatic Git installation on Windows requires manual steps");
  logger.info("");
  logger.info("=== Git for Windows Installation Guide ===");
  logger.info("");
  logger.info("1. Visit the Git downloads page:");
  logger.info("   https://git-scm.com/download/win");
  logger.info("");
  logger.info("2. Download the Git installer:");
  logger.info("   - The download should start automatically");
  logger.info("   - Or click '64-bit Git for Windows Setup'");
  logger.info("");
  logger.info("3. Run the installer:");
  logger.info("   - Double-click the downloaded .exe file");
  logger.info("   - Follow the installation wizard");
  logger.info("   - Accept default settings or customize as needed");
  logger.info("");
  logger.info("4. Verify installation:");
  logger.info("   - Open Command Prompt, PowerShell, or Git Bash");
  logger.info("   - Run: git --version");
  logger.info("");
  logger.info("5. Configure Git (optional):");
  logger.info("   git config --global user.name 'Your Name'");
  logger.info("   git config --global user.email 'your.email@example.com'");
  logger.info("");
  logger.info("===========================================");
  logger.info("");
}

export function createPlugin(
  instance: GenesisPluginInstance<GitOptions>,
): GenesisPlugin<GitOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      return detectGit(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const { install_method } = runtime.options;
      const platform = getPlatform();

      // Skip task registration for Windows (use installer)
      if (platform === "windows") {
        return;
      }

      logger.debug("Registering system tasks for Git installation");

      if (install_method === "package") {
        // Register package manager update (will be deduplicated across plugins)
        const updateTask = createPackageManagerUpdateTask(
          runtime.context.cwd,
          runtime.context.env,
        );
        taskRegistry.register(updateTask);

        // Register Git installation via package manager
        const packageName = platform === "macos" ? "git" : "git-all";
        const gitTask = createPackageInstallTask(
          packageName,
          runtime.context.cwd,
          runtime.context.env,
        );
        taskRegistry.register(gitTask);

        logger.debug(
          `System tasks registered: package manager update, ${packageName} installation`,
        );
      } else {
        // For source or binary installation, we need build tools
        const updateTask = createPackageManagerUpdateTask(
          runtime.context.cwd,
          runtime.context.env,
        );
        taskRegistry.register(updateTask);

        if (install_method === "source") {
          // Source installation needs build tools
          const buildPackages =
            platform === "macos"
              ? ["make", "gcc", "autoconf"]
              : ["build-essential", "autoconf", "make"];

          for (const pkg of buildPackages) {
            const task = createPackageInstallTask(
              pkg,
              runtime.context.cwd,
              runtime.context.env,
            );
            taskRegistry.register(task);
          }
        } else {
          // Binary installation needs curl and tar
          const curlTask = createPackageInstallTask(
            "curl",
            runtime.context.cwd,
            runtime.context.env,
          );
          taskRegistry.register(curlTask);

          const tarTask = createPackageInstallTask(
            "tar",
            runtime.context.cwd,
            runtime.context.env,
          );
          taskRegistry.register(tarTask);
        }

        logger.debug(
          `System tasks registered: package manager update, build tools for ${install_method} installation`,
        );
      }
    },
    async apply(runtime) {
      const { logger } = runtime.context;
      const { install_method } = runtime.options;
      const platform = getPlatform();

      // Check if Git is already installed and correct version
      const detectResult = await detectGit(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Git is already installed");
        return {
          ok: true,
          didChange: false,
          details: detectResult.details,
        };
      }

      // Handle Windows separately
      if (platform === "windows") {
        logWindowsGitGuide(logger);
        return {
          ok: false,
          didChange: false,
          details:
            "Automatic installation not supported on Windows. See installation guide in logs.",
        };
      }

      // macOS/Linux installation
      let installResult: { ok: boolean; details: string };

      switch (install_method) {
        case "package":
          logger.info("Installing Git via package manager...");
          // Package installation is handled by task registry
          installResult = {
            ok: true,
            details: "Git installed via package manager",
          };
          break;

        case "source":
          installResult = await installGitFromSource(runtime);
          break;

        case "binary":
          installResult = await installGitFromBinary(runtime);
          break;

        default:
          installResult = {
            ok: false,
            details: `Unknown installation method: ${install_method}`,
          };
      }

      if (!installResult.ok) {
        logger.error("Failed to install Git");
        return {
          ok: false,
          didChange: false,
          details: installResult.details,
        };
      }

      // Configure Git with sensible defaults
      logger.info("Configuring Git with default settings...");

      try {
        // Set some sensible defaults
        await runCommand("git", [
          "config",
          "--global",
          "init.defaultBranch",
          "main",
        ]);
        await runCommand("git", ["config", "--global", "pull.rebase", "false"]);

        logger.info("Git configured with default settings");
      } catch (error) {
        logger.warn(`Failed to configure Git: ${error}`);
      }

      logger.info("Git installation completed successfully");
      return {
        ok: true,
        didChange: true,
        details: installResult.details,
      };
    },
    async validate(runtime) {
      const detectResult = await detectGit(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
