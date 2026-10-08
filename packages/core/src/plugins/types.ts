import {
  type GenesisPluginCategory,
  type GenesisPluginInstance,
} from "../config/schema.js";
import { type Logger } from "../utils/logger.js";
import { type TaskRegistry } from "../execution/task-registry.js";

export interface DetectResult {
  ok: boolean;
  details?: string;
}

export interface ApplyResult {
  ok: boolean;
  didChange: boolean;
  details?: string;
}

export interface ValidateResult {
  ok: boolean;
  message?: string;
}

export interface GenesisPluginContext {
  cwd: string;
  env: NodeJS.ProcessEnv;
  logger: Logger;
  /**
   * Task registry for deduplicating system-level operations
   * Plugins can register tasks that should only run once per Genesis execution
   */
  taskRegistry: TaskRegistry;
}

export interface PluginRuntime<TOptions = unknown> {
  instance: GenesisPluginInstance<TOptions>;
  options: TOptions;
  context: GenesisPluginContext;
}

export interface GenesisPlugin<TOptions = unknown> {
  id: string;
  category: GenesisPluginCategory;
  dependsOn?: string[];
  parseOptions?(options: unknown): TOptions;
  prepare?(runtime: PluginRuntime<TOptions>): Promise<void>;
  preApply?(runtime: PluginRuntime<TOptions>): Promise<void>;
  postApply?(runtime: PluginRuntime<TOptions>): Promise<void>;
  detect?(runtime: PluginRuntime<TOptions>): Promise<DetectResult>;
  /**
   * Optional: Register system-level tasks (e.g., package manager updates, system package installations)
   * These tasks will be deduplicated and executed before the apply phase
   * Use this to register prerequisites that need to be installed at the system level
   */
  registerTasks?(runtime: PluginRuntime<TOptions>): Promise<void>;
  apply?(runtime: PluginRuntime<TOptions>): Promise<ApplyResult>;
  validate?(runtime: PluginRuntime<TOptions>): Promise<ValidateResult>;
}
