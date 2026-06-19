import {
  type GenesisPlugin,
  type GenesisPluginInstance,
  type PluginRuntime,
  runCommand,
  getPlatform,
  createPackageManagerUpdateTask,
  createPackageInstallTask,
} from "@ossl/genesis-core";

export interface PythonOptions {
  version: string;
}

export function python(
  options: PythonOptions
): GenesisPluginInstance<PythonOptions> {
  return {
    id: "python",
    category: "language",
    module: "@ossl/genesis-plugins/python",
    options,
  };
}

/**
 * Parse Python version from command output
 */
function parsePythonVersion(output: string): string | undefined {
  const match = output.match(/Python (\d+\.\d+\.\d+)/);
  return match?.[1];
}

async function detectPython(runtime: PluginRuntime<PythonOptions>) {
  const result = await runCommand("python3", ["--version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Python is not available on PATH",
    };
  }

  const version = parsePythonVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Python version could not be determined",
    };
  }

  const requestedMajorMinor = runtime.options.version
    .split(".")
    .slice(0, 2)
    .join(".");
  const installedMajorMinor = version.split(".").slice(0, 2).join(".");

  if (installedMajorMinor === requestedMajorMinor) {
    return {
      ok: true,
      details: `Detected Python ${version}`,
    };
  }

  return {
    ok: false,
    details: `Detected Python ${version} but ${runtime.options.version} is requested`,
  };
}

export function createPlugin(
  instance: GenesisPluginInstance<PythonOptions>
): GenesisPlugin<PythonOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      return detectPython(runtime);
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const platform = getPlatform();

      // Skip task registration for Windows (manual installation required)
      if (platform === "windows") {
        return;
      }

      logger.debug("Registering system tasks for Python installation");

      // Register package manager update (will be deduplicated across plugins)
      const updateTask = createPackageManagerUpdateTask(
        runtime.context.cwd,
        runtime.context.env
      );
      taskRegistry.register(updateTask);

      // Determine package name based on platform
      let packageName: string;
      if (platform === "macos") {
        packageName = `python@${runtime.options.version.split(".")[0]}`;
      } else {
        // Linux
        packageName = `python${runtime.options.version
          .split(".")
          .slice(0, 2)
          .join(".")}`;
      }

      // Register Python installation
      const installTask = createPackageInstallTask(
        packageName,
        runtime.context.cwd,
        runtime.context.env
      );
      taskRegistry.register(installTask);

      logger.debug(
        `System tasks registered: package manager update, ${packageName} installation`
      );
    },
    async apply(runtime) {
      const { logger } = runtime.context;
      const platform = getPlatform();

      // Check if Python is already installed and correct version
      const detectResult = await detectPython(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Python is already installed");
        return {
          ok: true,
          didChange: false,
          details: detectResult.details,
        };
      }

      // Handle Windows separately
      if (platform === "windows") {
        logger.warn(
          "Automatic Python installation on Windows is not yet supported"
        );
        logger.info(
          "Please download and install Python manually from https://www.python.org/"
        );
        return {
          ok: false,
          didChange: false,
          details:
            "Manual installation required. Visit https://www.python.org/",
        };
      }

      // macOS/Linux installation using package manager
      // System tasks have already been executed in Phase 2
      logger.info(
        `Python ${runtime.options.version} should now be installed via system package manager`
      );

      return {
        ok: true,
        didChange: true,
        details: `Python ${runtime.options.version} installation completed`,
      };
    },
    async validate(runtime) {
      const detectResult = await detectPython(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
