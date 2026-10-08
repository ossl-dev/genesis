import { type GenesisPlugin, type GenesisPluginInstance, type PluginRuntime, runCommand, getPlatform, createPackageManagerUpdateTask, createPackageInstallTask } from "@ossl/genesis-core";
import { optionSchemas, matchesVersion } from "../../options.js";
import { dockerDistribution, installDockerLinux } from "../../install/docker-linux.js";

export interface DockerOptions {
  version?: string;
  include_compose?: boolean;
  install_desktop?: boolean;
}

export function docker(options: DockerOptions = {}): GenesisPluginInstance<DockerOptions> {
  return { id: "docker", category: "tool", module: "@ossl/genesis-plugins/docker", options: { version: "latest", include_compose: true, install_desktop: false, ...options } };
}

async function detectDocker(runtime: PluginRuntime<DockerOptions>): Promise<{ ok: boolean; details: string; composeMissing?: boolean }> {
  const { cwd, env } = runtime.context;
  const result = await runCommand("docker", ["--version"], { cwd, env });
  if (result.code !== 0) return { ok: false, details: "Docker is not available on PATH" };
  const version = (result.stdout || result.stderr).match(/Docker version (\d+\.\d+\.\d+)/)?.[1];
  if (!version) return { ok: false, details: "Docker version could not be determined" };
  if (runtime.options.version && runtime.options.version !== "latest" && !matchesVersion(version, runtime.options.version)) return { ok: false, details: `Detected Docker ${version} but ${runtime.options.version} is requested` };
  if (!runtime.options.include_compose) return { ok: true, details: `Detected Docker ${version}` };
  for (const [command, args] of [["docker", ["compose", "version"]], ["docker-compose", ["--version"]]] as const) {
    const compose = await runCommand(command, [...args], { cwd, env });
    const composeVersion = (compose.stdout || compose.stderr).match(/(?:Docker Compose|docker-compose) version v?(\d+\.\d+\.\d+)/i)?.[1];
    if (compose.code === 0 && composeVersion) return { ok: true, details: `Detected Docker ${version} and Docker Compose ${composeVersion}` };
  }
  return { ok: false, composeMissing: true, details: `Detected Docker ${version} but Docker Compose is unavailable` };
}

async function validateDocker(runtime: PluginRuntime<DockerOptions>, detected?: Awaited<ReturnType<typeof detectDocker>>) {
  detected ??= await detectDocker(runtime);
  if (!detected.ok) return detected;
  const result = await runCommand("docker", ["info", "--format", "{{.ServerVersion}}"], { cwd: runtime.context.cwd, env: runtime.context.env, timeout: 15_000 });
  if (result.code !== 0) return { ok: false, details: `Docker daemon is unavailable: ${result.stderr || result.stdout}; start Docker and check socket permissions` };
  const version = result.stdout.trim().match(/^(\d+\.\d+\.\d+)(?:[-+\s]|$)/)?.[1];
  if (!version) return { ok: false, details: "Docker daemon version could not be determined" };
  if (runtime.options.version && runtime.options.version !== "latest" && !matchesVersion(version, runtime.options.version)) return { ok: false, details: `Docker daemon ${version} does not match requested ${runtime.options.version}` };
  return { ok: true, details: `${detected.details}; daemon ${version} is available` };
}

function unsupported(runtime: PluginRuntime<DockerOptions>): string | undefined {
  const platform = getPlatform();
  if (platform === "windows" || runtime.options.install_desktop) return "Install and start Docker Desktop manually (https://docs.docker.com/desktop/), then rerun Genesis";
  if (platform === "macos" && runtime.options.version !== "latest") return "Automatic Docker version pinning on macOS is unsupported; preinstall a matching client and daemon or use version: latest";
}

export function createPlugin(instance: GenesisPluginInstance<DockerOptions>): GenesisPlugin<DockerOptions> {
  return {
    id: instance.id, category: instance.category,
    parseOptions: options => optionSchemas.docker.parse(options),
    detect: detectDocker,
    async prepare(runtime) {
      if ((await detectDocker(runtime)).ok) return;
      const reason = unsupported(runtime);
      if (reason) throw new Error(reason);
      if (getPlatform() === "linux") dockerDistribution();
    },
    async registerTasks(runtime) {
      if (unsupported(runtime)) return;
      const detected = await detectDocker(runtime);
      if (detected.ok) {
        if (getPlatform() !== "macos" || (await validateDocker(runtime, detected)).ok) return;
      }
      if (getPlatform() === "linux") dockerDistribution();
      const { cwd, env, taskRegistry } = runtime.context;
      taskRegistry.register(createPackageManagerUpdateTask(cwd, env));
      const packages = getPlatform() === "macos" ? ["colima", ...(!detected.ok ? ["docker", ...(runtime.options.include_compose ? ["docker-compose"] : [])] : [])] : ["ca-certificates"];
      for (const name of packages) taskRegistry.register(createPackageInstallTask(name, cwd, env));
    },
    async apply(runtime) {
      const detected = await detectDocker(runtime);
      if (detected.ok) {
        const validated = await validateDocker(runtime, detected);
        if (validated.ok || getPlatform() !== "macos" || unsupported(runtime)) return { ...validated, didChange: false };
        if (!validated.details.startsWith("Docker daemon is unavailable")) return { ...validated, didChange: false };
      }
      const reason = unsupported(runtime);
      if (reason) return { ok: false, didChange: false, details: reason };
      let didChange = false;
      try {
        if (getPlatform() === "macos") {
          didChange = true;
          const result = await runCommand("colima", ["start"], { cwd: runtime.context.cwd, env: runtime.context.env });
          if (result.code !== 0) throw new Error(`Colima start failed: ${result.stderr || result.stdout}`);
        } else {
          await installDockerLinux(runtime, () => { didChange = true; }, detected.composeMissing === true);
        }
        return { ...await validateDocker(runtime), didChange };
      } catch (error) {
        return { ok: false, didChange, details: `Docker installation failed: ${error instanceof Error ? error.message : error}` };
      }
    },
    async validate(runtime) {
      const result = await validateDocker(runtime);
      return { ok: result.ok, message: result.details };
    },
  };
}
