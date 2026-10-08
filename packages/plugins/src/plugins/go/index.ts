import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { type GenesisPlugin, type GenesisPluginInstance, type PluginRuntime, runCommand, getPlatform } from "@ossl/genesis-core";
import { optionSchemas, matchesVersion } from "../../options.js";
import { InstallationRecoveryError, installArchive, prependPath, singleDirectory } from "../../install/archive.js";
import { goRelease } from "../../install/releases.js";

export interface GoOptions {
  version: string;
  install_dir?: string;
}

export function go(options: GoOptions): GenesisPluginInstance<GoOptions> {
  return { id: "go", category: "language", module: "@ossl/genesis-plugins/go", options };
}

function goDirectory(runtime: PluginRuntime<GoOptions>): string {
  return runtime.options.install_dir ?? (getPlatform() === "windows" ? path.join(runtime.context.env.USERPROFILE ?? os.homedir(), ".genesis", "go") : "/usr/local/go");
}

async function checkGo(runtime: PluginRuntime<GoOptions>, executable: string) {
  const result = await runCommand(executable, ["version"], { cwd: runtime.context.cwd, env: runtime.context.env });
  if (result.code !== 0) return { ok: false, details: "Go is not available on PATH or in its installation directory" };
  const version = (result.stdout || result.stderr).match(/go version go(\d+\.\d+(?:\.\d+)?)/)?.[1];
  if (!version) return { ok: false, details: "Go version could not be determined" };
  return matchesVersion(version, runtime.options.version)
    ? { ok: true, details: `Detected Go ${version}` }
    : { ok: false, details: `Detected Go ${version} but ${runtime.options.version} is requested` };
}

async function detectGo(runtime: PluginRuntime<GoOptions>) {
  const directory = goDirectory(runtime);
  const executable = path.join(directory, "bin", getPlatform() === "windows" ? "go.exe" : "go");
  const managed = fs.existsSync(executable);
  const result = await checkGo(runtime, managed || runtime.options.install_dir ? executable : "go");
  if (result.ok && managed) {
    runtime.context.env.GOROOT = directory;
    prependPath(runtime.context.env, path.join(directory, "bin"));
  }
  return result;
}

export function createPlugin(instance: GenesisPluginInstance<GoOptions>): GenesisPlugin<GoOptions> {
  return {
    id: instance.id,
    category: instance.category,
    parseOptions: options => optionSchemas.go.parse(options),
    detect: detectGo,
    async apply(runtime) {
      const detected = await detectGo(runtime);
      if (detected.ok) return { ok: true, didChange: false, details: detected.details };
      try {
        const platform = getPlatform();
        const destination = goDirectory(runtime);
        const release = await goRelease(runtime.options.version, platform);
        await installArchive({
          release, destination, executable: path.join("bin", platform === "windows" ? "go.exe" : "go"), context: runtime.context, select: singleDirectory,
          async verify(root) {
            const result = await checkGo(runtime, path.join(root, "bin", platform === "windows" ? "go.exe" : "go"));
            if (!result.ok) throw new Error(result.details);
          },
        });
        runtime.context.env.GOROOT = destination;
        prependPath(runtime.context.env, path.join(destination, "bin"));
        runtime.context.logger.info(`Add ${path.join(destination, "bin")} to your shell PATH for future sessions.`);
        return { ok: true, didChange: true, details: `Go ${runtime.options.version} installed to ${destination}` };
      } catch (error) {
        return { ok: false, didChange: error instanceof InstallationRecoveryError, details: `Go installation failed: ${error instanceof Error ? error.message : error}` };
      }
    },
    async validate(runtime) {
      const result = await detectGo(runtime);
      return { ok: result.ok, message: result.details };
    },
  };
}
