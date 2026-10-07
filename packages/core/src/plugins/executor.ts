import { type GenesisPluginCategory } from "../config/schema.js";
import { type PluginExecutionNode } from "./loader.js";
import { type GenesisPluginContext } from "./types.js";

export type PluginStatus = "unknown" | "present" | "missing";

export interface DetectSummary {
  id: string;
  category: GenesisPluginCategory;
  status: PluginStatus;
  details?: string;
}

export interface ApplySummary {
  id: string;
  category: GenesisPluginCategory;
  ok: boolean;
  didChange: boolean;
  details?: string;
}

export interface ValidateSummary {
  id: string;
  category: GenesisPluginCategory;
  ok: boolean;
  message?: string;
}

function getDependencies(node: PluginExecutionNode): string[] {
  return node.plugin.dependsOn ?? [];
}

export function buildPluginGraph(
  nodes: PluginExecutionNode[]
): PluginExecutionNode[] {
  const result: PluginExecutionNode[] = [];
  const temp = new Set<string>();
  const perm = new Set<string>();
  const byId = new Map<string, PluginExecutionNode>();
  for (const node of nodes) {
    if (byId.has(node.instance.id)) throw new Error(`Duplicate plugin id '${node.instance.id}'`);
    byId.set(node.instance.id, node);
  }
  function visit(id: string): void {
    if (perm.has(id)) {
      return;
    }
    if (temp.has(id)) {
      throw new Error(`Cyclic plugin dependency involving '${id}'`);
    }
    const node = byId.get(id);
    if (!node) {
      throw new Error(`Missing plugin '${id}' referenced in dependency`);
    }
    temp.add(id);
    for (const dep of getDependencies(node)) {
      visit(dep);
    }
    temp.delete(id);
    perm.add(id);
    result.push(node);
  }
  for (const node of nodes) {
    visit(node.instance.id);
  }
  return result;
}

export async function runDetect(
  nodes: PluginExecutionNode[],
  context: GenesisPluginContext
): Promise<DetectSummary[]> {
  const result: DetectSummary[] = [];
  for (const node of nodes) {
    if (!node.plugin.detect) {
      result.push({
        id: node.instance.id,
        category: node.instance.category,
        status: "unknown",
      });
      continue;
    }
    const detectResult = await node.plugin.detect({
      instance: node.instance,
      options: node.instance.options,
      context,
    });
    result.push({
      id: node.instance.id,
      category: node.instance.category,
      status: detectResult.ok ? "present" : "missing",
      details: detectResult.details,
    });
  }
  return result;
}

export async function runApply(
  nodes: PluginExecutionNode[],
  context: GenesisPluginContext
): Promise<ApplySummary[]> {
  const ordered = buildPluginGraph(nodes);
  for (const node of ordered) {
    await node.plugin.registerTasks?.({ instance: node.instance, options: node.instance.options, context });
  }

  const taskResults = await context.taskRegistry.executeAll();
  const failures = [...taskResults].filter(([, result]) => !result.ok);
  if (failures.length) {
    throw new Error(`System tasks failed: ${failures.map(([id, result]) => `${id}: ${result.error ?? result.details ?? "failed"}`).join("; ")}`);
  }

  const summaries: ApplySummary[] = [];
  const failed = new Set<string>();
  for (const node of ordered) {
    const runtime = { instance: node.instance, options: node.instance.options, context };
    const dependencies = getDependencies(node).filter(id => failed.has(id));
    let outcome;
    if (dependencies.length) {
      outcome = { ok: false, didChange: false, details: `Dependencies failed: ${dependencies.join(", ")}` };
    } else {
      await node.plugin.preApply?.(runtime);
      outcome = await node.plugin.apply?.(runtime) ?? { ok: true, didChange: false };
      if (outcome.ok) await node.plugin.postApply?.(runtime);
    }
    if (!outcome.ok) failed.add(node.instance.id);
    summaries.push({ id: node.instance.id, category: node.instance.category, ...outcome });
  }
  return summaries;
}

export async function runValidate(
  nodes: PluginExecutionNode[],
  context: GenesisPluginContext
): Promise<ValidateSummary[]> {
  const result: ValidateSummary[] = [];
  for (const node of nodes) {
    if (!node.plugin.validate) {
      result.push({
        id: node.instance.id,
        category: node.instance.category,
        ok: true,
      });
      continue;
    }
    const validateResult = await node.plugin.validate({
      instance: node.instance,
      options: node.instance.options,
      context,
    });
    result.push({
      id: node.instance.id,
      category: node.instance.category,
      ok: validateResult.ok,
      message: validateResult.message,
    });
  }
  return result;
}

export async function runDiff(
  nodes: PluginExecutionNode[],
  context: GenesisPluginContext
): Promise<DetectSummary[]> {
  const summaries = await runDetect(nodes, context);
  return summaries;
}
