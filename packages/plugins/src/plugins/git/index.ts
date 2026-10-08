import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { type GenesisPlugin, type GenesisPluginInstance, type PluginRuntime, runCommand, getPlatform, createPackageManagerUpdateTask, createPackageInstallTask } from "@ossl/genesis-core";
import { optionSchemas, matchesVersion } from "../../options.js";
import { InstallationRecoveryError, installArchive, prependPath, singleDirectory } from "../../install/archive.js";
import { gitSourceRelease, gitWindowsRelease } from "../../install/git-release.js";

export interface GitOptions {
  version?: string;
  install_method?: "package" | "source" | "binary";
  install_dir?: string;
}

export function git(options: GitOptions = {}): GenesisPluginInstance<GitOptions> {
  return { id: "git", category: "tool", module: "@ossl/genesis-plugins/git", options: { version: "latest", install_method: "package", ...options } };
}

function directory(runtime: PluginRuntime<GitOptions>): string {
  return runtime.options.install_dir ?? path.join((getPlatform() === "windows" ? runtime.context.env.USERPROFILE : runtime.context.env.HOME) ?? os.homedir(), ".genesis", "git");
}

function executable(root: string): string {
  return path.join(root, getPlatform() === "windows" ? "cmd" : "bin", getPlatform() === "windows" ? "git.exe" : "git");
}

async function checkGit(runtime: PluginRuntime<GitOptions>, command: string, requested = runtime.options.version ?? "latest") {
  const result = await runCommand(command, ["--version"], { cwd: runtime.context.cwd, env: runtime.context.env });
  if (result.code !== 0) return { ok: false, details: "Git is not available on PATH or in its installation directory" };
  const version = (result.stdout || result.stderr).match(/git version (\d+\.\d+\.\d+)/)?.[1];
  if (!version) return { ok: false, details: "Git version could not be determined" };
  return requested === "latest" || matchesVersion(version, requested)
    ? { ok: true, details: `Detected Git ${version}` }
    : { ok: false, details: `Detected Git ${version} but ${requested} is requested` };
}

async function detectGit(runtime: PluginRuntime<GitOptions>) {
  const managed = runtime.options.install_method !== "package";
  const target = executable(directory(runtime));
  const exists = managed && fs.existsSync(target);
  const result = await checkGit(runtime, exists || (managed && runtime.options.install_dir) ? target : "git");
  if (result.ok && exists) prependPath(runtime.context.env, path.dirname(target));
  return result;
}

function unsupported(runtime: PluginRuntime<GitOptions>): string | undefined {
  const platform = getPlatform();
  if (runtime.options.install_method === "binary" && platform !== "windows") return "Git publishes no standalone Unix binaries; use package or source installation";
  if (runtime.options.install_method === "source" && platform === "windows") return "Git source builds on Windows are unsupported; use install_method: binary for MinGit";
  if (runtime.options.install_method === "package" && platform === "windows") return "Use install_method: binary for automatic MinGit installation on Windows, or preinstall Git";
  if (runtime.options.install_method === "package" && runtime.options.install_dir) return "install_dir requires source or binary installation";
}

export function createPlugin(instance: GenesisPluginInstance<GitOptions>): GenesisPlugin<GitOptions> {
  return {
    id: instance.id, category: instance.category,
    parseOptions: options => optionSchemas.git.parse(options),
    detect: detectGit,
    async prepare(runtime) {
      const reason = unsupported(runtime);
      if (reason && !(await detectGit(runtime)).ok) throw new Error(reason);
    },
    async registerTasks(runtime) {
      if (unsupported(runtime) || runtime.options.install_method === "binary" || (await detectGit(runtime)).ok) return;
      const { cwd, env, taskRegistry } = runtime.context;
      taskRegistry.register(createPackageManagerUpdateTask(cwd, env));
      const packages = runtime.options.install_method === "package" ? ["git"]
        : getPlatform() === "macos" ? ["make"]
        : ["build-essential", "libcurl4-openssl-dev", "zlib1g-dev", "libexpat1-dev", "perl"];
      for (const name of packages) taskRegistry.register(createPackageInstallTask(name, cwd, env));
    },
    async apply(runtime) {
      const detected = await detectGit(runtime);
      if (detected.ok) return { ok: true, didChange: false, details: detected.details };
      const reason = unsupported(runtime);
      if (reason) return { ok: false, didChange: false, details: reason };
      if (runtime.options.install_method === "package") return { ok: false, didChange: false, details: `Git is still unavailable after package installation: ${detected.details}; use source installation for a release pin` };
      try {
        const windows = getPlatform() === "windows";
        const release = await (windows ? gitWindowsRelease : gitSourceRelease)(runtime.options.version ?? "latest");
        const destination = directory(runtime);
        await installArchive({
          release, destination, executable: path.relative(destination, executable(destination)), context: runtime.context,
          async select(root) {
            if (windows) return root;
            const extracted = await singleDirectory(root);
            const staged = path.join(root, "install");
            // Git's Makefile requires a source path without whitespace. Keep the
            // final prefix in staging, but compile in an isolated temporary path.
            if (/\s/.test(os.tmpdir())) throw new Error("Git source builds require a temporary directory without whitespace; set TMPDIR");
            const work = await fs.promises.mkdtemp(path.join(os.tmpdir(), "genesis-git-"));
            const source = path.join(work, "source");
            try {
              await fs.promises.cp(extracted, source, { recursive: true });
              const args = ["prefix=/usr/local", "RUNTIME_PREFIX=YesPlease", "NO_GETTEXT=YesPlease", "NO_TCLTK=YesPlease", "NO_OPENSSL=YesPlease"];
              for (const step of [[`-j${Math.min(os.cpus().length || 1, 8)}`, ...args, "all"], [...args, `DESTDIR=${staged}`, "install"]]) {
                const result = await runCommand("make", step, { cwd: source, env: runtime.context.env });
                if (result.code !== 0) throw new Error(`Git build failed: ${result.stderr || result.stdout}`);
              }
            } finally {
              await fs.promises.rm(work, { recursive: true, force: true }).catch(error => runtime.context.logger.warn(`Could not remove Git build directory ${work}: ${error}`));
            }
            return path.join(staged, "usr", "local");
          },
          async verify(root) {
            const result = await checkGit(runtime, executable(root), release.version);
            if (!result.ok) throw new Error(result.details);
            const helpers = await runCommand(executable(root), ["--exec-path"], { cwd: runtime.context.cwd, env: runtime.context.env });
            const helperDirectory = helpers.stdout.trim();
            const relative = path.relative(await fs.promises.realpath(root), await fs.promises.realpath(helperDirectory));
            if (helpers.code !== 0 || path.isAbsolute(relative) || relative === ".." || relative.startsWith(`..${path.sep}`) || !fs.existsSync(path.join(helperDirectory, windows ? "git-remote-https.exe" : "git-remote-https"))) throw new Error("Git HTTPS helper is unavailable inside the installation directory");
          },
        });
        prependPath(runtime.context.env, path.dirname(executable(destination)));
        runtime.context.logger.info(`Add ${path.dirname(executable(destination))} to your shell PATH for future sessions.`);
        return { ok: true, didChange: true, details: `Git ${release.version} installed to ${destination}` };
      } catch (error) {
        return { ok: false, didChange: error instanceof InstallationRecoveryError, details: `Git installation failed: ${error instanceof Error ? error.message : error}` };
      }
    },
    async validate(runtime) {
      const result = await detectGit(runtime);
      return { ok: result.ok, message: result.details };
    },
  };
}
