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

export interface DockerOptions {
  version?: string;
  include_compose?: boolean;
  install_desktop?: boolean;
}

export function docker(
  options: DockerOptions = {},
): GenesisPluginInstance<DockerOptions> {
  return {
    id: "docker",
    category: "tool",
    module: "@ossl/genesis-plugins/docker",
    options: {
      version: options.version ?? "latest",
      include_compose: options.include_compose ?? true,
      install_desktop: options.install_desktop ?? false,
    },
  };
}

/**
 * Parse Docker version from command output
 */
function parseDockerVersion(output: string): string | undefined {
  const match = output.match(/Docker version (\d+\.\d+\.\d+)/);
  return match?.[1];
}

/**
 * Parse Docker Compose version from command output
 */
function parseDockerComposeVersion(output: string): string | undefined {
  const match = output.match(/Docker Compose version v?(\d+\.\d+\.\d+)/);
  return match?.[1];
}

/**
 * Check if Docker is available and get version
 */
async function detectDocker(runtime: PluginRuntime<DockerOptions>) {
  const result = await runCommand("docker", ["--version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Docker is not available on PATH",
    };
  }

  const version = parseDockerVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Docker version could not be determined",
    };
  }

  // If specific version requested, check if it matches
  if (runtime.options.version && runtime.options.version !== "latest") {
    if (version.startsWith(runtime.options.version)) {
      return {
        ok: true,
        details: `Detected Docker ${version}`,
      };
    }
    return {
      ok: false,
      details: `Detected Docker ${version} but ${runtime.options.version} is requested`,
    };
  }

  return {
    ok: true,
    details: `Detected Docker ${version}`,
  };
}

/**
 * Check if Docker Compose is available
 */
async function detectDockerCompose(runtime: PluginRuntime<DockerOptions>) {
  const result = await runCommand("docker-compose", ["--version"], {
    cwd: runtime.context.cwd,
    env: runtime.context.env,
  });

  if (result.code !== 0) {
    return {
      ok: false,
      details: "Docker Compose is not available on PATH",
    };
  }

  const version = parseDockerComposeVersion(result.stdout || result.stderr);
  if (!version) {
    return {
      ok: false,
      details: "Docker Compose version could not be determined",
    };
  }

  return {
    ok: true,
    details: `Detected Docker Compose ${version}`,
  };
}

/**
 * Install Docker Engine on Linux using official repository
 */
async function installDockerLinux(
  runtime: PluginRuntime<DockerOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;

  logger.info("Installing Docker Engine on Linux...");

  try {
    // Detect Linux distribution
    const osRelease = await fs.promises.readFile("/etc/os-release", "utf8");
    const isUbuntu = osRelease.includes("ubuntu");
    const isDebian = osRelease.includes("debian");
    const isFedora = osRelease.includes("fedora");
    const isCentOS = osRelease.includes("centos") || osRelease.includes("rhel");

    if (isUbuntu || isDebian) {
      // Ubuntu/Debian installation
      logger.debug("Installing Docker on Ubuntu/Debian");

      const commands = [
        // Install prerequisites
        "sudo apt-get update",
        "sudo apt-get install -y ca-certificates curl gnupg",
        // Add Docker's official GPG key
        "sudo install -m 0755 -d /etc/apt/keyrings",
        "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg",
        "sudo chmod a+r /etc/apt/keyrings/docker.gpg",
        // Add Docker repository
        'echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null',
        // Install Docker Engine
        "sudo apt-get update",
        "sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin",
        // Add user to docker group
        "sudo usermod -aG docker $USER",
      ];

      for (const cmd of commands) {
        const [command, ...args] = cmd.split(" ");
        const result = await runCommand(command, args);

        if (result.code !== 0 && !cmd.includes("usermod")) {
          // usermod might fail if user is already in group, that's ok
          throw new Error(`Failed to execute ${cmd}: ${result.stderr}`);
        }
      }
    } else if (isFedora || isCentOS) {
      // Fedora/CentOS installation
      logger.debug("Installing Docker on Fedora/CentOS");

      const commands = [
        // Install prerequisites
        "sudo dnf update -y",
        "sudo dnf install -y dnf-plugins-core",
        // Add Docker repository
        "sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo",
        // Install Docker Engine
        "sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin",
        // Start and enable Docker
        "sudo systemctl start docker",
        "sudo systemctl enable docker",
        // Add user to docker group
        "sudo usermod -aG docker $USER",
      ];

      for (const cmd of commands) {
        const [command, ...args] = cmd.split(" ");
        const result = await runCommand(command, args);

        if (result.code !== 0 && !cmd.includes("usermod")) {
          throw new Error(`Failed to execute ${cmd}: ${result.stderr}`);
        }
      }
    } else {
      throw new Error(
        "Unsupported Linux distribution for automatic Docker installation",
      );
    }

    logger.info("Docker Engine installed successfully");
    logger.info(
      "You may need to log out and log back in for group changes to take effect",
    );

    return {
      ok: true,
      details: "Docker Engine installed on Linux",
    };
  } catch (error) {
    logger.error(`Failed to install Docker on Linux: ${error}`);
    return {
      ok: false,
      details: `Docker Linux installation failed: ${error}`,
    };
  }
}

/**
 * Install Docker Desktop on macOS
 */
async function installDockerMacOS(
  runtime: PluginRuntime<DockerOptions>,
): Promise<{ ok: boolean; details: string }> {
  const { logger } = runtime.context;
  const { install_desktop } = runtime.options;

  if (install_desktop) {
    logger.info("Installing Docker Desktop on macOS...");

    try {
      // Download Docker Desktop
      const downloadUrl =
        "https://desktop.docker.com/mac/main/amd64/Docker.dmg";
      const tempDir = os.tmpdir();
      const dmgPath = path.join(tempDir, "Docker.dmg");

      logger.debug(`Downloading Docker Desktop from ${downloadUrl}`);
      const downloadResult = await runCommand("curl", [
        "-L",
        "-o",
        dmgPath,
        downloadUrl,
      ]);

      if (downloadResult.code !== 0) {
        throw new Error(
          `Failed to download Docker Desktop: ${downloadResult.stderr}`,
        );
      }

      logger.info("Docker Desktop downloaded successfully");
      logger.info(
        "Please run the following commands to complete installation:",
      );
      logger.info(`1. Open the downloaded file: open ${dmgPath}`);
      logger.info("2. Drag Docker to Applications folder");
      logger.info("3. Launch Docker from Applications");
      logger.info("4. Follow the setup wizard");

      // Clean up
      await fs.promises.unlink(dmgPath);

      return {
        ok: true,
        details: "Docker Desktop downloaded. Manual installation required.",
      };
    } catch (error) {
      logger.error(`Failed to download Docker Desktop: ${error}`);
      return {
        ok: false,
        details: `Docker Desktop download failed: ${error}`,
      };
    }
  } else {
    // Install Docker Engine using Colima or similar
    logger.info("Installing Docker Engine on macOS using Colima...");

    try {
      // Install Colima (Docker Desktop alternative)
      const commands = ["brew install colima", "colima start"];

      for (const cmd of commands) {
        const [command, ...args] = cmd.split(" ");
        const result = await runCommand(command, args);

        if (result.code !== 0) {
          throw new Error(`Failed to execute ${cmd}: ${result.stderr}`);
        }
      }

      logger.info("Docker Engine installed via Colima");

      return {
        ok: true,
        details: "Docker Engine installed on macOS via Colima",
      };
    } catch (error) {
      logger.error(`Failed to install Docker via Colima: ${error}`);
      return {
        ok: false,
        details: `Docker Colima installation failed: ${error}`,
      };
    }
  }
}

/**
 * Log Windows installation guide for Docker
 */
function logWindowsDockerGuide(logger: any, installDesktop: boolean): void {
  logger.warn("Automatic Docker installation on Windows requires manual steps");
  logger.info("");
  logger.info("=== Docker for Windows Installation Guide ===");
  logger.info("");

  if (installDesktop) {
    logger.info("Docker Desktop Installation:");
    logger.info("1. Visit the Docker downloads page:");
    logger.info("   https://www.docker.com/products/docker-desktop/");
    logger.info("");
    logger.info("2. Download Docker Desktop for Windows");
    logger.info("   - Click 'Download for Windows'");
    logger.info("   - Requires Windows 10/11 Pro, Enterprise, or Education");
    logger.info("");
    logger.info("3. Run the installer:");
    logger.info("   - Double-click the downloaded .exe file");
    logger.info("   - Follow the installation wizard");
    logger.info("   - Enable WSL 2 when prompted");
    logger.info("");
  } else {
    logger.info("Docker Engine via WSL 2:");
    logger.info("1. Install WSL 2:");
    logger.info("   wsl --install");
    logger.info("");
    logger.info("2. Install Ubuntu from Microsoft Store");
    logger.info("");
    logger.info("3. Install Docker in WSL:");
    logger.info("   wsl -e bash -c 'curl -fsSL https://get.docker.com | sh'");
    logger.info("");
  }

  logger.info("4. Verify installation:");
  logger.info("   docker --version");
  logger.info("   docker run hello-world");
  logger.info("");
  logger.info("===========================================");
  logger.info("");
}

export function createPlugin(
  instance: GenesisPluginInstance<DockerOptions>,
): GenesisPlugin<DockerOptions> {
  return {
    id: instance.id,
    category: instance.category,
    async detect(runtime) {
      const dockerResult = await detectDocker(runtime);

      if (!dockerResult.ok) {
        return dockerResult;
      }

      // If Docker Compose is requested, check it too
      if (runtime.options.include_compose) {
        const composeResult = await detectDockerCompose(runtime);
        if (!composeResult.ok) {
          return {
            ok: false,
            details: `${dockerResult.details} but ${composeResult.details}`,
          };
        }
        return {
          ok: true,
          details: `${dockerResult.details} and ${composeResult.details}`,
        };
      }

      return dockerResult;
    },
    async registerTasks(runtime) {
      const { taskRegistry, logger } = runtime.context;
      const platform = getPlatform();

      // Skip task registration for Windows (manual installation)
      if (platform === "windows") {
        return;
      }

      logger.debug("Registering system tasks for Docker installation");

      // Register package manager update (will be deduplicated across plugins)
      const updateTask = createPackageManagerUpdateTask(
        runtime.context.cwd,
        runtime.context.env,
      );
      taskRegistry.register(updateTask);

      if (platform === "macos") {
        // macOS needs Homebrew for Colima
        if (!runtime.options.install_desktop) {
          const brewTask = createPackageInstallTask(
            "colima",
            runtime.context.cwd,
            runtime.context.env,
          );
          taskRegistry.register(brewTask);
        }
      } else {
        // Linux needs various packages
        const packages = ["curl", "ca-certificates", "gnupg"];
        for (const pkg of packages) {
          const task = createPackageInstallTask(
            pkg,
            runtime.context.cwd,
            runtime.context.env,
          );
          taskRegistry.register(task);
        }
      }

      logger.debug(
        "System tasks registered: package manager update, Docker dependencies",
      );
    },
    async apply(runtime) {
      const { logger } = runtime.context;
      const { install_desktop } = runtime.options;
      const platform = getPlatform();

      // Check if Docker is already installed
      const detectResult = await this.detect!(runtime);
      if (detectResult.ok) {
        logger.info(detectResult.details || "Docker is already installed");
        return {
          ok: true,
          didChange: false,
          details: detectResult.details,
        };
      }

      // Handle Windows separately
      if (platform === "windows") {
        logWindowsDockerGuide(logger, install_desktop ?? false);
        return {
          ok: false,
          didChange: false,
          details:
            "Automatic installation not supported on Windows. See installation guide in logs.",
        };
      }

      // macOS/Linux installation
      let installResult: { ok: boolean; details: string };

      if (platform === "macos") {
        installResult = await installDockerMacOS(runtime);
      } else {
        installResult = await installDockerLinux(runtime);
      }

      if (!installResult.ok) {
        logger.error("Failed to install Docker");
        return {
          ok: false,
          didChange: false,
          details: installResult.details,
        };
      }

      // Test Docker installation
      logger.info("Testing Docker installation...");

      try {
        const testResult = await runCommand("docker", ["run", "hello-world"], {
          cwd: runtime.context.cwd,
          env: runtime.context.env,
        });

        if (testResult.code === 0) {
          logger.info("Docker installation verified successfully");
        } else {
          logger.warn("Docker installation completed but test failed");
          logger.debug(`Test error: ${testResult.stderr}`);
        }
      } catch (error) {
        logger.warn(`Could not test Docker installation: ${error}`);
      }

      return {
        ok: true,
        didChange: true,
        details: installResult.details,
      };
    },
    async validate(runtime) {
      const detectResult = await this.detect!(runtime);
      return {
        ok: detectResult.ok,
        message: detectResult.details,
      };
    },
  };
}
