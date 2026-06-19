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

export interface NodeOptions {
  version: string;
  use_nvm?: boolean;
  global_packages?: string[];
}

export function node(options: NodeOptions): GenesisPluginInstance<NodeOptions> {
  return {
    id: "node",
    category: "tool",
    module: "@ossl/genesis-plugins/node",
    options: {
      ...options,
      use_nvm: options.use_nvm ?? true,
    },
  };
}

/**
 * Parse Node.js version from command output
 */
function parseNodeVersion(output: string): string | undefined {
  const value = output.trim();
  if (!value) {
    return undefined;
  }
  if (value.startsWith("v")) {
    return value.slice(1);
  }
  return value;
}

/**
 * Get the NVM directory path
 */
function getNvmDir(env: NodeJS.ProcessEnv): string {
  if (env.NVM_DIR) {
    return env.NVM_DIR;
  }
  const home = os.homedir();
  const xdgConfigHome = env.XDG_CONFIG_HOME;
  if (xdgConfigHome) {
    return path.join(xdgConfigHome, "nvm");
  }
  return path.join(home, ".nvm");
}

/**
 * Check if NVM is installed
 */
async function isNvmInstalled(
  runtime: PluginRuntime<NodeOptions>,
): Promise<boolean> {
  const nvmDir = getNvmDir(runtime.context.env);
  const nvmScript = path.join(nvmDir, "nvm.sh");

  try {
    await fs.promises.access(nvmScript, fs.constants.F_OK);
    return true;
  } catch {
    return false;
  }
}

/**
 * Check if NVM command is available in the current shell
 */
async function isNvmAvailable(
  runtime: PluginRuntime<NodeOptions>,
): Promise<boolean> {
  // NVM is a shell function, so we need to check if it's sourced
  // We'll try to run a command that sources NVM and checks if it exists
  const nvmDir = getNvmDir(runtime.context.env);
  const nvmScript = path.join(nvmDir, "nvm.sh");

  const result = await runCommand(
    "bash",
    ["-c", `source "${nvmScript}" && command -v nvm`],
    {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    },
  );

  return result.code === 0 && result.stdout.trim() === "nvm";
}

/**
 * Install NVM on macOS/Linux using the official install script
 */
async function installNvm(
  runtime: PluginRuntime<NodeOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const platform = getPlatform();

  if (platform === "windows") {
    return {
      ok: false,
      details:
        "NVM installation on Windows requires manual setup. See details in logs.",
    };
  }

  logger.info("Installing NVM...");

  const nvmVersion = "v0.40.3";
  const installUrl = `https://raw.githubusercontent.com/nvm-sh/nvm/${nvmVersion}/install.sh`;

  logger.debug(`Downloading NVM install script from ${installUrl}`);

  // Download and run the install script (curl should be available from registerTasks phase)
  const result = await runCommand(
    "bash",
    ["-c", `curl -o- ${installUrl} | bash`],
    {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    },
  );

  if (result.code !== 0) {
    logger.error("Failed to install NVM");
    logger.debug(`stderr: ${result.stderr}`);
    return {
      ok: false,
      details: `NVM installation failed: ${result.stderr || "Unknown error"}`,
    };
  }

  logger.info("NVM installed successfully");
  logger.info(
    "Note: You may need to restart your shell or source your profile for NVM to be available",
  );

  return {
    ok: true,
    details:
      "NVM installed successfully. Restart your shell or source your profile.",
  };
}

/**
 * Install Node.js using NVM
 */
async function installNodeViaNvm(
  runtime: PluginRuntime<NodeOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const { version } = runtime.options;

  logger.info(`Installing Node.js ${version} via NVM...`);

  const nvmDir = getNvmDir(runtime.context.env);
  const nvmScript = path.join(nvmDir, "nvm.sh");

  // Install the requested Node version
  const installResult = await runCommand(
    "bash",
    ["-c", `source "${nvmScript}" && nvm install ${version}`],
    {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    },
  );

  if (installResult.code !== 0) {
    logger.error(`Failed to install Node.js ${version}`);
    logger.debug(`stderr: ${installResult.stderr}`);
    return {
      ok: false,
      details: `Node.js installation failed: ${
        installResult.stderr || "Unknown error"
      }`,
    };
  }

  logger.info(`Node.js ${version} installed successfully`);

  // Set the installed version as default
  logger.debug(`Setting Node.js ${version} as default...`);
  const aliasResult = await runCommand(
    "bash",
    ["-c", `source "${nvmScript}" && nvm alias default ${version}`],
    {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    },
  );

  if (aliasResult.code !== 0) {
    logger.warn(`Failed to set Node.js ${version} as default`);
  } else {
    logger.info(`Node.js ${version} set as default`);
  }

  return {
    ok: true,
    details: `Node.js ${version} installed via NVM`,
  };
}

/**
 * Log Windows installation guide for NVM
 */
function logWindowsNvmGuide(logger: any): void {
  logger.warn("Automatic NVM installation is not supported on Windows");
  logger.info("");
  logger.info("=== NVM for Windows Installation Guide ===");
  logger.info("");
  logger.info("1. Visit the nvm-windows repository:");
  logger.info("   https://github.com/coreybutler/nvm-windows");
  logger.info("");
  logger.info("2. Download the latest release:");
  logger.info("   - Click on 'Releases' on the right sidebar");
  logger.info("   - Download 'nvm-setup.exe' from the latest release");
  logger.info("");
  logger.info("3. Run the installer:");
  logger.info("   - Double-click the downloaded nvm-setup.exe");
  logger.info("   - Follow the installation wizard");
  logger.info("");
  logger.info("4. Verify installation:");
  logger.info("   - Open a new Command Prompt or PowerShell");
  logger.info("   - Run: nvm version");
  logger.info("");
  logger.info("5. Install Node.js:");
  logger.info(`   - Run: nvm install ${logger.version || "latest"}`);
  logger.info(`   - Run: nvm use ${logger.version || "latest"}`);
  logger.info("");
  logger.info("===========================================");
  logger.info("");
}

/**
 * Install global npm packages
 */
async function installGlobalNpmPackages(
  runtime: PluginRuntime<NodeOptions>,
  packages: string[],
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  if (packages.length === 0) {
    return { ok: true, details: "No global npm packages to install" };
  }

  logger.info(`Installing global npm packages: ${packages.join(", ")}`);

  const result = await runCommand("npm", ["install", "-g", ...packages], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: `Failed to install npm packages: ${result.stderr || "Unknown error"}`,
    };
  }

  return {
    ok: true,
    details: `Successfully installed npm packages: ${packages.join(", ")}`,
  };
}

/**
 * Install global yarn packages
 */
async function installGlobalYarnPackages(
  runtime: PluginRuntime<NodeOptions>,
  packages: string[],
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  if (packages.length === 0) {
    return { ok: true, details: "No global yarn packages to install" };
  }

  logger.info(`Installing global yarn packages: ${packages.join(", ")}`);

  const result = await runCommand("yarn", ["global", "add", ...packages], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: `Failed to install yarn packages: ${result.stderr || "Unknown error"}`,
    };
  }

  return {
    ok: true,
    details: `Successfully installed yarn packages: ${packages.join(", ")}`,
  };
}

/**
 * Install global pnpm packages
 */
async function installGlobalPnpmPackages(
  runtime: PluginRuntime<NodeOptions>,
  packages: string[],
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  if (packages.length === 0) {
    return { ok: true, details: "No global pnpm packages to install" };
  }

  logger.info(`Installing global pnpm packages: ${packages.join(", ")}`);

  const result = await runCommand("pnpm", ["add", "-g", ...packages], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: `Failed to install pnpm packages: ${result.stderr || "Unknown error"}`,
    };
  }

  return {
    ok: true,
    details: `Successfully installed pnpm packages: ${packages.join(", ")}`,
  };
}

/**
 * Install global bun packages
 */
async function installGlobalBunPackages(
  runtime: PluginRuntime<NodeOptions>,
  packages: string[],
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  if (packages.length === 0) {
    return { ok: true, details: "No global bun packages to install" };
  }

  logger.info(`Installing global bun packages: ${packages.join(", ")}`);

  const result = await runCommand("bun", ["add", "-g", ...packages], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: `Failed to install bun packages: ${result.stderr || "Unknown error"}`,
    };
  }

  return {
    ok: true,
    details: `Successfully installed bun packages: ${packages.join(", ")}`,
  };
}

async function detectNode(runtime: PluginRuntime<NodeOptions>) {
  const result = await runCommand("node", ["-v"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });
  if (result.code !== 0) {
    return {
      ok: false,
      details: "Node is not available on PATH",
    };
  }
  const version = parseNodeVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Node version could not be determined",
    };
  }
  if (version.startsWith(runtime.options.version)) {
    return {
      ok: true,
      details: `Detected Node ${version}`,
    };
  }
  return {
    ok: false,
    details: `Detected Node ${version} but ${runtime.options.version} is requested`,
  };
}

export function createPlugin(
  instance: GenesisPluginInstance<NodeOptions>,
): GenesisPlugin<NodeOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      return detectNode(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const { use_nvm, global_packages } = runtime.options;
      const platform = getPlatform();

      // Skip task registration for Windows (manual installation required)
      if (platform === "windows") {
        return;
      }

      // If using NVM, we need curl to download the NVM install script
      if (use_nvm) {
        logger.debug(
          "Registering system tasks for NVM installation prerequisites",
        );

        // Register package manager update (will be deduplicated across plugins)
        const updateTask = createPackageManagerUpdateTask(
          runtime.context.cwd,
          runtime.context.env,
        );
        taskRegistry.register(updateTask);

        // Register curl installation (will be deduplicated if other plugins need it)
        const curlTask = createPackageInstallTask(
          "curl",
          runtime.context.cwd,
          runtime.context.env,
        );
        taskRegistry.register(curlTask);

        logger.debug(
          "System tasks registered: package manager update, curl installation",
        );
      }

      // Register global package installation tasks if packages are specified
      if (global_packages && global_packages.length > 0) {
        logger.debug("Registering global package installation tasks");

        // Register package manager availability checks
        const packageManagers = ["npm", "yarn", "pnpm", "bun"];
        for (const pm of packageManagers) {
          const checkTask = createCommandCheckTask(
            pm,
            runtime.context.cwd,
            runtime.context.env,
          );
          taskRegistry.register(checkTask);
        }

        // Register global package installation task
        const installTask = createCustomTask(
          "node-global-packages",
          "Install global Node.js packages",
          async () => {
            const results = [];
            let hasErrors = false;

            // Try npm first
            const npmResult = await installGlobalNpmPackages(
              runtime,
              global_packages,
            );
            results.push(npmResult.details);
            if (!npmResult.ok) hasErrors = true;

            return {
              ok: !hasErrors,
              details: results.join("; "),
              error: hasErrors
                ? "Some package installations failed"
                : undefined,
            };
          },
          {
            priority: 10, // Low priority - run after other setup
            dependsOn: [
              "*:command-check:npm",
              "*:command-check:yarn",
              "*:command-check:pnpm",
              "*:command-check:bun",
            ],
          },
        );

        taskRegistry.register(installTask);
        logger.debug("Global package installation tasks registered");
      }
    },
    async apply(runtime) {
      const { logger } = runtime.context;
      const { use_nvm } = runtime.options;
      const platform = getPlatform();

      // Check if Node is already installed and correct version
      const detectResult = await detectNode(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Node.js is already installed");
        return {
          ok: true,
          didChange: false,
          details: detectResult.details,
        };
      }

      // Handle Windows separately
      if (platform === "windows") {
        if (use_nvm) {
          logWindowsNvmGuide({ ...logger, version: runtime.options.version });
          return {
            ok: false,
            didChange: false,
            details:
              "Automatic installation not supported on Windows. See installation guide in logs.",
          };
        } else {
          logger.warn(
            "Standalone Node.js installation on Windows is not yet supported",
          );
          logger.info(
            "Please download and install Node.js manually from https://nodejs.org/",
          );
          return {
            ok: false,
            didChange: false,
            details: "Manual installation required. Visit https://nodejs.org/",
          };
        }
      }

      // macOS/Linux installation
      if (use_nvm) {
        logger.info("Installing Node.js via NVM...");

        // Check if NVM is installed
        const nvmInstalled = await isNvmInstalled(runtime);

        if (!nvmInstalled) {
          logger.debug("NVM not found, installing...");
          const installNvmResult = await installNvm(runtime);

          if (!installNvmResult.ok) {
            logger.error("Failed to install NVM");
            return {
              ok: false,
              didChange: false,
              details: installNvmResult.details,
            };
          }
        } else {
          logger.debug("NVM is already installed");
        }

        // Install Node via NVM
        const installNodeResult = await installNodeViaNvm(runtime);

        if (!installNodeResult.ok) {
          logger.error("Failed to install Node.js via NVM");
          return {
            ok: false,
            didChange: true, // NVM might have been installed
            details: installNodeResult.details,
          };
        }

        logger.info("Node.js installation completed successfully");
        return {
          ok: true,
          didChange: true,
          details: installNodeResult.details,
        };
      } else {
        // Standalone installation (not yet implemented)
        logger.warn("Standalone Node.js installation is not yet supported");
        logger.info(
          "Please install Node.js manually or set use_nvm: true in your config",
        );
        return {
          ok: false,
          didChange: false,
          details:
            "Standalone installation not supported. Use use_nvm: true or install manually.",
        };
      }
    },
    async validate(runtime) {
      const detectResult = await detectNode(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
