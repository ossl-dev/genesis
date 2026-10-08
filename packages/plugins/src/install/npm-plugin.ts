import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { getPlatform, runCommand, type GenesisPlugin, type GenesisPluginInstance, type PluginRuntime } from "@ossl/genesis-core";
import { matchesVersion } from "../options.js";
import { InstallationRecoveryError, installDirectory, prependPath } from "./archive.js";

export interface PackageManagerOptions {
  version: string;
  install_dir?: string;
  node_plugin?: string | null;
}

export function createNpmPlugin(instance: GenesisPluginInstance<PackageManagerOptions>, name: "pnpm" | "yarn", parseOptions: (value: unknown) => PackageManagerOptions): GenesisPlugin<PackageManagerOptions> {
  const directory = (runtime: PluginRuntime<PackageManagerOptions>) => runtime.options.install_dir ?? path.join((getPlatform() === "windows" ? runtime.context.env.USERPROFILE : runtime.context.env.HOME) ?? os.homedir(), ".genesis", name);
  const binDirectory = (root: string) => getPlatform() === "windows" ? root : path.join(root, "bin");
  const executable = (root: string) => path.join(binDirectory(root), getPlatform() === "windows" ? `${name}.cmd` : name);
  async function check(runtime: PluginRuntime<PackageManagerOptions>, command: string) {
    const result = await runCommand(command, ["--version"], { cwd: runtime.context.cwd, env: { ...runtime.context.env, YARN_IGNORE_PATH: "1", COREPACK_ENABLE_PROJECT_SPEC: "0" } });
    const version = result.stdout.trim().match(/^(\d+\.\d+\.\d+)$/)?.[1];
    return { ok: result.code === 0 && !!version && matchesVersion(version, runtime.options.version), details: version ? `Detected ${name} ${version}; requested ${runtime.options.version}` : `${name} is unavailable: ${result.stderr}` };
  }
  async function detect(runtime: PluginRuntime<PackageManagerOptions>) {
    const root = directory(runtime);
    const managed = fs.existsSync(executable(root));
    const result = await check(runtime, managed || runtime.options.install_dir ? executable(root) : name);
    if (result.ok && managed) prependPath(runtime.context.env, binDirectory(root));
    return result;
  }
  const dependency = instance.options?.node_plugin;
  return {
    id: instance.id, category: instance.category, parseOptions, dependsOn: dependency === null ? [] : [dependency ?? "node"], detect,
    async apply(runtime) {
      const detected = await detect(runtime);
      if (detected.ok) return { ok: true, didChange: false, details: detected.details };
      try {
        const options = { cwd: runtime.context.cwd, env: runtime.context.env };
        const npm = await runCommand("npm", ["--version"], options);
        if (npm.code !== 0) throw new Error("npm is required; configure a Node plugin or provide Node/npm on PATH");
        const packageName = name === "yarn" && Number(runtime.options.version.split(".")[0]) >= 2 ? "@yarnpkg/cli-dist" : name;
        await installDirectory({
          destination: directory(runtime), context: runtime.context,
          async prepare(stage) {
            const prefix = path.join(stage, "prefix");
            const installed = await runCommand("npm", ["install", "--global", "--prefix", prefix, "--engine-strict", "--ignore-scripts", "--no-audit", "--no-fund", `${packageName}@${runtime.options.version}`], options);
            if (installed.code !== 0) throw new Error(installed.stderr || installed.stdout || "npm installation failed");
            return prefix;
          },
          async verify(root) {
            const result = await check(runtime, executable(root));
            if (!result.ok) throw new Error(result.details);
          },
        });
        prependPath(runtime.context.env, binDirectory(directory(runtime)));
        runtime.context.logger.info(`Add ${binDirectory(directory(runtime))} to your shell PATH for future sessions.`);
        return { ok: true, didChange: true, details: `${name} ${runtime.options.version} installed to ${directory(runtime)}` };
      } catch (error) {
        return { ok: false, didChange: error instanceof InstallationRecoveryError, details: `${name} installation failed: ${error instanceof Error ? error.message : error}` };
      }
    },
    async validate(runtime) { const result = await detect(runtime); return { ok: result.ok, message: result.details }; },
  };
}
