import { optionSchemas, matchesVersion } from "../../options.js";
import { InstallationRecoveryError, installArchive, prependPath, singleDirectory } from "../../install/archive.js";
import { nodeRelease } from "../../install/releases.js";
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

export interface NodeOptions {
  version: string;
  use_nvm?: boolean;
  global_packages?: string[];
  install_dir?: string;
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
  return output.trim().match(/^v?(\d+\.\d+\.\d+)$/)?.[1];
}

/**
 * Get the NVM directory path
 */
function getNvmDir(env: NodeJS.ProcessEnv): string {
  if (env.NVM_DIR) {
    return env.NVM_DIR;
  }
  const home = env.HOME ?? os.homedir();
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
    ["-c", `set -o pipefail; curl -fsSL ${installUrl} | bash`],
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
    ["-c", 'source "$1" && nvm install "$2"', "genesis-nvm", nvmScript, version],
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
    ["-c", 'source "$1" && nvm alias default "$2"', "genesis-nvm", nvmScript, version],
    {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    },
  );

  if (aliasResult.code !== 0) {
    return { ok: false, details: `Failed to set Node.js ${version} as NVM default: ${aliasResult.stderr}` };
  } else {
    logger.info(`Node.js ${version} set as default`);
  }

  const resolved = await runCommand("bash", ["-c", 'source "$1" && nvm which "$2"', "genesis-nvm", nvmScript, version], {
    cwd: runtime.context.cwd, env: runtime.context.env,
  });
  const executable = resolved.stdout.trim();
  if (resolved.code !== 0 || !path.isAbsolute(executable)) {
    return { ok: false, details: `Node.js installed but its executable could not be resolved: ${resolved.stderr}` };
  }
  prependPath(runtime.context.env, path.dirname(executable));
  const verified = await checkNode(runtime, executable);
  if (!verified.ok) return verified;
  return {
    ok: true,
    details: `Node.js ${version} installed via NVM`,
  };
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

  const result = await runCommand("npm", ["install", "-g", "--", ...packages], {
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

async function checkNode(runtime: PluginRuntime<NodeOptions>, executable: string) {
  const result = await runCommand(executable, ["-v"], {
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
  if (matchesVersion(version, runtime.options.version)) {
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

function archiveDirectory(runtime: PluginRuntime<NodeOptions>): string {
  return runtime.options.install_dir ?? path.join((getPlatform() === "windows" ? runtime.context.env.USERPROFILE : runtime.context.env.HOME) ?? os.homedir(), ".genesis", "node");
}

function archiveBin(root: string): string {
  return getPlatform() === "windows" ? root : path.join(root, "bin");
}

function archiveExecutable(root: string): string {
  return path.join(archiveBin(root), getPlatform() === "windows" ? "node.exe" : "node");
}

async function detectNode(runtime: PluginRuntime<NodeOptions>) {
  if (runtime.options.use_nvm && !runtime.options.install_dir && getPlatform() !== "windows") return checkNode(runtime, "node");
  const root = archiveDirectory(runtime);
  const executable = archiveExecutable(root);
  const managed = fs.existsSync(executable);
  const result = await checkNode(runtime, managed || runtime.options.install_dir ? executable : "node");
  if (result.ok && managed) prependPath(runtime.context.env, archiveBin(root));
  return result;
}

async function installStandalone(runtime: PluginRuntime<NodeOptions>) {
  try {
    const platform = getPlatform();
    const destination = archiveDirectory(runtime);
    const release = await nodeRelease(runtime.options.version, platform);
    await installArchive({
      release, destination, executable: path.relative(destination, archiveExecutable(destination)), context: runtime.context, select: singleDirectory,
      async verify(root) {
        const result = await checkNode(runtime, archiveExecutable(root));
        if (!result.ok) throw new Error(result.details);
      },
    });
    prependPath(runtime.context.env, archiveBin(destination));
    runtime.context.logger.info(`Add ${archiveBin(destination)} to your shell PATH for future sessions.`);
    return { ok: true, didChange: true, details: `Node ${runtime.options.version} installed to ${destination}` };
  } catch (error) {
    return { ok: false, didChange: error instanceof InstallationRecoveryError, details: `Node installation failed: ${error instanceof Error ? error.message : error}` };
  }
}

export function createPlugin(
  instance: GenesisPluginInstance<NodeOptions>,
): GenesisPlugin<NodeOptions> {
  return {
    id: instance.id,
    category: instance.category,
    parseOptions: options => optionSchemas.node.parse(options),
    async detect(runtime) {
      return detectNode(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const { use_nvm } = runtime.options;
      const platform = getPlatform();

      // Windows uses a standalone archive.
      if (platform === "windows") {
        return;
      }

      // If using NVM, we need curl to download the NVM install script
      if (use_nvm && !runtime.options.install_dir) {
        if ((await this.detect!(runtime)).ok) return;

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

    },
    async postApply(runtime) {
      const packages = runtime.options.global_packages ?? [];
      if (!packages.length) return;
      const result = await installGlobalNpmPackages(runtime, packages);
      if (!result.ok) throw new Error(result.details);
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

      if (platform === "windows") return installStandalone(runtime);

      // macOS/Linux installation
      if (use_nvm && !runtime.options.install_dir) {
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
        return installStandalone(runtime);
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
