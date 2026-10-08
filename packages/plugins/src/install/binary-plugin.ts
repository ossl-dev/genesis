import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { getPlatform, runCommand, type GenesisPlugin, type GenesisPluginInstance, type PluginRuntime, type Platform } from "@ossl/genesis-core";
import { matchesVersion } from "../options.js";
import { InstallationRecoveryError, installArchive, prependPath, singleDirectory, type ArchiveRelease } from "./archive.js";

export interface BinaryOptions {
  version: string;
  install_dir?: string;
}

export function createBinaryPlugin<T extends BinaryOptions>(instance: GenesisPluginInstance<T>, spec: {
  name: "bun" | "deno";
  parseOptions(options: unknown): T;
  release(options: T, platform: Platform): Promise<ArchiveRelease>;
}): GenesisPlugin<T> {
  const directory = (runtime: PluginRuntime<T>) => runtime.options.install_dir ?? path.join((getPlatform() === "windows" ? runtime.context.env.USERPROFILE : runtime.context.env.HOME) ?? os.homedir(), ".genesis", spec.name);
  const filename = () => `${spec.name}${getPlatform() === "windows" ? ".exe" : ""}`;
  const expose = (runtime: PluginRuntime<T>) => {
    const root = directory(runtime);
    runtime.context.env[`${spec.name.toUpperCase()}_INSTALL`] = root;
    prependPath(runtime.context.env, path.join(root, "bin"));
  };
  async function check(runtime: PluginRuntime<T>, command: string) {
    const result = await runCommand(command, ["--version"], { cwd: runtime.context.cwd, env: runtime.context.env });
    if (result.code !== 0) return { ok: false, details: `${spec.name} is unavailable: ${result.stderr}` };
    const version = result.stdout.trim().match(spec.name === "deno" ? /^deno (\d+\.\d+\.\d+)(?:\s|$)/ : /^(\d+\.\d+\.\d+)(?:\s|$)/)?.[1];
    if (!version) return { ok: false, details: `${spec.name} version could not be determined` };
    return { ok: matchesVersion(version, runtime.options.version), details: `Detected ${spec.name} ${version}; requested ${runtime.options.version}` };
  }
  async function detect(runtime: PluginRuntime<T>) {
    const executable = path.join(directory(runtime), "bin", filename());
    const managed = fs.existsSync(executable);
    const result = await check(runtime, managed || runtime.options.install_dir ? executable : spec.name);
    if (result.ok && managed) expose(runtime);
    return result;
  }
  return {
    id: instance.id, category: instance.category, parseOptions: spec.parseOptions, detect,
    async apply(runtime) {
      const found = await detect(runtime);
      if (found.ok) return { ok: true, didChange: false, details: found.details };
      try {
        const release = await spec.release(runtime.options, getPlatform());
        const destination = directory(runtime);
        await installArchive({
          release, destination, executable: path.join("bin", filename()), context: runtime.context,
          async select(root) {
            const source = spec.name === "bun" ? await singleDirectory(root) : root;
            const runtimeRoot = path.join(root, "runtime");
            await fs.promises.mkdir(path.join(runtimeRoot, "bin"), { recursive: true });
            const executable = path.join(runtimeRoot, "bin", filename());
            await fs.promises.rename(path.join(source, filename()), executable);
            if (getPlatform() !== "windows") await fs.promises.chmod(executable, 0o755);
            return runtimeRoot;
          },
          async verify(root) {
            const result = await check(runtime, path.join(root, "bin", filename()));
            if (!result.ok) throw new Error(result.details);
          },
        });
        expose(runtime);
        runtime.context.logger.info(`Add ${path.join(destination, "bin")} to your shell PATH for future sessions.`);
        return { ok: true, didChange: true, details: `${spec.name} ${runtime.options.version} installed to ${destination}` };
      } catch (error) {
        return { ok: false, didChange: error instanceof InstallationRecoveryError, details: `${spec.name} installation failed: ${error instanceof Error ? error.message : error}` };
      }
    },
    async validate(runtime) { const result = await detect(runtime); return { ok: result.ok, message: result.details }; },
  };
}
