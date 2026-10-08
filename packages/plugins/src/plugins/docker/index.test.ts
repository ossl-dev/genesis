import { afterEach, beforeEach, expect, it, vi } from "vitest";
const mocks = vi.hoisted(() => ({ run: vi.fn(), platform: vi.fn(), install: vi.fn(), distro: vi.fn(), update: vi.fn(), pkg: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getPlatform: mocks.platform, createPackageManagerUpdateTask: mocks.update, createPackageInstallTask: mocks.pkg }));
vi.mock("../../install/docker-linux.js", () => ({ installDockerLinux: mocks.install, dockerDistribution: mocks.distro }));
import { docker, createPlugin, type DockerOptions } from "./index.js";
const ok = (stdout = "", code = 0) => ({ code, stdout, stderr: code ? "unavailable" : "" });
function setup(options: DockerOptions = {}) {
  const instance = docker(options), plugin = createPlugin(instance);
  const runtime = { instance, options: plugin.parseOptions!(instance.options), context: { cwd: "/work", env: { PATH: "/tools" }, logger: { info: vi.fn(), warn: vi.fn(), error: vi.fn(), debug: vi.fn() }, taskRegistry: { register: vi.fn() } as any } };
  return { plugin, runtime };
}
function available(compose = true, daemon = true) {
  mocks.run.mockImplementation(async (command, args) => {
    if (command === "docker" && args[0] === "--version") return ok("Docker version 29.1.0, build 123");
    if (command === "docker" && args[0] === "compose" && compose) return ok("Docker Compose version v2.40.0");
    if (command === "docker" && args[0] === "info" && daemon) return ok("29.1.0");
    return ok("", 1);
  });
}
beforeEach(() => { vi.resetAllMocks(); mocks.platform.mockReturnValue("linux"); mocks.run.mockResolvedValue(ok("", 1)); mocks.update.mockReturnValue({ id: "update" }); mocks.pkg.mockReturnValue({ id: "pkg" }); });
afterEach(() => vi.restoreAllMocks());

it("validates options before installation", () => {
  expect(() => createPlugin(docker()).parseOptions!({ version: "29;bad" })).toThrow();
});
it("detects a matching client and Compose", async () => {
  available();
  const { plugin, runtime } = setup({ version: "29.1" });
  expect((await plugin.detect!(runtime)).ok).toBe(true);
});
it("does not accept component-prefix version collisions", async () => {
  available();
  const { plugin, runtime } = setup({ version: "2" });
  expect((await plugin.detect!(runtime)).ok).toBe(false);
});
it("does not require Compose when disabled", async () => {
  available(false);
  const { plugin, runtime } = setup({ include_compose: false });
  expect((await plugin.detect!(runtime)).ok).toBe(true);
  expect(mocks.run).toHaveBeenCalledTimes(1);
});
it.each(["docker-compose version 1.29.2, build 123", "Docker Compose version v2.40.0"])("recognizes legacy fallback output %s", async output => {
  mocks.run.mockResolvedValueOnce(ok("Docker version 29.1.0")).mockResolvedValueOnce(ok("", 1)).mockResolvedValueOnce(ok(output));
  const { plugin, runtime } = setup();
  expect((await plugin.detect!(runtime)).ok).toBe(true);
  expect(mocks.run).toHaveBeenLastCalledWith("docker-compose", ["--version"], { cwd: "/work", env: runtime.context.env });
});
it("falls back when a successful Compose subcommand returns malformed output", async () => {
  mocks.run.mockResolvedValueOnce(ok("Docker version 29.1.0")).mockResolvedValueOnce(ok("bad")).mockResolvedValueOnce(ok("docker-compose version 1.29.2"));
  const { plugin, runtime } = setup();
  expect((await plugin.detect!(runtime)).ok).toBe(true);
});
it("validates the daemon even when the CLI already exists", async () => {
  available(true, false);
  const { plugin, runtime } = setup();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining("daemon is unavailable") });
  expect((await plugin.validate!(runtime)).ok).toBe(false);
  expect(mocks.install).not.toHaveBeenCalled();
});
it("checks the daemon version separately from the client", async () => {
  available();
  const run = mocks.run.getMockImplementation()!;
  mocks.run.mockImplementation((command, args) => args[0] === "info" ? Promise.resolve(ok("28.5.1")) : run(command, args));
  const { plugin, runtime } = setup({ version: "29.1" });
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining("daemon 28.5.1") });
});
it("skips matching installations and package work", async () => {
  available();
  const { plugin, runtime } = setup();
  await plugin.prepare!(runtime);
  await plugin.registerTasks!(runtime);
  expect(runtime.context.taskRegistry.register).not.toHaveBeenCalled();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: false });
});
it.each(["windows", "macos"])("rejects unsupported pins before tasks on %s", async platform => {
  mocks.platform.mockReturnValue(platform);
  const { plugin, runtime } = setup({ version: "29.1" });
  await expect(plugin.prepare!(runtime)).rejects.toThrow();
  await plugin.registerTasks!(runtime);
  expect(runtime.context.taskRegistry.register).not.toHaveBeenCalled();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: false });
});
it("reports Desktop setup as manual without downloading an unverified DMG", async () => {
  mocks.platform.mockReturnValue("macos");
  const { plugin, runtime } = setup({ install_desktop: true });
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining("manually") });
  expect(mocks.run.mock.calls.every(call => call[0] === "docker")).toBe(true);
});
it("fails unsupported distributions before registering packages", async () => {
  mocks.distro.mockImplementation(() => { throw new Error("Unsupported Linux distribution"); });
  const { plugin, runtime } = setup();
  await expect(plugin.prepare!(runtime)).rejects.toThrow("Unsupported");
  await expect(plugin.registerTasks!(runtime)).rejects.toThrow("Unsupported");
  expect(runtime.context.taskRegistry.register).not.toHaveBeenCalled();
});
it("does not reinstall Homebrew packages while starting Colima", async () => {
  mocks.platform.mockReturnValue("macos");
  mocks.run.mockResolvedValueOnce(ok("", 1)).mockImplementation(async command => {
    if (command === "colima") { available(); return ok(); }
    return ok("", 1);
  });
  const { plugin, runtime } = setup();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: true });
  expect(mocks.run).toHaveBeenCalledWith("colima", ["start"], { cwd: "/work", env: runtime.context.env });
  expect(mocks.run.mock.calls.some(call => call[0] === "brew")).toBe(false);
});
it("starts Colima after shared tasks made the CLI available", async () => {
  mocks.platform.mockReturnValue("macos");
  available(true, false);
  const run = mocks.run.getMockImplementation()!;
  mocks.run.mockImplementation(async (command, args) => {
    if (command === "colima") { available(); return ok(); }
    return run(command, args);
  });
  const { plugin, runtime } = setup();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: true });
});
it("registers only Colima for an existing client with a stopped daemon", async () => {
  mocks.platform.mockReturnValue("macos");
  available(true, false);
  const { plugin, runtime } = setup();
  await plugin.registerTasks!(runtime);
  expect(mocks.pkg).toHaveBeenCalledTimes(1);
  expect(mocks.pkg).toHaveBeenCalledWith("colima", "/work", runtime.context.env);
});
it("propagates partial installation failures as changed", async () => {
  mocks.install.mockImplementation(async (_runtime, changed) => { changed(); throw new Error("package install failed"); });
  const { plugin, runtime } = setup();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: true, details: expect.stringContaining("package install failed") });
});
it("verifies client, Compose, and daemon after installation", async () => {
  mocks.install.mockImplementation(async (_runtime, changed) => { changed(); available(false); });
  const { plugin, runtime } = setup();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: true, details: expect.stringContaining("Compose is unavailable") });
});

it("adds only Compose when the existing Docker client already matches", async () => {
  available(false);
  const { plugin, runtime } = setup();
  mocks.install.mockImplementation(async (_runtime, changed) => { changed(); available(); });
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: true });
  expect(mocks.install).toHaveBeenCalledWith(runtime, expect.any(Function), true);
});
