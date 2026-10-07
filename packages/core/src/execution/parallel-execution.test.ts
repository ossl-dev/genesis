import { beforeEach, describe, expect, it, vi } from "vitest";
const cpu = vi.hoisted(() => ({ count: 4 }));
vi.mock("node:os", () => ({ default: { availableParallelism: () => cpu.count } }));
import { ParallelExecutionEngine } from "./parallel-execution.js";
import { TaskRegistry } from "./task-registry.js";
import type { PluginExecutionNode } from "../plugins/loader.js";
import type { GenesisPluginContext } from "../plugins/types.js";

const logger = { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() } as any;
const node = (id: string, dependencies: string[] = [], options = {}): PluginExecutionNode => ({
  instance: { id, category: "tool", module: "test", options },
  plugin: { id, category: "tool", dependsOn: dependencies, apply: vi.fn(async () => ({ ok: true, didChange: true })) },
});
const context = (): GenesisPluginContext => ({ cwd: "/tmp/project", env: {}, logger, taskRegistry: new TaskRegistry(logger) });

beforeEach(() => { cpu.count = 4; vi.clearAllMocks(); });

describe("parallel scheduling", () => {
  it("retains real nodes and groups independent plugins before their dependents", async () => {
    const engine = new ParallelExecutionEngine(logger);
    const a = node("a"); const b = node("b", ["a"]); const c = node("c", ["a"]); const d = node("d", ["b", "c"]);
    const plan = await engine.createExecutionPlan([d, b, c, a]);
    expect(plan.phases.map(phase => phase.plugins.map(plugin => plugin.instance.id))).toEqual([["a"], ["b", "c"], ["d"]]);
    expect(plan.phases[0].plugins[0]).toBe(a);
    expect(plan.parallelism).toBe(2);
    const results = await engine.executeParallel(plan, context());
    expect(results.every(result => result.success)).toBe(true);
    for (const plugin of [a, b, c, d]) expect(plugin.plugin.apply).toHaveBeenCalledOnce();
    expect(results[0].metrics.dependents).toEqual(["b", "c"]);
  });

  it("enforces worker limits and waits for all tasks", async () => {
    let active = 0; let peak = 0; let completed = 0;
    const plugins = Array.from({ length: 9 }, (_, index) => node(`p${index}`));
    for (const plugin of plugins) plugin.plugin.apply = async () => {
      active++; peak = Math.max(peak, active);
      await new Promise(resolve => setTimeout(resolve, 2));
      active--; completed++;
      return { ok: true, didChange: true };
    };
    const engine = new ParallelExecutionEngine(logger, 2);
    const plan = await engine.createExecutionPlan(plugins);
    expect(plan.estimatedTime).toBe(5000);
    expect(await engine.executeParallel(plan, context())).toHaveLength(9);
    expect(peak).toBe(2); expect(completed).toBe(9); expect(active).toBe(0);
  });

  it("clamps concurrency to available CPUs", async () => {
    cpu.count = 1;
    const engine = new ParallelExecutionEngine(logger, 8);
    const plan = await engine.createExecutionPlan([node("a"), node("b")]);
    expect(plan.parallelism).toBe(1);
    expect(plan.estimatedTime).toBe(2000);
  });

  it.each([0, -1, 1.5, NaN, Infinity])("rejects invalid concurrency %s", limit => {
    expect(() => new ParallelExecutionEngine(logger, limit)).toThrow("positive integer");
  });

  it("handles empty plans with finite metrics", async () => {
    const engine = new ParallelExecutionEngine(logger);
    const plan = await engine.createExecutionPlan([]);
    expect(plan).toMatchObject({ phases: [], parallelism: 0, estimatedTime: 0 });
    expect(await engine.executeParallel(plan, context())).toEqual([]);
    const report = await engine.analyzePerformance(plan, []);
    expect(Number.isFinite(report.speedup)).toBe(true);
    expect(Number.isFinite(report.efficiency)).toBe(true);
    expect((await engine.optimizeExecution([])).expectedImprovement).toBe(1);
  });

  it("blocks transitive dependents but still executes independent plugins", async () => {
    const engine = new ParallelExecutionEngine(logger);
    const a = node("a"); a.plugin.apply = vi.fn(async () => ({ ok: false, didChange: false, details: "installer failed" }));
    const b = node("b", ["a"]); const c = node("c", ["b"]); const d = node("d");
    const results = await engine.executeParallel(await engine.createExecutionPlan([c, b, a, d]), context());
    expect(results.find(result => result.pluginId === "a")).toMatchObject({ success: false, error: "installer failed" });
    expect(b.plugin.apply).not.toHaveBeenCalled(); expect(c.plugin.apply).not.toHaveBeenCalled();
    expect(d.plugin.apply).toHaveBeenCalledOnce();
  });

  it.each([["./cache", "cache/subdir"], ["./cache", "cache/../cache"], ["/tmp/cache", "/tmp/cache"]])("serializes overlapping paths %s and %s", async (a, b) => {
    const engine = new ParallelExecutionEngine(logger);
    const plan = await engine.createExecutionPlan([node("a", [], { path: a }), node("b", [], { install_dir: b })]);
    expect(plan.phases[0].parallelizable).toBe(false);
    expect(plan.parallelism).toBe(1);
  });

  it("serializes shared ports", async () => {
    const plan = await new ParallelExecutionEngine(logger).createExecutionPlan([node("a", [], { port: 3000 }), node("b", [], { port: "3000" })]);
    expect(plan.phases[0].parallelizable).toBe(false);
  });

  it("runs prerequisites and hooks before recording success", async () => {
    const order: string[] = [];
    const plugin = node("a");
    plugin.plugin.registerTasks = async runtime => { runtime.context.taskRegistry.register({ id: "setup", description: "setup", executor: async () => { order.push("system"); return { ok: true }; } }); };
    plugin.plugin.preApply = async () => { order.push("pre"); };
    plugin.plugin.apply = async () => { order.push("apply"); return { ok: true, didChange: true }; };
    plugin.plugin.postApply = async () => { order.push("post"); throw new Error("post failed"); };
    const engine = new ParallelExecutionEngine(logger);
    const results = await engine.executeParallel(await engine.createExecutionPlan([plugin]), context());
    expect(order).toEqual(["system", "pre", "apply", "post"]);
    expect(results[0]).toMatchObject({ success: false, error: "post failed" });
  });

  it("rejects missing and cyclic dependencies before execution", async () => {
    const engine = new ParallelExecutionEngine(logger);
    await expect(engine.createExecutionPlan([node("a", ["missing"])])).rejects.toThrow("Missing plugin");
    await expect(engine.createExecutionPlan([node("a", ["b"]), node("b", ["a"])])).rejects.toThrow("Cyclic");
  });
});
