import { beforeEach, describe, expect, it, vi } from "vitest";

const core = vi.hoisted(() => ({
  loadConfig: vi.fn(),
  runApply: vi.fn(),
  runValidate: vi.fn(),
  runDetect: vi.fn(),
  runDiff: vi.fn(),
}));
vi.mock("@ossl/genesis-core", async importOriginal => ({
  ...await importOriginal<typeof import("@ossl/genesis-core")>(),
  ...core,
}));
import { runApply, runDoctor, runValidate } from "./runner.js";

beforeEach(() => {
  vi.clearAllMocks();
  core.loadConfig.mockResolvedValue({ env: { GENESIS_TEST: "configured" } });
  core.runApply.mockResolvedValue([]);
  core.runValidate.mockResolvedValue([]);
  core.runDetect.mockResolvedValue([]);
  vi.spyOn(console, "log").mockImplementation(() => {});
});

describe("runner", () => {
  it("passes the selected config and environment to plugins without mutating process.env", async () => {
    await runApply({ cwd: "/tmp/project", configPath: "dev.yaml" });
    expect(core.loadConfig).toHaveBeenCalledWith("/tmp/project", "dev.yaml");
    expect(core.runApply).toHaveBeenCalledWith([], expect.objectContaining({
      cwd: "/tmp/project", env: expect.objectContaining({ GENESIS_TEST: "configured", PATH: process.env.PATH }),
    }));
    expect(process.env.GENESIS_TEST).toBeUndefined();
  });

  it("fails apply when a plugin returns ok:false", async () => {
    core.runApply.mockResolvedValue([{ id: "node", ok: false, details: "installation failed" }]);
    await expect(runApply({ cwd: "/tmp/project" })).rejects.toThrow("Environment apply failed");
  });

  it("fails validation on environment mismatch", async () => {
    core.runValidate.mockResolvedValue([{ id: "node", ok: false }]);
    await expect(runValidate({ cwd: "/tmp/project" })).rejects.toThrow("Environment validation failed");
  });

  it("fails doctor when an installed plugin fails validation", async () => {
    core.runDetect.mockResolvedValue([{ id: "node", status: "present" }]);
    core.runValidate.mockResolvedValue([{ id: "node", ok: false, message: "wrong version" }]);
    await expect(runDoctor({ cwd: "/tmp/project" })).rejects.toThrow("Environment diagnostics failed");
    expect(console.log).toHaveBeenCalledWith(expect.stringContaining("error"));
  });
});
