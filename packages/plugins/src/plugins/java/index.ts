import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { type GenesisPlugin, type GenesisPluginInstance, type PluginRuntime, runCommand, getPlatform } from "@ossl/genesis-core";
import { optionSchemas, matchesVersion } from "../../options.js";
import { InstallationRecoveryError, installArchive, prependPath, singleDirectory } from "../../install/archive.js";
import { javaRelease, normalizeJavaVersion } from "../../install/releases.js";

export interface JavaOptions {
  version: string;
  distribution?: "openjdk" | "oracle";
  install_dir?: string;
}

export function java(options: JavaOptions): GenesisPluginInstance<JavaOptions> {
  return { id: "java", category: "language", module: "@ossl/genesis-plugins/java", options: { ...options, distribution: options.distribution ?? "openjdk" } };
}

function javaDirectory(runtime: PluginRuntime<JavaOptions>): string {
  if (runtime.options.install_dir) return runtime.options.install_dir;
  const platform = getPlatform();
  const parent = platform === "windows" ? path.join(runtime.context.env.USERPROFILE ?? os.homedir(), ".genesis", "java") : platform === "macos" ? "/usr/local/java" : "/opt/java";
  return path.join(parent, `jdk-${runtime.options.version}`);
}

async function checkJava(runtime: PluginRuntime<JavaOptions>, executable: string) {
  const result = await runCommand(executable, ["-version"], { cwd: runtime.context.cwd, env: runtime.context.env });
  if (result.code !== 0) return { ok: false, details: "Java is not available on PATH or in its installation directory" };
  const output = result.stderr || result.stdout;
  const version = output.match(/version "(\d+(?:\.\d+)*(?:_\d+)?)"/)?.[1] ?? output.match(/build (\d+(?:\.\d+)*(?:_\d+)?)/)?.[1];
  if (!version) return { ok: false, details: "Java version could not be determined" };
  return matchesVersion(normalizeJavaVersion(version), normalizeJavaVersion(runtime.options.version))
    ? { ok: true, details: `Detected Java ${version}` }
    : { ok: false, details: `Detected Java ${version} but Java ${runtime.options.version} is requested` };
}

function exposeJava(runtime: PluginRuntime<JavaOptions>, directory: string): void {
  runtime.context.env.JAVA_HOME = directory;
  prependPath(runtime.context.env, path.join(directory, "bin"));
}

async function detectJava(runtime: PluginRuntime<JavaOptions>) {
  const directory = javaDirectory(runtime);
  const executable = path.join(directory, "bin", getPlatform() === "windows" ? "java.exe" : "java");
  const managed = fs.existsSync(executable);
  const result = await checkJava(runtime, managed ? executable : "java");
  if (result.ok && managed) exposeJava(runtime, directory);
  return result;
}

export function createPlugin(instance: GenesisPluginInstance<JavaOptions>): GenesisPlugin<JavaOptions> {
  return {
    id: instance.id,
    category: instance.category,
    parseOptions: options => optionSchemas.java.parse(options),
    detect: detectJava,
    async apply(runtime) {
      const detected = await detectJava(runtime);
      if (detected.ok) return { ok: true, didChange: false, details: detected.details };
      if (runtime.options.distribution === "oracle") return { ok: false, didChange: false, details: "Oracle JDK requires manual download and license acceptance" };
      try {
        const platform = getPlatform();
        const destination = javaDirectory(runtime);
        const release = await javaRelease(runtime.options.version, platform);
        await installArchive({
          release, destination, context: runtime.context,
          async select(root) {
            const directory = await singleDirectory(root);
            return platform === "macos" ? path.join(directory, "Contents", "Home") : directory;
          },
          async verify(root) {
            const result = await checkJava(runtime, path.join(root, "bin", platform === "windows" ? "java.exe" : "java"));
            if (!result.ok) throw new Error(result.details);
          },
        });
        exposeJava(runtime, destination);
        runtime.context.logger.info(`Set JAVA_HOME=${destination} and add its bin directory to your shell PATH for future sessions.`);
        return { ok: true, didChange: true, details: `Java ${runtime.options.version} installed to ${destination}` };
      } catch (error) {
        return { ok: false, didChange: error instanceof InstallationRecoveryError, details: `Java installation failed: ${error instanceof Error ? error.message : error}` };
      }
    },
    async validate(runtime) {
      const result = await detectJava(runtime);
      return { ok: result.ok, message: result.details };
    },
  };
}
