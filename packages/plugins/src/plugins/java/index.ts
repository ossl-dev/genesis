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

export interface JavaOptions {
  version: string;
  distribution?: "openjdk" | "oracle";
}

export function java(options: JavaOptions): GenesisPluginInstance<JavaOptions> {
  return {
    id: "java",
    category: "language",
    module: "@ossl/genesis-plugins/java",
    options: {
      ...options,
      distribution: options.distribution ?? "openjdk",
    },
  };
}

/**
 * Parse Java version from command output
 */
function parseJavaVersion(output: string): string | undefined {
  const match = output.match(/version "(\d+\.\d+\.\d+_\d+)"/);
  if (match) {
    return match[1];
  }
  // Fallback for other version formats
  const altMatch = output.match(/(\d+\.\d+\.\d+)/);
  return altMatch?.[1];
}

/**
 * Get Java major version from version string
 */
function getJavaMajorVersion(version: string): string {
  if (version.startsWith("1.")) {
    return version.split(".")[1]; // Java 8 style: 1.8.0_291
  }
  return version.split(".")[0]; // Java 11+ style: 11.0.12
}

async function detectJava(runtime: PluginRuntime<JavaOptions>) {
  const result = await runCommand("java", ["-version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Java is not available on PATH",
    };
  }

  // Java version is typically printed to stderr
  const versionOutput = result.stderr || result.stdout || "";
  const version = parseJavaVersion(versionOutput);

  if (!version) {
    return {
      ok: false,
      details: "Java version could not be determined",
    };
  }

  const installedMajor = getJavaMajorVersion(version);
  const requestedMajor = getJavaMajorVersion(runtime.options.version);

  if (installedMajor === requestedMajor) {
    return {
      ok: true,
      details: `Detected Java ${version}`,
    };
  }

  return {
    ok: false,
    details: `Detected Java ${version} (major ${installedMajor}) but Java ${runtime.options.version} (major ${requestedMajor}) is requested`,
  };
}

/**
 * Download and extract Java JDK on Linux/macOS
 */
async function installJavaFromArchive(
  runtime: PluginRuntime<JavaOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const { version, distribution } = runtime.options;
  const platform = getPlatform();
  const arch = os.arch();

  logger.info(`Installing Java ${version} (${distribution})...`);

  // Determine the correct archive URL
  let downloadUrl: string;
  let fileName: string;

  if (distribution === "openjdk") {
    // Use Adoptium (Eclipse Temurin) for OpenJDK
    const majorVersion = getJavaMajorVersion(version);
    const archMap: Record<string, string> = {
      x64: "x64",
      arm64: platform === "macos" ? "aarch64" : "arm64",
    };
    const platformMap: Record<string, string> = {
      macos: "mac",
      linux: "linux",
    };

    const archSuffix = archMap[arch] || "x64";
    const platformSuffix = platformMap[platform] || "linux";

    downloadUrl = `https://github.com/adoptium/temurin${majorVersion}-binaries/releases/download/jdk-${version}/OpenJDK${majorVersion}-U_${archSuffix}_${platformSuffix}_hotspot_${version}.tar.gz`;
    fileName = `openjdk-${version}.tar.gz`;
  } else {
    // Oracle JDK (simplified - would need license acceptance in real implementation)
    logger.warn("Oracle JDK installation requires manual license acceptance");
    return {
      ok: false,
      details: "Oracle JDK requires manual download and license acceptance",
    };
  }

  try {
    // Download the archive
    const tempDir = os.tmpdir();
    const archivePath = path.join(tempDir, fileName);

    logger.debug(`Downloading Java from ${downloadUrl}`);
    const downloadResult = await runCommand("curl", [
      "-L",
      "-o",
      archivePath,
      downloadUrl,
    ]);

    if (downloadResult.code !== 0) {
      throw new Error(`Failed to download Java: ${downloadResult.stderr}`);
    }

    // Extract to /usr/local/java or similar
    const installDir = platform === "macos" ? "/usr/local/java" : "/opt/java";
    const javaHome = path.join(installDir, `jdk-${version}`);

    // Ensure install directory exists
    await fs.promises.mkdir(installDir, { recursive: true });

    // Extract the archive
    logger.debug(`Extracting Java to ${javaHome}`);
    const extractResult = await runCommand("tar", [
      "-xzf",
      archivePath,
      "-C",
      installDir,
    ]);

    if (extractResult.code !== 0) {
      throw new Error(
        `Failed to extract Java archive: ${extractResult.stderr}`,
      );
    }

    // Rename the extracted folder to a consistent name
    const extractedDirs = await fs.promises.readdir(installDir);
    const extractedDir = extractedDirs.find(
      (dir) => dir.startsWith("jdk-") && dir !== `jdk-${version}`,
    );

    if (extractedDir && extractedDir !== `jdk-${version}`) {
      const oldPath = path.join(installDir, extractedDir);
      await fs.promises.rename(oldPath, javaHome);
    }

    // Clean up
    await fs.promises.unlink(archivePath);

    logger.info(`Java ${version} installed to ${javaHome}`);

    return {
      ok: true,
      details: `Java ${version} installed to ${javaHome}`,
    };
  } catch (error) {
    logger.error(`Failed to install Java: ${error}`);
    return {
      ok: false,
      details: `Java installation failed: ${error}`,
    };
  }
}

export function createPlugin(
  instance: GenesisPluginInstance<JavaOptions>,
): GenesisPlugin<JavaOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      return detectJava(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const platform = getPlatform();

      // Skip task registration for Windows (use installer)
      if (platform === "windows") {
        return;
      }

      logger.debug("Registering system tasks for Java installation");

      // Register package manager update (will be deduplicated across plugins)
      const updateTask = createPackageManagerUpdateTask(
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(updateTask);

      // Register curl for downloading Java archives
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
      const { distribution } = runtime.options;
      const platform = getPlatform();

      // Check if Java is already installed and correct version
      const detectResult = await detectJava(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Java is already installed");
        return {
          ok: true,
          didChange: false,
          details: detectResult.details,
        };
      }

      // Handle Windows separately
      if (platform === "windows") {
        logger.warn(
          "Automatic Java installation on Windows is not yet supported",
        );
        logger.info(
          "Please download and install Java manually from https://adoptium.net/ or https://www.oracle.com/java/technologies/downloads/",
        );
        return {
          ok: false,
          didChange: false,
          details:
            "Manual installation required. Visit https://adoptium.net/ for OpenJDK or https://www.oracle.com/java/technologies/downloads/ for Oracle JDK.",
        };
      }

      // macOS/Linux installation
      if (distribution === "openjdk") {
        logger.info("Installing OpenJDK from archive...");

        const installResult = await installJavaFromArchive(runtime);

        if (!installResult.ok) {
          logger.error("Failed to install Java");
          return {
            ok: false,
            didChange: false,
            details: installResult.details,
          };
        }

        // Set up JAVA_HOME and PATH
        const majorVersion = getJavaMajorVersion(runtime.options.version);
        const javaHome =
          platform === "macos"
            ? `/usr/local/java/jdk-${runtime.options.version}`
            : `/opt/java/jdk-${runtime.options.version}`;

        logger.info(`Setting JAVA_HOME to ${javaHome}`);
        logger.info(`Adding ${javaHome}/bin to PATH`);

        // Note: In a real implementation, we would update shell profiles
        // For now, we'll just provide instructions
        logger.info("");
        logger.info("=== Java Environment Setup ===");
        logger.info(
          "Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):",
        );
        logger.info(`export JAVA_HOME="${javaHome}"`);
        logger.info(`export PATH="$JAVA_HOME/bin:$PATH"`);
        logger.info("");
        logger.info("Then restart your shell or run: source ~/.bashrc");
        logger.info("================================");

        return {
          ok: true,
          didChange: true,
          details: installResult.details,
        };
      } else {
        // Oracle JDK
        logger.warn("Oracle JDK installation requires manual steps");
        logger.info(
          "Please download Oracle JDK from: https://www.oracle.com/java/technologies/downloads/",
        );
        return {
          ok: false,
          didChange: false,
          details: "Oracle JDK requires manual download and installation",
        };
      }
    },
    async validate(runtime) {
      const detectResult = await detectJava(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
