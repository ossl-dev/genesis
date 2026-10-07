import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { applyEnvironment, createEnvironmentPlan } from "./environment.js";
import { TaskRegistry } from "./task-registry.js";
import { Logger } from "../utils/logger.js";
import { runCommand } from "../os/shell.js";
import type { PluginExecutionNode } from "../plugins/loader.js";
import type { GenesisPluginContext } from "../plugins/types.js";

let cwd: string;
let context: GenesisPluginContext;
const logger = new Logger({ level: "error" });
beforeEach(async () => {
  cwd = await fs.mkdtemp(path.join(os.tmpdir(), "genesis-test-"));
  context = { cwd, env: { ...process.env }, logger, taskRegistry: new TaskRegistry(logger) };
});
afterEach(async () => { await fs.rm(cwd, { recursive: true, force: true }); });
const script = (text: string) => `"${process.execPath}" -e "require('node:fs').appendFileSync('order.txt', '${text}' + process.env.GENESIS_TEST)"`;
const plugin = (apply: PluginExecutionNode["plugin"]["apply"]): PluginExecutionNode => ({
  instance: { id: "fixture", category: "tool", module: "fixture" }, plugin: { id: "fixture", category: "tool", apply },
});

async function createSourceRepo(): Promise<string> {
  const source = path.join(cwd, "source");
  const init = await runCommand("git", ["init", "--initial-branch=main", source]);
  if (init.code) throw new Error(init.stderr);
  await fs.writeFile(path.join(source, "file.txt"), "tracked");
  await runCommand("git", ["-C", source, "add", "."]);
  const commit = await runCommand("git", ["-C", source, "-c", "user.name=Genesis Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "-m", "fixture"]);
  if (commit.code) throw new Error(commit.stderr);
  return source;
}

describe("environment integration", () => {
  it("runs real scripts around plugin apply with configured environment values", async () => {
    const fixture = plugin(async runtime => {
      await fs.appendFile(path.join(runtime.context.cwd, "order.txt"), `plugin${runtime.context.env.GENESIS_TEST}`);
      return { ok: true, didChange: true };
    });
    await applyEnvironment({ env: { GENESIS_TEST: "=value;" }, scripts: [
      { name: "after", command: script("after") },
      { name: "before", command: script("before"), when: "before" },
    ] }, [fixture], context);
    expect(await fs.readFile(path.join(cwd, "order.txt"), "utf8")).toBe("before=value;plugin=value;after=value;");
    expect(process.env.GENESIS_TEST).toBeUndefined();
  });

  it("clones a real local repository and preserves local edits on repeated apply", async () => {
    const source = await createSourceRepo();
    const config = { repositories: [{ url: source, folder: "repos/clone", branch: "main" }] };
    await applyEnvironment(config, [], context);
    const target = path.join(cwd, "repos", "clone", "file.txt");
    expect(await fs.readFile(target, "utf8")).toBe("tracked");
    await fs.writeFile(target, "local edit");
    await applyEnvironment(config, [], context);
    expect(await fs.readFile(target, "utf8")).toBe("local edit");
  });

  it("refuses an unrelated existing directory", async () => {
    await fs.mkdir(path.join(cwd, "existing"));
    await fs.writeFile(path.join(cwd, "existing", "keep.txt"), "keep");
    await expect(applyEnvironment({ repositories: [{ url: "unused", folder: "existing" }] }, [], context)).rejects.toThrow("not a repository root");
    expect(await fs.readFile(path.join(cwd, "existing", "keep.txt"), "utf8")).toBe("keep");
  });

  it("refuses a repository with a different remote or branch", async () => {
    const source = await createSourceRepo();
    const config = { repositories: [{ url: source, folder: "clone", branch: "main" }] };
    await applyEnvironment(config, [], context);
    await expect(applyEnvironment({ repositories: [{ ...config.repositories[0], url: "other" }] }, [], context)).rejects.toThrow("different origin");
    await expect(applyEnvironment({ repositories: [{ ...config.repositories[0], branch: "other" }] }, [], context)).rejects.toThrow("requested branch");
  });

  it("stops after a failed script and does not apply plugins", async () => {
    let applied = false;
    await expect(applyEnvironment({ scripts: [{ name: "broken", command: `"${process.execPath}" -e "process.exit(9)"`, when: "before" }] }, [plugin(async () => {
      applied = true; return { ok: true, didChange: true };
    })], context)).rejects.toThrow("Script 'broken' failed (exit 9)");
    expect(applied).toBe(false);
  });

  it("does not clone or run after scripts following plugin failure", async () => {
    const summaries = await applyEnvironment({ repositories: [{ url: "unused", folder: "clone" }], scripts: [{ name: "after", command: script("after") }] }, [plugin(async () => ({ ok: false, didChange: false }))], context);
    expect(summaries[0].ok).toBe(false);
    await expect(fs.access(path.join(cwd, "order.txt"))).rejects.toThrow();
    await expect(fs.access(path.join(cwd, "clone"))).rejects.toThrow();
  });

  it("builds a plan without calling plugins or exposing environment values", () => {
    let applied = false;
    const plan = createEnvironmentPlan({ env: { TOKEN: "secret" } }, [plugin(async () => {
      applied = true; return { ok: true, didChange: true };
    })]);
    expect(plan.environment).toEqual(["TOKEN"]);
    expect(JSON.stringify(plan)).not.toContain("secret");
    expect(applied).toBe(false);
  });
});
