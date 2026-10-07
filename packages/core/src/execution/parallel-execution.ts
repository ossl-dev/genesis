import os from "node:os";
import path from "node:path";
import { type Logger } from "../utils/logger.js";
import { type PluginExecutionNode } from "../plugins/loader.js";
import { type GenesisPluginContext } from "../plugins/types.js";
import { buildPluginGraph, runApplyPrerequisites } from "../plugins/executor.js";

export interface ParallelExecutionPlan {
  phases: ExecutionPhase[];
  totalPlugins: number;
  estimatedTime: number;
  parallelism: number;
}

export interface ExecutionPhase {
  id: string;
  plugins: PluginExecutionNode[];
  dependencies: string[];
  estimatedDuration: number;
  parallelizable: boolean;
}

export interface PluginMetrics {
  id: string;
  executionTime: number;
  memoryUsage: number;
  cpuUsage: number;
  dependencies: string[];
  dependents: string[];
  criticalPath: boolean;
}

export interface ExecutionResult {
  pluginId: string;
  success: boolean;
  duration: number;
  error?: string;
  metrics: PluginMetrics;
}

export interface OptimizationReport {
  originalTime: number;
  optimizedTime: number;
  speedup: number;
  efficiency: number;
  recommendations: string[];
}

export class ParallelExecutionEngine {
  private readonly maxConcurrency: number;
  private metrics = new Map<string, PluginMetrics>();
  private executionHistory: ExecutionResult[] = [];
  private elapsed = new WeakMap<ParallelExecutionPlan, number>();

  constructor(private logger: Logger, maxConcurrency = 4) {
    if (!Number.isInteger(maxConcurrency) || maxConcurrency < 1) {
      throw new Error("maxConcurrency must be a positive integer");
    }
    this.maxConcurrency = Math.min(maxConcurrency, Math.max(1, os.availableParallelism()));
  }

  async createExecutionPlan(plugins: PluginExecutionNode[]): Promise<ParallelExecutionPlan> {
    const ordered = buildPluginGraph(plugins);
    const levels = new Map<string, number>();
    const phases: ExecutionPhase[] = [];
    for (const node of ordered) {
      const dependencies = node.plugin.dependsOn ?? [];
      const level = dependencies.reduce((max, id) => Math.max(max, levels.get(id)! + 1), 0);
      levels.set(node.instance.id, level);
      phases[level] ??= {
        id: `phase-${level}`, plugins: [], dependencies: [], estimatedDuration: 0, parallelizable: false,
      };
      phases[level].plugins.push(node);
    }
    for (const phase of phases) {
      phase.dependencies = [...new Set(phase.plugins.flatMap(node => node.plugin.dependsOn ?? []))];
      phase.parallelizable = phase.plugins.length > 1 && !this.hasResourceConflicts(phase.plugins);
      phase.estimatedDuration = this.estimatePhaseDuration(phase);
    }
    const plan = {
      phases, totalPlugins: plugins.length,
      estimatedTime: phases.reduce((sum, phase) => sum + phase.estimatedDuration, 0),
      parallelism: this.calculateParallelism(phases),
    };
    this.logger.debug(`Execution plan: ${phases.length} phases, up to ${plan.parallelism} workers`);
    return plan;
  }

  async executeParallel(plan: ParallelExecutionPlan, context: GenesisPluginContext): Promise<ExecutionResult[]> {
    const nodes = plan.phases.flatMap(phase => phase.plugins);
    // Validate the graph before running any system prerequisites.
    await runApplyPrerequisites(buildPluginGraph(nodes), context);
    const start = Date.now();
    const outcomes = new Map<string, ExecutionResult>();
    const dependents = new Map<string, string[]>();
    for (const node of nodes) {
      for (const id of node.plugin.dependsOn ?? []) {
        const list = dependents.get(id) ?? [];
        list.push(node.instance.id);
        dependents.set(id, list);
      }
    }
    const results: ExecutionResult[] = [];
    for (const phase of plan.phases) {
      const phaseResults = new Array<ExecutionResult>(phase.plugins.length);
      let next = 0;
      const worker = async () => {
        while (next < phase.plugins.length) {
          const index = next++;
          const node = phase.plugins[index];
          const dependencies = node.plugin.dependsOn ?? [];
          const failed = dependencies.filter(id => !outcomes.get(id)?.success);
          const result = await this.executePlugin(node, context, dependents.get(node.instance.id) ?? [], failed);
          phaseResults[index] = result;
        }
      };
      // Recheck paths against the actual execution cwd, including relative aliases.
      const parallel = phase.parallelizable && !this.hasResourceConflicts(phase.plugins, context.cwd);
      const workers = parallel ? Math.min(this.maxConcurrency, phase.plugins.length) : 1;
      await Promise.all(Array.from({ length: workers }, worker));
      for (const result of phaseResults) outcomes.set(result.pluginId, result);
      results.push(...phaseResults);
    }
    this.elapsed.set(plan, Date.now() - start);
    this.executionHistory.push(...results);
    return results;
  }

  async analyzePerformance(plan: ParallelExecutionPlan, results: ExecutionResult[]): Promise<OptimizationReport> {
    const byId = new Map(results.map(result => [result.pluginId, result]));
    const originalTime = results.reduce((sum, result) => sum + result.duration, 0);
    const optimizedTime = this.elapsed.get(plan) ?? plan.phases.reduce((sum, phase) => {
      const durations = phase.plugins.map(node => byId.get(node.instance.id)?.duration ?? 0);
      return sum + (phase.parallelizable ? Math.max(0, ...durations) : durations.reduce((a, b) => a + b, 0));
    }, 0);
    const speedup = optimizedTime > 0 ? originalTime / optimizedTime : 1;
    const recommendations: string[] = [];
    const failed = results.filter(result => !result.success);
    if (failed.length) recommendations.push(`Resolve failed plugins: ${failed.map(result => result.pluginId).join(", ")}`);
    const slow = results.filter(result => result.duration > 5000).sort((a, b) => b.duration - a.duration).slice(0, 3);
    if (slow.length) recommendations.push(`Review slow plugins: ${slow.map(result => result.pluginId).join(", ")}`);
    return { originalTime, optimizedTime, speedup, efficiency: speedup / Math.max(1, plan.parallelism), recommendations };
  }

  async optimizeExecution(plugins: PluginExecutionNode[]): Promise<{ optimizedPlan: ParallelExecutionPlan; expectedImprovement: number }> {
    const optimizedPlan = await this.createExecutionPlan(plugins);
    const sequentialTime = plugins.reduce((sum, node) => sum + this.estimatePluginDuration(node), 0);
    return { optimizedPlan, expectedImprovement: optimizedPlan.estimatedTime > 0 ? sequentialTime / optimizedPlan.estimatedTime : 1 };
  }

  getMetrics(): PluginMetrics[] {
    return [...this.metrics.values()];
  }

  getExecutionHistory(): ExecutionResult[] {
    return [...this.executionHistory];
  }

  private async executePlugin(
    node: PluginExecutionNode,
    context: GenesisPluginContext,
    dependents: string[],
    failedDependencies: string[],
  ): Promise<ExecutionResult> {
    const start = Date.now();
    const memory = process.memoryUsage().heapUsed;
    const cpu = process.cpuUsage();
    let error: string | undefined;
    try {
      if (failedDependencies.length) throw new Error(`Dependencies failed: ${failedDependencies.join(", ")}`);
      const runtime = { instance: node.instance, options: node.instance.options, context };
      await node.plugin.preApply?.(runtime);
      const result = await node.plugin.apply?.(runtime);
      if (result && !result.ok) throw new Error(result.details ?? "Plugin apply failed");
      await node.plugin.postApply?.(runtime);
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    }
    const duration = Date.now() - start;
    const usage = process.cpuUsage(cpu);
    const metrics: PluginMetrics = {
      id: node.instance.id, executionTime: duration,
      memoryUsage: Math.max(0, process.memoryUsage().heapUsed - memory),
      cpuUsage: duration > 0 ? (usage.user + usage.system) / (duration * 1000) : 0,
      dependencies: node.plugin.dependsOn ?? [], dependents, criticalPath: dependents.length > 0,
    };
    this.metrics.set(node.instance.id, metrics);
    return { pluginId: node.instance.id, success: error === undefined, duration, ...(error !== undefined ? { error } : {}), metrics };
  }

  private calculateParallelism(phases: ExecutionPhase[]): number {
    return phases.reduce((max, phase) => Math.max(max, phase.parallelizable ? Math.min(this.maxConcurrency, phase.plugins.length) : 1), 0);
  }

  private estimatePluginDuration(node: PluginExecutionNode): number {
    return this.metrics.get(node.instance.id)?.executionTime ?? 1000;
  }

  private estimatePhaseDuration(phase: ExecutionPhase): number {
    const slots = Array<number>(phase.parallelizable ? Math.min(this.maxConcurrency, phase.plugins.length) : 1).fill(0);
    for (const node of phase.plugins) {
      const slot = slots.indexOf(Math.min(...slots));
      slots[slot] += this.estimatePluginDuration(node);
    }
    return Math.max(0, ...slots);
  }

  private hasResourceConflicts(nodes: PluginExecutionNode[], cwd = process.cwd()): boolean {
    const paths: string[] = [];
    const ports = new Set<unknown>();
    const keys = ["path", "install_dir", "home", "prefix", "target", "destination", "output_dir", "cache_dir"];
    for (const node of nodes) {
      const options = node.instance.options as Record<string, unknown> | undefined;
      if (!options) continue;
      const current = keys.flatMap(key => typeof options[key] === "string" ? [path.resolve(cwd, options[key] as string)] : []);
      if (current.some(candidate => paths.some(existing => {
        const a = path.relative(candidate, existing);
        const b = path.relative(existing, candidate);
        const contains = (relative: string) => relative === "" || (!path.isAbsolute(relative) && relative !== ".." && !relative.startsWith(`..${path.sep}`));
        return contains(a) || contains(b);
      }))) return true;
      paths.push(...current);
      if (options.port !== undefined) {
        if (ports.has(String(options.port))) return true;
        ports.add(String(options.port));
      }
    }
    return false;
  }
}
