import { beforeEach, describe, expect, it, vi } from "vitest";
import path from "node:path";
const mocks = vi.hoisted(() => ({ run: vi.fn(), platform: vi.fn(), exists: vi.fn(), install: vi.fn(), release: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getPlatform: mocks.platform }));
vi.mock("node:fs", () => ({ default: { existsSync: mocks.exists } }));
vi.mock("../../install/archive.js", () => ({ InstallationRecoveryError: class extends Error {}, installArchive: mocks.install, singleDirectory: async (root: string) => path.join(root, "jdk"), prependPath: (env: NodeJS.ProcessEnv, dir: string) => { env.PATH = dir; } }));
vi.mock("../../install/releases.js", async original => ({ ...await original<typeof import("../../install/releases.js")>(), javaRelease: mocks.release }));
import { java, createPlugin } from "./index.js";

const success = { code: 0, stdout: "", stderr: 'openjdk version "17.0.9" 2023-10-17' };
function runtime(version = "17") {
  const instance = java({ version, install_dir: path.resolve("test-java") });
  return { instance, options: instance.options, context: { cwd: process.cwd(), env: {} as NodeJS.ProcessEnv, logger: { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() }, taskRegistry: {} as any } };
}
beforeEach(() => {
  vi.resetAllMocks();
  mocks.platform.mockReturnValue("linux");
  mocks.exists.mockReturnValue(false);
  mocks.run.mockResolvedValue(success);
  mocks.release.mockResolvedValue({ url: "https://example.com/jdk", sha256: "a".repeat(64), format: "tar.gz" });
});

describe("Java", () => {
  it.each(['17', '17.0', '17.0.9'])("matches the requested %s components", async version => {
    const rt = runtime(version);
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: false });
    expect(mocks.install).not.toHaveBeenCalled();
  });
  it.each(['1.8', '8'])("recognizes legacy Java 8 for %s", async version => {
    mocks.run.mockResolvedValue({ ...success, stderr: 'java version "1.8.0_202"' });
    const rt = runtime(version);
    expect((await createPlugin(rt.instance).detect!(rt)).ok).toBe(true);
  });
  it("recognizes versions with no minor/patch", async () => {
    mocks.run.mockResolvedValue({ ...success, stderr: 'openjdk version "17"' });
    const rt = runtime();
    expect((await createPlugin(rt.instance).detect!(rt)).ok).toBe(true);
  });
  it.each(["macos", "linux", "windows"])("installs and exposes JAVA_HOME on %s", async platform => {
    mocks.platform.mockReturnValue(platform);
    mocks.run.mockResolvedValueOnce({ code: 1, stdout: "", stderr: "missing" });
    mocks.install.mockImplementation(async options => {
      const selected = await options.select(path.resolve("unpack"));
      expect(selected).toBe(path.resolve("unpack", "jdk", ...(platform === "macos" ? ["Contents", "Home"] : [])));
      await options.verify(selected);
    });
    const rt = runtime();
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: true });
    expect(rt.context.env.JAVA_HOME).toBe(rt.options.install_dir);
    expect(rt.context.env.PATH).toBe(path.join(rt.options.install_dir!, "bin"));
  });
  it("rejects a mismatched patch instead of only checking the major", async () => {
    const rt = runtime('17.0.8');
    expect((await createPlugin(rt.instance).validate!(rt)).ok).toBe(false);
  });
  it("rejects failed staged verification", async () => {
    mocks.run.mockResolvedValue({ code: 1, stdout: "", stderr: "missing" });
    mocks.install.mockImplementation(async options => { await options.verify(path.resolve("stage")); });
    const rt = runtime();
    expect((await createPlugin(rt.instance).apply!(rt)).ok).toBe(false);
  });
  it("does not install Oracle JDK automatically", async () => {
    mocks.run.mockResolvedValue({ code: 1, stdout: "", stderr: "missing" });
    const rt = runtime(); rt.options.distribution = "oracle";
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: false, details: expect.stringContaining("license") });
    expect(mocks.release).not.toHaveBeenCalled();
  });
  it("discovers a managed install and exports its environment", async () => {
    mocks.exists.mockReturnValue(true);
    const rt = runtime();
    expect((await createPlugin(rt.instance).detect!(rt)).ok).toBe(true);
    expect(rt.context.env.JAVA_HOME).toBe(rt.options.install_dir);
  });
});
