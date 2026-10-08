import { type Task, type TaskId } from "./task-registry.js";
import { runCommand } from "../os/shell.js";
import { getPlatform } from "../os/platform.js";
import { getLinuxDistribution } from "../os/linux.js";

type PackageManager = "brew" | "apt" | "dnf" | "pacman" | "apk" | "choco";

function getPackageManager(): PackageManager {
  const platform = getPlatform();
  if (platform === "macos") return "brew";
  if (platform === "windows") return "choco";
  const { id, like } = getLinuxDistribution();
  for (const candidate of [id, ...like]) {
    if (["debian", "ubuntu"].includes(candidate)) return "apt";
    if (["fedora", "rhel", "centos", "rocky", "almalinux"].includes(candidate)) return "dnf";
    if (candidate === "arch") return "pacman";
    if (candidate === "alpine") return "apk";
  }
  throw new Error(`Unsupported Linux package manager for ${id}`);
}

function taskId(manager: PackageManager, operation: "update" | "install", packageName?: string): TaskId {
  return `${getPlatform()}:package-manager:${manager}-${operation}${operation === "install" ? `:${packageName}` : ""}`;
}

export function createPackageManagerTaskId(operation: "update" | "install", packageName?: string): TaskId {
  return taskId(getPackageManager(), operation, packageName);
}

const linuxPackages: Record<string, Partial<Record<PackageManager, string[]>>> = {
  "build-essential": { dnf: ["gcc", "gcc-c++", "make"], pacman: ["base-devel"], apk: ["build-base"] },
  "libssl-dev": { dnf: ["openssl-devel"], pacman: ["openssl"], apk: ["openssl-dev"] },
  "libcurl4-openssl-dev": { dnf: ["libcurl-devel"], pacman: ["curl"], apk: ["curl-dev"] },
  "zlib1g-dev": { dnf: ["zlib-devel"], pacman: ["zlib"], apk: ["zlib-dev"] },
  "libexpat1-dev": { dnf: ["expat-devel"], pacman: ["expat"], apk: ["expat-dev"] },
  gnupg: { dnf: ["gnupg2"] },
};

function commandTask(id: TaskId, description: string, command: string, args: string[], cwd: string, env: NodeJS.ProcessEnv, dependsOn?: TaskId[]): Task {
  const linux = getPlatform() === "linux";
  const needsSudo = linux && process.getuid?.() !== 0;
  return {
    id, description, priority: dependsOn ? 50 : 100, dependsOn,
    async executor() {
      try {
        const result = await runCommand(needsSudo ? "sudo" : command, needsSudo ? [command, ...args] : args, { cwd, env });
        return result.code === 0
          ? { ok: true, details: description }
          : { ok: false, error: result.stderr || `${description} failed` };
      } catch (error) {
        return { ok: false, error: error instanceof Error ? error.message : String(error) };
      }
    },
  };
}

export function createPackageManagerUpdateTask(cwd: string, env: NodeJS.ProcessEnv): Task {
  const manager = getPackageManager();
  const id = taskId(manager, "update");
  // Neither manager has a safe cache-only refresh: pacman -Sy permits partial
  // upgrades, and Chocolatey resolves metadata during each install.
  if (manager === "pacman" || manager === "choco") {
    return { id, description: `Use existing ${manager} package index`, priority: 100,
      executor: async () => ({ ok: true, details: manager === "pacman" ? "Using existing pacman database; maintain the host with pacman -Syu before provisioning" : "Chocolatey resolves packages during installation" }) };
  }
  const command = manager === "apt" ? "apt-get" : manager;
  return commandTask(id, `Update ${manager} package index`, command, [manager === "dnf" ? "makecache" : "update"], cwd, env);
}

export function createPackageInstallTask(packageName: string, cwd: string, env: NodeJS.ProcessEnv): Task {
  if (!/^[a-zA-Z0-9][a-zA-Z0-9@+_.:/-]*$/.test(packageName)) throw new Error(`Invalid system package name: ${packageName}`);
  const manager = getPackageManager();
  const packages = linuxPackages[packageName]?.[manager] ?? [packageName];
  const args = manager === "pacman" ? ["-S", "--needed", "--noconfirm", ...packages]
    : manager === "apk" ? ["add", ...packages]
    : manager === "brew" ? ["install", ...packages]
    : manager === "choco" ? ["install", ...packages, "-y"]
    : ["install", "-y", ...packages];
  return commandTask(taskId(manager, "install", packageName), `Install ${packages.join(", ")} via ${manager}`, manager === "apt" ? "apt-get" : manager, args, cwd, env, [taskId(manager, "update")]);
}

/**
 * Create a task for checking if a command exists
 */
export function createCommandCheckTask(
  commandName: string,
  cwd: string,
  env: NodeJS.ProcessEnv
): Task {
  const platform = getPlatform();
  const taskId = `${platform}:command-check:${commandName}`;

  return {
    id: taskId,
    description: `Check if ${commandName} is available`,
    priority: 200, // Very high priority - checks should run first
    executor: async () => {
      try {
        const checkCommand = platform === "windows" ? "where" : "which";
        const result = await runCommand(checkCommand, [commandName], { cwd, env });
        
        if (result.code === 0) {
          return {
            ok: true,
            details: `${commandName} is available`,
          };
        }

        return {
          ok: false,
          details: `${commandName} is not available`,
        };
      } catch (error) {
        return {
          ok: false,
          error: (error as Error).message,
        };
      }
    },
  };
}

/**
 * Create a custom task
 */
export function createCustomTask(
  id: string,
  description: string,
  executor: () => Promise<{ ok: boolean; details?: string; error?: string }>,
  options?: {
    priority?: number;
    dependsOn?: TaskId[];
  }
): Task {
  const platform = getPlatform();
  const taskId = `${platform}:custom:${id}`;

  return {
    id: taskId,
    description,
    priority: options?.priority ?? 0,
    dependsOn: options?.dependsOn,
    executor,
  };
}

/**
 * Helper to ensure package manager is updated
 * Returns the task ID for the update operation
 */
export function ensurePackageManagerUpdate(): TaskId {
  return createPackageManagerTaskId("update");
}

/**
 * Helper to ensure a package is installed
 * Returns the task ID for the install operation
 */
export function ensurePackageInstalled(packageName: string): TaskId {
  return createPackageManagerTaskId("install", packageName);
}

