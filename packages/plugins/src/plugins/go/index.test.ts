import { beforeEach, describe, expect, it, vi } from "vitest";
import path from "node:path";
const mocks = vi.hoisted(() => ({ run: vi.fn(), platform: vi.fn(), exists: vi.fn(), install: vi.fn(), release: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getPlatform: mocks.platform }));
vi.mock("node:fs", () => ({ default: { existsSync: mocks.exists } }));
vi.mock("../../install/archive.js", () => ({ InstallationRecoveryError: class extends Error {}, installArchive: mocks.install, singleDirectory: vi.fn(), prependPath: (env: NodeJS.ProcessEnv, dir: string) => { env.PATH = dir; } }));
vi.mock("../../install/releases.js", () => ({ goRelease: mocks.release }));
import { go, createPlugin } from "./index.js";

const success = { code: 0, stdout: "go version go1.22.5 linux/amd64", stderr: "" };
function runtime(version = "1.22.5") {
  const instance = go({ version, install_dir: path.resolve("test-go") });
  return { instance, options: instance.options, context: { cwd: process.cwd(), env: {} as NodeJS.ProcessEnv, logger: { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() }, taskRegistry: {} as any } };
}
beforeEach(() => {
  vi.resetAllMocks();
  mocks.platform.mockReturnValue("linux");
  mocks.exists.mockReturnValue(false);
  mocks.run.mockResolvedValue(success);
  mocks.release.mockResolvedValue({ url: "https://go.dev/archive", sha256: "a".repeat(64), format: "tar.gz" });
});

describe("Go", () => {
  it("skips installation for matching versions", async () => {
    const rt = runtime();
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: false });
    expect(mocks.install).not.toHaveBeenCalled();
  });
  it.each(["macos", "linux", "windows"])("installs and verifies a staged runtime on %s", async platform => {
    mocks.platform.mockReturnValue(platform);
    mocks.run.mockResolvedValueOnce({ code: 1, stdout: "", stderr: "missing" });
    mocks.install.mockImplementation(async options => { await options.verify(path.resolve("staged")); });
    const rt = runtime();
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: true });
    expect(mocks.release).toHaveBeenCalledWith("1.22.5", platform);
    expect(rt.context.env.PATH).toBe(path.join(rt.options.install_dir!, "bin"));
    expect(rt.context.env.GOROOT).toBe(rt.options.install_dir);
    expect(mocks.run.mock.calls[1][0]).toBe(path.resolve("staged", "bin", platform === "windows" ? "go.exe" : "go"));
  });
  it("finds a managed install on later invocations", async () => {
    mocks.exists.mockReturnValue(true);
    const rt = runtime();
    expect((await createPlugin(rt.instance).detect!(rt)).ok).toBe(true);
    expect(mocks.run.mock.calls[0][0]).toBe(path.join(rt.options.install_dir!, "bin", "go"));
  });
  it("rejects a version mismatch before promotion", async () => {
    mocks.run.mockResolvedValue({ ...success, stdout: "go version go1.22.4 linux/amd64" });
    mocks.install.mockImplementation(async options => { await options.verify(path.resolve("staged")); });
    const rt = runtime();
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: false, details: expect.stringContaining("1.22.4") });
  });
  it("reports download/extraction failures", async () => {
    mocks.run.mockResolvedValueOnce({ code: 1, stdout: "", stderr: "" });
    mocks.install.mockRejectedValue(new Error("checksum mismatch"));
    const rt = runtime();
    expect(await createPlugin(rt.instance).apply!(rt)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining("checksum mismatch") });
  });
  it("does not accept version component prefixes", async () => {
    const rt = runtime("1.2");
    expect((await createPlugin(rt.instance).validate!(rt)).ok).toBe(false);
  });
});
