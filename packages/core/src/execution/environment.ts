import fs from "node:fs/promises";
import path from "node:path";
import { type GenesisConfig, type ScriptSpec, type RepoSpec } from "../config/schema.js";
import { type PluginExecutionNode } from "../plugins/loader.js";
import { type GenesisPluginContext } from "../plugins/types.js";
import { buildPluginGraph, runApply, type ApplySummary } from "../plugins/executor.js";
import { runCommand } from "../os/shell.js";
import { getPlatform } from "../os/platform.js";

export type EnvironmentAction =
  | { type: "script"; name: string; command: string; when: "before" | "after" }
  | { type: "plugin"; id: string; module: string; dependsOn: string[] }
  | { type: "repository"; url: string; folder: string; branch?: string };

export function createEnvironmentPlan(config: GenesisConfig, nodes: PluginExecutionNode[]) {
  const scripts = config.scripts ?? [];
  const action = (script: ScriptSpec): EnvironmentAction => ({
    type: "script", name: script.name, command: script.command, when: script.when ?? "after",
  });
  const actions: EnvironmentAction[] = [
    ...scripts.filter(script => script.when === "before").map(action),
    ...buildPluginGraph(nodes).map(node => ({ type: "plugin" as const, id: node.instance.id, module: node.instance.module, dependsOn: node.plugin.dependsOn ?? [] })),
    ...(config.repositories ?? []).map(repo => ({ type: "repository" as const, ...repo })),
    ...scripts.filter(script => script.when !== "before").map(action),
  ];
  return { environment: Object.keys(config.env ?? {}), actions };
}

async function runScript(script: ScriptSpec, context: GenesisPluginContext): Promise<void> {
  context.logger.info(`Running script: ${script.name}`);
  const result = getPlatform() === "windows"
    ? await runCommand("cmd.exe", ["/d", "/s", "/c", script.command], context)
    : await runCommand("sh", ["-c", script.command], context);
  if (result.stdout) context.logger.info(result.stdout);
  if (result.code !== 0) throw new Error(`Script '${script.name}' failed (exit ${result.code}): ${result.stderr}`);
}

async function ensureRepository(repo: RepoSpec, context: GenesisPluginContext): Promise<void> {
  const folder = path.resolve(context.cwd, repo.folder);
  let exists = false;
  try {
    await fs.lstat(folder);
    exists = true;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "ENOENT") throw error;
  }
  if (exists) {
    const root = await runCommand("git", ["-C", folder, "rev-parse", "--show-toplevel"], context);
    if (root.code !== 0 || await fs.realpath(folder) !== await fs.realpath(root.stdout.trim())) {
      throw new Error(`Repository destination '${repo.folder}' exists but is not a repository root`);
    }
    const remote = await runCommand("git", ["-C", folder, "remote", "get-url", "origin"], context);
    if (remote.code !== 0 || remote.stdout.trim() !== repo.url) {
      throw new Error(`Repository '${repo.folder}' has a different origin; refusing to overwrite it`);
    }
    if (repo.branch) {
      const branch = await runCommand("git", ["-C", folder, "symbolic-ref", "--short", "HEAD"], context);
      if (branch.code !== 0 || branch.stdout.trim() !== repo.branch) {
        throw new Error(`Repository '${repo.folder}' is not on requested branch '${repo.branch}'`);
      }
    }
    context.logger.debug(`Repository already present: ${repo.folder}`);
    return;
  }
  await fs.mkdir(path.dirname(folder), { recursive: true });
  const result = await runCommand("git", ["clone", ...(repo.branch ? ["--branch", repo.branch] : []), "--", repo.url, folder], context);
  if (result.code !== 0) throw new Error(`Failed to clone '${repo.url}': ${result.stderr}`);
}

export async function applyEnvironment(
  config: GenesisConfig,
  nodes: PluginExecutionNode[],
  context: GenesisPluginContext,
): Promise<ApplySummary[]> {
  const ordered = buildPluginGraph(nodes);
  context = { ...context, env: { ...context.env, ...config.env } };
  for (const script of config.scripts ?? []) {
    if (script.when === "before") await runScript(script, context);
  }
  const summaries = await runApply(ordered, context);
  if (summaries.some(summary => !summary.ok)) return summaries;
  for (const repo of config.repositories ?? []) await ensureRepository(repo, context);
  for (const script of config.scripts ?? []) {
    if (script.when !== "before") await runScript(script, context);
  }
  return summaries;
}
