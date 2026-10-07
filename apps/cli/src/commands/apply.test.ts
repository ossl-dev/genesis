import { beforeEach, describe, expect, it, vi } from "vitest";
import { Command } from "commander";
const runApply = vi.hoisted(() => vi.fn());
vi.mock("../lib/runner.js", () => ({ runApply }));
import { registerApplyCommand } from "./apply.js";

const cli = () => { const program = new Command(); registerApplyCommand(program); return program; };
beforeEach(() => { vi.clearAllMocks(); vi.spyOn(console, "log").mockImplementation(() => {}); });

describe("apply CLI", () => {
  it("forwards config and dry-run flags", async () => {
    await cli().parseAsync(["apply", "--config", "dev.yml", "--dry-run", "--json"], { from: "user" });
    expect(runApply).toHaveBeenCalledWith({ cwd: process.cwd(), configPath: "dev.yml", dryRun: true, json: true });
    expect(console.log).not.toHaveBeenCalled();
  });

  it("does not print success after an apply failure", async () => {
    runApply.mockRejectedValueOnce(new Error("plugin failed"));
    await expect(cli().parseAsync(["apply"], { from: "user" })).rejects.toThrow("plugin failed");
    expect(console.log).not.toHaveBeenCalled();
  });

  it.each(["--cloud", "env-id"])("fails unsupported cloud requests %j", async arg => {
    await expect(cli().parseAsync(["apply", arg], { from: "user" })).rejects.toThrow("not available");
    expect(runApply).not.toHaveBeenCalled();
  });

  it("requires dry-run for JSON output", async () => {
    await expect(cli().parseAsync(["apply", "--json"], { from: "user" })).rejects.toThrow("requires --dry-run");
  });
});
