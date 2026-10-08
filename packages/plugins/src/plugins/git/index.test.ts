import { afterEach, beforeEach, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
const mocks = vi.hoisted(() => ({ run: vi.fn(), platform: vi.fn(), install: vi.fn(), source: vi.fn(), windows: vi.fn(), update: vi.fn(), pkg: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getPlatform: mocks.platform, createPackageManagerUpdateTask: mocks.update, createPackageInstallTask: mocks.pkg }));
vi.mock("../../install/archive.js", async original => ({ ...await original<typeof import("../../install/archive.js")>(), installArchive: mocks.install }));
vi.mock("../../install/git-release.js", () => ({ gitSourceRelease: mocks.source, gitWindowsRelease: mocks.windows }));
import { git, createPlugin, type GitOptions } from "./index.js";

const result = (stdout = "", code = 0) => ({ code, stdout, stderr: code ? "failed" : "" });
function setup(options: GitOptions = {}) {
  const instance = git(options);
  const plugin = createPlugin(instance);
  const runtime = { instance, options: plugin.parseOptions!(instance.options), context: { cwd: "/work", env: { HOME: "/home/test", USERPROFILE: "/home/test", PATH: "/host/bin" }, logger: { info: vi.fn(), warn: vi.fn(), error: vi.fn(), debug: vi.fn() }, taskRegistry: { register: vi.fn() } as any } };
  return { plugin, runtime };
}
beforeEach(() => {
  vi.resetAllMocks();
  mocks.platform.mockReturnValue("macos");
  mocks.run.mockResolvedValue(result("", 1));
  mocks.install.mockResolvedValue(undefined);
  mocks.source.mockResolvedValue({ version: "2.53.0", url: "https://example.com/git.tar.gz", sha256: "a".repeat(64), format: "tar.gz" });
  mocks.windows.mockResolvedValue({ version: "2.53.0", url: "https://example.com/git.zip", sha256: "a".repeat(64), format: "zip" });
  mocks.update.mockReturnValue({ id: "update" });
  mocks.pkg.mockReturnValue({ id: "package" });
  vi.spyOn(fs, "existsSync").mockReturnValue(false);
});
afterEach(() => vi.restoreAllMocks());

it.each(["git version 2.53.0", "git version 2.53.0.windows.1", "git version 2.53.0 (Apple Git-1)"])("detects %s", async output => {
  mocks.run.mockResolvedValue(result(output));
  const { plugin, runtime } = setup({ version: "2.53" });
  expect((await plugin.detect!(runtime)).ok).toBe(true);
});
it.each(["git version 2.5.0", "unparseable", ""])("rejects wrong or missing versions: %j", async output => {
  mocks.run.mockResolvedValue(result(output));
  const { plugin, runtime } = setup({ version: "2.53" });
  expect((await plugin.detect!(runtime)).ok).toBe(false);
});
it("skips all work for a matching host Git", async () => {
  mocks.run.mockResolvedValue(result("git version 2.53.0"));
  const { plugin, runtime } = setup();
  await plugin.registerTasks!(runtime);
  expect(runtime.context.taskRegistry.register).not.toHaveBeenCalled();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: false });
});
it("rediscovers managed Git and exposes its binary directory", async () => {
  vi.mocked(fs.existsSync).mockReturnValue(true);
  mocks.run.mockResolvedValue(result("git version 2.53.0"));
  const { plugin, runtime } = setup({ install_method: "source", install_dir: path.resolve("managed git") });
  expect((await plugin.detect!(runtime)).ok).toBe(true);
  expect(mocks.run).toHaveBeenCalledWith(path.join(runtime.options.install_dir!, "bin", "git"), ["--version"], expect.objectContaining({ env: runtime.context.env }));
  expect(runtime.context.env.PATH.split(path.delimiter)[0]).toBe(path.join(runtime.options.install_dir!, "bin"));
});
it("requires an explicit managed directory even when host Git matches", async () => {
  const { plugin, runtime } = setup({ install_method: "source", install_dir: path.resolve("missing") });
  await plugin.detect!(runtime);
  expect(mocks.run.mock.calls[0][0]).toBe(path.join(runtime.options.install_dir!, "bin", "git"));
});
it.each(["macos", "linux"])("rejects nonexistent Unix binary releases before shared tasks on %s", async platform => {
  mocks.platform.mockReturnValue(platform);
  const { plugin, runtime } = setup({ install_method: "binary" });
  await expect(plugin.prepare!(runtime)).rejects.toThrow("no standalone Unix binaries");
  await plugin.registerTasks!(runtime);
  expect(runtime.context.taskRegistry.register).not.toHaveBeenCalled();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: false });
});
it("registers portable Linux package names and source dependencies", async () => {
  mocks.platform.mockReturnValue("linux");
  const { plugin, runtime } = setup();
  await plugin.registerTasks!(runtime);
  expect(mocks.pkg).toHaveBeenCalledWith("git", "/work", runtime.context.env);
  const source = setup({ install_method: "source" });
  await source.plugin.registerTasks!(source.runtime);
  expect(mocks.pkg).toHaveBeenCalledWith("libcurl4-openssl-dev", "/work", source.runtime.context.env);
});
it("fails package apply if the requested version remains unavailable", async () => {
  mocks.run.mockResolvedValue(result("git version 2.5.0"));
  const { plugin, runtime } = setup({ version: "2.53" });
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining("still unavailable") });
});
it("installs pinned sources without cloning Git or changing global user configuration", async () => {
  const { plugin, runtime } = setup({ version: "2.53", install_method: "source" });
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: true });
  expect(mocks.source).toHaveBeenCalledWith("2.53");
  expect(mocks.install).toHaveBeenCalledWith(expect.objectContaining({ destination: path.join("/home/test", ".genesis", "git"), executable: path.join("bin", "git") }));
  expect(mocks.run.mock.calls.every(call => call[1][0] === "--version")).toBe(true);
});
it("checks the selected release and HTTPS helpers before promotion", async () => {
  mocks.install.mockImplementation(async options => options.verify("/stage"));
  mocks.run.mockResolvedValueOnce(result("", 1)).mockResolvedValueOnce(result("git version 2.52.0"));
  const { plugin, runtime } = setup({ install_method: "source" });
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, details: expect.stringContaining("2.53.0 is requested") });
});
it("builds relocatable CLI tools and propagates build failures", async () => {
  const root = await fs.promises.mkdtemp(path.join(os.tmpdir(), "git-test-"));
  await fs.promises.mkdir(path.join(root, "source"));
  mocks.install.mockImplementation(async options => options.select(root));
  mocks.run.mockResolvedValueOnce(result("", 1)).mockResolvedValueOnce(result("compiler failed", 1));
  try {
    const { plugin, runtime } = setup({ install_method: "source" });
    expect(await plugin.apply!(runtime)).toMatchObject({ ok: false, details: expect.stringContaining("Git build failed") });
    expect(mocks.run).toHaveBeenLastCalledWith("make", expect.arrayContaining(["RUNTIME_PREFIX=YesPlease", "all"]), { cwd: expect.stringContaining("genesis-git-"), env: runtime.context.env });
  } finally { await fs.promises.rm(root, { recursive: true, force: true }); }
});
it("installs MinGit on Windows without system package tasks", async () => {
  mocks.platform.mockReturnValue("windows");
  const { plugin, runtime } = setup({ version: "2.53", install_method: "binary" });
  await plugin.registerTasks!(runtime);
  expect(runtime.context.taskRegistry.register).not.toHaveBeenCalled();
  expect(await plugin.apply!(runtime)).toMatchObject({ ok: true, didChange: true });
  expect(mocks.windows).toHaveBeenCalledWith("2.53");
  expect(mocks.install).toHaveBeenCalledWith(expect.objectContaining({ executable: path.join("cmd", "git.exe") }));
});
it("rejects Windows source builds with an actionable alternative", async () => {
  mocks.platform.mockReturnValue("windows");
  const { plugin, runtime } = setup({ install_method: "source" });
  await expect(plugin.prepare!(runtime)).rejects.toThrow("MinGit");
});

it("rejects managed directories in system package mode before detecting the host", () => {
  expect(() => setup({ install_dir: path.resolve("managed git") })).toThrow("install_dir requires source or binary");
  expect(mocks.run).not.toHaveBeenCalled();
});
