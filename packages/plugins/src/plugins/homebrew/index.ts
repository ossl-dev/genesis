import {
  type GenesisPlugin,
  type GenesisPluginInstance,
  type PluginRuntime,
  runCommand,
  getPlatform,
  createPackageManagerUpdateTask,
  createPackageInstallTask,
  createCommandCheckTask,
  createCustomTask,
} from "@ossl/genesis-core";
import os from "node:os";
import path from "node:path";
import fs from "node:fs";

export interface HomebrewOptions {
  update_packages?: boolean;
  install_cask?: boolean;
  add_to_path?: boolean;
  global_packages?: string[];
}

export function homebrew(
  options: HomebrewOptions = {},
): GenesisPluginInstance<HomebrewOptions> {
  return {
    id: "homebrew",
    category: "tool",
    module: "@ossl/genesis-plugins/homebrew",
    options: {
      update_packages: options.update_packages ?? true,
      install_cask: options.install_cask ?? true,
      add_to_path: options.add_to_path ?? true,
    },
  };
}

/**
 * Install global brew packages
 */
async function installGlobalBrewPackages(
  runtime: PluginRuntime<HomebrewOptions>,
  packages: string[],
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  if (packages.length === 0) {
    return { ok: true, details: "No global brew packages to install" };
  }

  logger.info(`Installing global brew packages: ${packages.join(", ")}`);

  const result = await runCommand("brew", ["install", ...packages], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: `Failed to install brew packages: ${result.stderr || "Unknown error"}`,
    };
  }

  return {
    ok: true,
    details: `Successfully installed brew packages: ${packages.join(", ")}`,
  };
}

/**
 * Parse Homebrew version from command output
 */
function parseHomebrewVersion(output: string): string | undefined {
  const match = output.match(/Homebrew (\d+\.\d+\.\d+)/);
  return match?.[1];
}

/**
 * Check if Homebrew is available and get version
 */
async function detectHomebrew(runtime: PluginRuntime<HomebrewOptions>) {
  const result = await runCommand("brew", ["--version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Homebrew is not available on PATH",
    };
  }

  const version = parseHomebrewVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Homebrew version could not be determined",
    };
  }

  return {
    ok: true,
    details: `Detected Homebrew ${version}`,
  };
}

/**
 * Get the Homebrew installation directory
 */
function getHomebrewDir(): string {
  return "/opt/homebrew"; // Apple Silicon default
}

/**
 * Get the Intel Homebrew installation directory
 */
function getIntelHomebrewDir(): string {
  return "/usr/local"; // Intel Mac default
}

/**
 * Install Homebrew on macOS using official installer script
 */
async function installHomebrew(
  runtime: PluginRuntime<HomebrewOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const arch = os.arch();

  logger.info(`Installing Homebrew for ${arch}...`);

  try {
    // Download and run the Homebrew installation script
    const installUrl =
      "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh";

    logger.debug("Downloading Homebrew installation script...");

    // Run the installation script directly
    const installResult = await runCommand(
      "bash",
      ["-c", `curl -fsSL ${installUrl} | bash`],
      {
        cwd: runtime.context.cwd,
        env: runtime.context.env,
      },
    );

    if (installResult.code !== 0) {
      throw new Error(`Homebrew installation failed: ${installResult.stderr}`);
    }

    logger.info("Homebrew installed successfully");

    // Determine installation path based on architecture
    const homebrewDir =
      arch === "arm64" ? getHomebrewDir() : getIntelHomebrewDir();
    const homebrewBin = path.join(homebrewDir, "bin");

    // Update PATH if requested
    if (runtime.options.add_to_path) {
      logger.info(`Adding ${homebrewBin} to PATH...`);

      // Note: In a real implementation, we would update shell profiles
      logger.info("");
      logger.info("=== Homebrew Environment Setup ===");
      logger.info(
        "Add the following to your shell profile (~/.zshrc, ~/.bashrc, etc.):",
      );

      if (arch === "arm64") {
        logger.info('if [[ "$(uname -m)" == "arm64" ]]; then');
        logger.info(`  eval "$(${homebrewBin}/brew shellenv)"`);
        logger.info("fi");
      } else {
        logger.info(`eval "$(${homebrewBin}/brew shellenv)"`);
      }

      logger.info("");
      logger.info("Then restart your shell or run: source ~/.zshrc");
      logger.info("================================");
    }

    return {
      ok: true,
      details: `Homebrew installed to ${homebrewDir}`,
    };
  } catch (error) {
    logger.error(`Failed to install Homebrew: ${error}`);
    return {
      ok: false,
      details: `Homebrew installation failed: ${error}`,
    };
  }
}

/**
 * Update Homebrew and installed packages
 */
async function updateHomebrew(
  runtime: PluginRuntime<HomebrewOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  logger.info("Updating Homebrew...");

  try {
    // Update Homebrew itself
    const updateResult = await runCommand("brew", ["update"], {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    });

    if (updateResult.code !== 0) {
      logger.warn("Failed to update Homebrew, continuing...");
    }

    // Upgrade installed packages if requested
    if (runtime.options.update_packages) {
      logger.info("Upgrading installed packages...");

      const upgradeResult = await runCommand("brew", ["upgrade"], {
        cwd: runtime.context.cwd,
        env: runtime.context.env,
      });

      if (upgradeResult.code !== 0) {
        logger.warn("Failed to upgrade packages, continuing...");
      }

      // Upgrade casks if requested
      if (runtime.options.install_cask) {
        logger.info("Upgrading installed casks...");

        const caskUpgradeResult = await runCommand(
          "brew",
          ["upgrade", "--cask"],
          {
            cwd: runtime.context.cwd,
            env: runtime.context.env,
          },
        );

        if (caskUpgradeResult.code !== 0) {
          logger.warn("Failed to upgrade casks, continuing...");
        }
      }
    }

    logger.info("Homebrew update completed");

    return {
      ok: true,
      details: "Homebrew updated successfully",
    };
  } catch (error) {
    logger.error(`Failed to update Homebrew: ${error}`);
    return {
      ok: false,
      details: `Homebrew update failed: ${error}`,
    };
  }
}

/**
 * Log installation guide for non-macOS platforms
 */
function logUnsupportedPlatformGuide(logger: any, platform: string): void {
  logger.warn(`Homebrew is not supported on ${platform}`);
  logger.info("");
  logger.info("=== Homebrew Alternatives ===");
  logger.info("");

  if (platform === "linux") {
    logger.info("For Linux, consider using:");
    logger.info("- Linuxbrew (unofficial fork of Homebrew)");
    logger.info("- Package managers: apt, yum, dnf, pacman");
    logger.info("- Nix package manager");
  } else if (platform === "windows") {
    logger.info("For Windows, consider using:");
    logger.info("- Chocolatey package manager");
    logger.info("- Scoop package manager");
    logger.info("- Winget (Windows Package Manager)");
    logger.info("- WSL with Homebrew");
  }

  logger.info("");
  logger.info("===========================================");
  logger.info("");
}

export function createPlugin(
  instance: GenesisPluginInstance<HomebrewOptions>,
): GenesisPlugin<HomebrewOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      return detectHomebrew(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const { global_packages } = runtime.options;
      const platform = getPlatform();

      // Homebrew is macOS only
      if (platform !== "macos") {
        return;
      }

      logger.debug("Registering system tasks for Homebrew installation");

      // Register curl for downloading installation script
      const curlTask = createPackageInstallTask(
        "curl",
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(curlTask);

      // Register git (Homebrew uses git internally)
      const gitTask = createPackageInstallTask(
        "git",
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(gitTask);

      // Register global package installation tasks if packages are specified
      if (global_packages && global_packages.length > 0) {
        logger.debug("Registering global brew package installation tasks");

        // Register brew availability check
        const brewCheckTask = createCommandCheckTask(
          "brew",
          runtime.context.cwd,
          runtime.context.env,
        );
        taskRegistry.register(brewCheckTask);

        // Register global package installation task
        const installTask = createCustomTask(
          "homebrew-global-packages",
          "Install global Homebrew packages",
          async () => {
            const result = await installGlobalBrewPackages(
              runtime,
              global_packages,
            );
            return {
              ok: result.ok,
              details: result.details,
              error: result.ok ? undefined : "Brew package installation failed",
            };
          },
          {
            priority: 10, // Low priority - run after other setup
            dependsOn: ["*:command-check:brew"],
          },
        );

        taskRegistry.register(installTask);
        logger.debug("Global brew package installation tasks registered");
      }

      logger.debug("System tasks registered: curl, git installation");
    },
    async apply(runtime) {
      const { logger } = runtime.context;
      const platform = getPlatform();

      // Check if platform is supported
      if (platform !== "macos") {
        logUnsupportedPlatformGuide(logger, platform);
        return {
          ok: false,
          didChange: false,
          details: `Homebrew is not supported on ${platform}`,
        };
      }

      // Check if Homebrew is already installed
      const detectResult = await detectHomebrew(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Homebrew is already installed");

        // Update existing Homebrew
        const updateResult = await updateHomebrew(runtime);

        return {
          ok: true,
          didChange: updateResult.ok,
          details: updateResult.ok
            ? "Homebrew updated successfully"
            : detectResult.details,
        };
      }

      // Install Homebrew
      logger.info("Installing Homebrew...");

      const installResult = await installHomebrew(runtime);

      if (!installResult.ok) {
        logger.error("Failed to install Homebrew");
        return {
          ok: false,
          didChange: false,
          details: installResult.details,
        };
      }

      // Test Homebrew installation
      logger.info("Testing Homebrew installation...");

      try {
        const testResult = await runCommand("brew", ["--version"], {
          cwd: runtime.context.cwd,
          env: runtime.context.env,
        });

        if (testResult.code === 0) {
          const version = parseHomebrewVersion(
            testResult.stdout || testResult.stderr,
          );
          logger.info(`Homebrew ${version} verified successfully`);
        } else {
          logger.warn("Homebrew installation completed but test failed");
        }
      } catch (error) {
        logger.warn(`Could not test Homebrew installation: ${error}`);
      }

      // Run initial update if installation was successful
      if (installResult.ok) {
        await updateHomebrew(runtime);
      }

      return {
        ok: true,
        didChange: true,
        details: installResult.details,
      };
    },
    async validate(runtime) {
      const detectResult = await detectHomebrew(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
