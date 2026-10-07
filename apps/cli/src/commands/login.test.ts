import { beforeEach, expect, it, vi } from "vitest";
import { Command } from "commander";
const fs = vi.hoisted(() => ({ mkdirSync: vi.fn(), writeFileSync: vi.fn(), chmodSync: vi.fn(), existsSync: vi.fn(), readFileSync: vi.fn() }));
vi.mock("node:fs", () => ({ default: fs }));
import { registerLoginCommand } from "./login.js";

beforeEach(() => {
  vi.resetAllMocks();
  vi.spyOn(console, "log").mockImplementation(() => {});
});
const cli = () => { const program = new Command(); registerLoginCommand(program); return program; };

it("writes private token metadata and tightens permissions on existing files", async () => {
  await cli().parseAsync(["login", "--token", "test-token", "--url", "https://example.test"], { from: "user" });
  expect(fs.writeFileSync).toHaveBeenCalledWith(expect.stringContaining("auth.json"), expect.any(String), { mode: 0o600 });
  expect(fs.chmodSync).toHaveBeenCalledWith(expect.stringContaining("auth.json"), 0o600);
  expect(console.log).not.toHaveBeenCalledWith(expect.stringContaining("Authenticated"));
});

it.each(["null", '{"url":"https://example.test"}', '{"token":42,"url":"https://example.test"}'])("ignores malformed stored metadata %s", async raw => {
  fs.existsSync.mockReturnValue(true);
  fs.readFileSync.mockReturnValue(raw);
  await cli().parseAsync(["login"], { from: "user" });
  expect(console.log).toHaveBeenCalledWith(expect.stringContaining("OAuth is not implemented"));
});
