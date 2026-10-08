import { expect, it } from "vitest";
import { runCommand } from "./shell.js";

it("stops a timed-out child and returns a failed result", async () => {
  const result = await runCommand(process.execPath, ["-e", "setInterval(() => {}, 30000)"], { timeout: 100 });
  expect(result.code).not.toBe(0);
  expect(result.stderr).toContain("timed out");
});
