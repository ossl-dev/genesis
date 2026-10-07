import { beforeEach, expect, it, vi } from "vitest";
import { Command } from "commander";
const fs = vi.hoisted(() => ({ existsSync: vi.fn(), readdirSync: vi.fn(), readFileSync: vi.fn() }));
vi.mock("node:fs", () => ({ default: fs }));
import { registerListCommand } from "./list.js";

beforeEach(() => {
  vi.resetAllMocks();
  vi.spyOn(console, "log").mockImplementation(() => {});
  vi.spyOn(console, "error").mockImplementation(() => {});
});
const cli = () => { const program = new Command(); registerListCommand(program); return program; };

it("keeps JSON stdout parseable when reporting an unavailable cloud backend", async () => {
  await cli().parseAsync(["list", "--cloud", "--format", "json"], { from: "user" });
  expect(console.log).toHaveBeenCalledOnce();
  expect(JSON.parse(vi.mocked(console.log).mock.calls[0][0])).toEqual([]);
  expect(console.error).toHaveBeenCalled();
});

it("rejects unknown output formats", async () => {
  await expect(cli().parseAsync(["list", "--format", "xml"], { from: "user" })).rejects.toThrow("table or json");
});
