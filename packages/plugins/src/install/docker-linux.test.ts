import { afterEach, beforeEach, expect, it, vi } from "vitest";
import os from "node:os";
const mocks = vi.hoisted(() => ({ run: vi.fn(), distro: vi.fn(), fetch: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getLinuxDistribution: mocks.distro }));
vi.mock("./archive.js", () => ({ fetchText: mocks.fetch }));
import { dockerDistribution, installDockerLinux, selectDockerPackages } from "./docker-linux.js";
const originalGetuid = Object.getOwnPropertyDescriptor(process, "getuid");
afterEach(() => { vi.restoreAllMocks(); if (originalGetuid) Object.defineProperty(process, "getuid", originalGetuid); else Reflect.deleteProperty(process, "getuid"); });
const runtime = { options: { version: "29.1", include_compose: false }, context: { cwd: "/work", env: { PATH: "/tools" } } } as any;
beforeEach(() => {
  vi.resetAllMocks();
  mocks.fetch.mockResolvedValue("repository contents");
  Object.defineProperty(process, "getuid", { configurable: true, writable: true, value: () => 0 });
  vi.spyOn(os, "arch").mockReturnValue("x64");
  mocks.distro.mockReturnValue({ id: "debian", like: [], codename: "bookworm" });
  mocks.run.mockImplementation(async (command, args) => ({ code: 0, stderr: "", stdout: command === "dpkg" ? "amd64" : command === "apt-cache" ? `${args[1]} | 5:29.1.0-1~debian.12~bookworm | repo` : "" }));
});
it("pairs RPM versions with different epochs and selects the newest matching release", () => {
  const packages = [
    { name: "docker-ce", value: "3:29.1.0-1.fc43" }, { name: "docker-ce-cli", value: "1:29.1.0-1.fc43" },
    { name: "docker-ce", value: "3:29.1.1-1.fc43" }, { name: "docker-ce-cli", value: "1:29.1.1-1.fc43" },
    { name: "docker-ce", value: "3:29.1.2-1.fc43" },
  ];
  expect(selectDockerPackages(packages, "29.1", "-")).toEqual(["docker-ce-3:29.1.1-1.fc43", "docker-ce-cli-1:29.1.1-1.fc43"]);
});
it("does not confuse numeric prefixes or select preview packages", () => {
  const packages = ["docker-ce", "docker-ce-cli"].flatMap(name => [{ name, value: "5:29.10.0-1" }, { name, value: "5:29.1.0~rc1-1" }]);
  expect(() => selectDockerPackages(packages, "29.1", "=")).toThrow("No matching");
});
it.each(["arch", "alpine", "notubuntu"])("rejects unsupported distro %s exactly", id => {
  mocks.distro.mockReturnValue({ id, like: ["ubuntu"], codename: "noble" });
  expect(() => dockerDistribution()).toThrow("Unsupported Linux");
});
it("rejects missing or unsafe codenames", () => {
  mocks.distro.mockReturnValue({ id: "ubuntu", codename: 'noble;bad' });
  expect(() => dockerDistribution()).toThrow("valid distribution codename");
});
it("installs exact Debian versions, excludes Compose, and starts the service", async () => {
  const changed = vi.fn();
  await installDockerLinux(runtime, changed);
  expect(changed).toHaveBeenCalled();
  expect(mocks.run).toHaveBeenCalledWith("apt-get", ["install", "-y", "--no-install-recommends", "--allow-downgrades", "docker-ce=5:29.1.0-1~debian.12~bookworm", "docker-ce-cli=5:29.1.0-1~debian.12~bookworm", "containerd.io", "docker-buildx-plugin"], { cwd: "/work", env: runtime.context.env });
  expect(mocks.run).toHaveBeenLastCalledWith("systemctl", ["enable", "--now", "docker"], expect.anything());
  expect(mocks.run.mock.calls.some(call => call[0] === "sudo" || call[1].includes("usermod"))).toBe(false);
});
it("passes repository values as shell arguments rather than shell code", async () => {
  await installDockerLinux(runtime, vi.fn());
  const writes = mocks.run.mock.calls.filter(call => call[0] === "bash");
  expect(mocks.fetch).toHaveBeenCalledWith("https://download.docker.com/linux/debian/gpg");
  expect(writes.every(call => call[1][1].includes("mktemp") && call[1][1].includes("mv -f"))).toBe(true);
  expect(writes.every(call => !call[1][1].includes("bookworm"))).toBe(true);
});
it("does not install an unrequested version if the repository has no match", async () => {
  await expect(installDockerLinux({ ...runtime, options: { ...runtime.options, version: "28" } }, vi.fn())).rejects.toThrow("No matching");
  expect(mocks.run.mock.calls.some(call => call[0] === "apt-get" && call[1][0] === "install")).toBe(false);
});
it.each(["fedora", "centos", "rhel"])("uses the correct %s repository and handles RPM epochs", async id => {
  mocks.distro.mockReturnValue({ id, like: [] });
  mocks.run.mockImplementation(async (command, args) => ({ code: 0, stderr: "", stdout: command === "dnf" && args[0] === "repoquery" ? "docker-ce|3:29.1.0-1.fc43|x86_64\ndocker-ce-cli|1:29.1.0-1.fc43|x86_64\ndocker-ce|3:30.0.0-1.fc43|aarch64" : "" }));
  await installDockerLinux({ ...runtime, options: { ...runtime.options, include_compose: true } }, vi.fn());
  expect(mocks.fetch).toHaveBeenCalledWith(`https://download.docker.com/linux/${id}/docker-ce.repo`);
  expect(mocks.run).toHaveBeenCalledWith("dnf", ["repoquery", "--available", "--queryformat", "%{name}|%{evr}|%{arch}\\n", "docker-ce", "docker-ce-cli"], expect.anything());
  expect(mocks.run).toHaveBeenCalledWith("dnf", ["install", "-y", "--setopt=install_weak_deps=False", "docker-ce-3:29.1.0-1.fc43", "docker-ce-cli-1:29.1.0-1.fc43", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"], expect.anything());
  expect(mocks.run.mock.calls.some(call => call[1][0] === "update")).toBe(false);
});
it("propagates repository write failures", async () => {
  mocks.run.mockImplementation(async command => ({ code: command === "bash" ? 1 : 0, stderr: "write failed", stdout: "amd64" }));
  await expect(installDockerLinux(runtime, vi.fn())).rejects.toThrow("write failed");
});

it("adds Compose without querying or reinstalling Engine packages or enabling a service", async () => {
  await installDockerLinux({ ...runtime, options: { ...runtime.options, include_compose: true } }, vi.fn(), true);
  expect(mocks.run).toHaveBeenCalledWith("apt-get", ["install", "-y", "--no-install-recommends", "--allow-downgrades", "docker-compose-plugin"], expect.anything());
  expect(mocks.run.mock.calls.some(call => ["apt-cache", "systemctl"].includes(call[0]))).toBe(false);
});
