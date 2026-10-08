import { getLinuxDistribution, runCommand, type PluginRuntime } from "@ossl/genesis-core";
import type { DockerOptions } from "../plugins/docker/index.js";
import { matchesVersion } from "../options.js";
import { fetchText } from "./archive.js";
import { releaseArchitecture } from "./releases.js";

export function dockerDistribution() {
  const distribution = getLinuxDistribution();
  if (!["ubuntu", "debian", "fedora", "centos", "rhel"].includes(distribution.id)) throw new Error(`Unsupported Linux distribution for Docker installation: ${distribution.id}`);
  if (["ubuntu", "debian"].includes(distribution.id) && !/^[a-z][a-z0-9-]*$/.test(distribution.codename ?? "")) throw new Error("Docker APT installation requires a valid distribution codename");
  releaseArchitecture();
  return distribution;
}

interface PackageVersion { name: string; value: string }

export function selectDockerPackages(packages: PackageVersion[], requested: string, separator: "=" | "-"): string[] {
  const cli = new Map(packages.filter(pkg => pkg.name === "docker-ce-cli").map(pkg => [pkg.value.replace(/^\d+:/, ""), pkg.value]));
  const candidates = packages.filter(pkg => pkg.name === "docker-ce").flatMap(pkg => {
    const key = pkg.value.replace(/^\d+:/, "");
    const version = key.match(/^(\d+\.\d+\.\d+)(?:[-~]|$)/)?.[1];
    const client = cli.get(key);
    return version && client && !/(?:^|[.~_-])(?:rc|beta|alpha)/i.test(key) && (requested === "latest" || matchesVersion(version, requested)) ? [{ version, engine: pkg.value, client }] : [];
  });
  candidates.sort((a, b) => {
    const left = a.version.split(".").map(Number), right = b.version.split(".").map(Number);
    return right[0] - left[0] || right[1] - left[1] || right[2] - left[2] || b.engine.localeCompare(a.engine, undefined, { numeric: true });
  });
  const selected = candidates[0];
  if (!selected) throw new Error(`No matching Docker Engine/CLI packages for ${requested} in the configured repository`);
  return [`docker-ce${separator}${selected.engine}`, `docker-ce-cli${separator}${selected.client}`];
}

export async function installDockerLinux(runtime: PluginRuntime<DockerOptions>, changed: () => void, composeOnly = false): Promise<void> {
  const distribution = dockerDistribution();
  const { cwd, env } = runtime.context;
  const sudo = process.getuid?.() !== 0;
  async function run(command: string, args: string[], elevated = false) {
    const result = await runCommand(elevated && sudo ? "sudo" : command, elevated && sudo ? [command, ...args] : args, { cwd, env });
    if (result.code !== 0) throw new Error(`${command} failed: ${result.stderr || result.stdout}`);
    return result.stdout;
  }
  async function write(urlOrContent: string, destination: string, download: boolean) {
    const content = download ? await fetchText(urlOrContent) : urlOrContent;
    changed();
    await run("bash", ["-c", `set -e; temporary=$(mktemp "$2.genesis.XXXXXX"); trap 'rm -f "$temporary"' EXIT; printf '%s\\n' "$1" >"$temporary"; chmod 0644 "$temporary"; mv -f "$temporary" "$2"`, "genesis", content, destination], true);
  }
  const base = `https://download.docker.com/linux/${distribution.id}`;
  let packages: string[];
  if (["ubuntu", "debian"].includes(distribution.id)) {
    const arch = (await run("dpkg", ["--print-architecture"])).trim();
    if (!["amd64", "arm64"].includes(arch)) throw new Error(`Unsupported Docker APT architecture: ${arch}`);
    changed();
    await run("install", ["-m", "0755", "-d", "/etc/apt/keyrings"], true);
    await write(`${base}/gpg`, "/etc/apt/keyrings/docker.asc", true);
    await run("chmod", ["a+r", "/etc/apt/keyrings/docker.asc"], true);
    await write(`deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.asc] ${base} ${distribution.codename} stable`, "/etc/apt/sources.list.d/docker.list", false);
    await run("apt-get", ["update"], true);
    const available: PackageVersion[] = [];
    for (const name of composeOnly ? [] : ["docker-ce", "docker-ce-cli"]) {
      const output = await run("apt-cache", ["madison", name]);
      for (const line of output.split(/\r?\n/)) {
        const fields = line.split("|").map(value => value.trim());
        if (fields[0] === name && /^[0-9][a-zA-Z0-9.+:~_-]*$/.test(fields[1] ?? "")) available.push({ name, value: fields[1] });
      }
    }
    packages = composeOnly ? [] : [...selectDockerPackages(available, runtime.options.version ?? "latest", "="), "containerd.io", "docker-buildx-plugin"];
    if (runtime.options.include_compose) packages.push("docker-compose-plugin");
    await run("apt-get", ["install", "-y", "--no-install-recommends", ...(runtime.options.version !== "latest" ? ["--allow-downgrades"] : []), ...packages], true);
  } else {
    await write(`${base}/docker-ce.repo`, "/etc/yum.repos.d/docker-ce.repo", true);
    const arch = releaseArchitecture() === "x64" ? "x86_64" : "aarch64";
    const output = composeOnly ? "" : await run("dnf", ["repoquery", "--available", "--queryformat", "%{name}|%{evr}|%{arch}\\n", "docker-ce", "docker-ce-cli"], true);
    const available = output.split(/\r?\n/).flatMap(line => {
      const [name, value, architecture] = line.trim().split("|");
      return ["docker-ce", "docker-ce-cli"].includes(name) && architecture === arch && /^[0-9][a-zA-Z0-9.+:~_-]*$/.test(value ?? "") ? [{ name, value }] : [];
    });
    packages = composeOnly ? [] : [...selectDockerPackages(available, runtime.options.version ?? "latest", "-"), "containerd.io", "docker-buildx-plugin"];
    if (runtime.options.include_compose) packages.push("docker-compose-plugin");
    await run("dnf", ["install", "-y", "--setopt=install_weak_deps=False", ...packages], true);
  }
  if (!composeOnly) await run("systemctl", ["enable", "--now", "docker"], true);
}
