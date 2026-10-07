/**
 * Task Registry for deduplicating and batching system-level operations
 * 
 * This module provides a centralized registry for system-level tasks that
 * should only be executed once per Genesis run, regardless of how many
 * plugins request them.
 * 
 * Examples:
 * - apt update (Linux)
 * - brew update (macOS)
 * - Package manager cache refreshes
 * - System dependency installations
 */

import { type Logger } from "../utils/logger.js";

/**
 * Unique identifier for a system task
 * Format: "platform:category:operation"
 * Examples:
 * - "linux:package-manager:apt-update"
 * - "macos:package-manager:brew-update"
 * - "linux:package-manager:apt-install:curl"
 */
export type TaskId = string;

/**
 * Task execution function
 */
export type TaskExecutor = () => Promise<TaskResult>;

/**
 * Result of a task execution
 */
export interface TaskResult {
  ok: boolean;
  details?: string;
  error?: string;
}

/**
 * Task definition
 */
export interface Task {
  id: TaskId;
  description: string;
  executor: TaskExecutor;
  /**
   * Priority for execution order (higher = earlier)
   * Default: 0
   */
  priority?: number;
  /**
   * Dependencies - task IDs that must complete before this task
   */
  dependsOn?: TaskId[];
}

/**
 * Task execution state
 */
interface TaskState {
  task: Task;
  status: "pending" | "running" | "completed" | "failed";
  result?: TaskResult;
  error?: Error;
}

/**
 * Global task registry for deduplicating system-level operations
 */
export class TaskRegistry {
  private tasks = new Map<TaskId, TaskState>();
  private logger: Logger;

  constructor(logger: Logger) {
    this.logger = logger;
  }

  /**
   * Register a task to be executed
   * If a task with the same ID already exists, it will not be registered again
   */
  register(task: Task): void {
    if (this.tasks.has(task.id)) {
      this.logger.debug(`Task ${task.id} already registered, skipping`);
      return;
    }

    this.logger.debug(`Registering task: ${task.id} - ${task.description}`);
    this.tasks.set(task.id, {
      task,
      status: "pending",
    });
  }

  /**
   * Check if a task has been registered
   */
  has(taskId: TaskId): boolean {
    return this.tasks.has(taskId);
  }

  /**
   * Get the status of a task
   */
  getStatus(taskId: TaskId): "pending" | "running" | "completed" | "failed" | "not-found" {
    const state = this.tasks.get(taskId);
    return state?.status ?? "not-found";
  }

  /**
   * Get the result of a completed task
   */
  getResult(taskId: TaskId): TaskResult | undefined {
    const state = this.tasks.get(taskId);
    return state?.result;
  }

  /**
   * Execute all registered tasks in dependency order
   */
  async executeAll(): Promise<Map<TaskId, TaskResult>> {
    const results = new Map<TaskId, TaskResult>();

    if (this.tasks.size === 0) {
      this.logger.debug("No tasks to execute");
      return results;
    }

    this.logger.info(`Executing ${this.tasks.size} system task(s)...`);

    // Sort tasks by priority and dependencies
    const sortedTasks = this.topologicalSort();

    // Execute tasks in order
    for (const taskId of sortedTasks) {
      const state = this.tasks.get(taskId);
      if (!state) continue;
      if (state.result) {
        results.set(taskId, state.result);
        continue;
      }

      // Check if dependencies completed successfully
      const deps = state.task.dependsOn ?? [];
      const depsFailed = deps.some((depId) => {
        const depState = this.tasks.get(depId);
        return depState?.status === "failed";
      });

      if (depsFailed) {
        this.logger.warn(`Skipping task ${taskId} due to failed dependencies`);
        state.status = "failed";
        state.result = {
          ok: false,
          error: "Dependencies failed",
        };
        results.set(taskId, state.result);
        continue;
      }

      // Execute task
      state.status = "running";
      this.logger.debug(`Executing task: ${state.task.description}`);

      try {
        const result = await state.task.executor();
        state.status = result.ok ? "completed" : "failed";
        state.result = result;
        results.set(taskId, result);

        if (result.ok) {
          this.logger.debug(`Task ${taskId} completed successfully`);
        } else {
          this.logger.warn(`Task ${taskId} failed: ${result.error || result.details}`);
        }
      } catch (error) {
        state.status = "failed";
        state.error = error as Error;
        state.result = {
          ok: false,
          error: (error as Error).message,
        };
        results.set(taskId, state.result);
        this.logger.error(`Task ${taskId} threw error: ${(error as Error).message}`);
      }
    }

    const successCount = Array.from(results.values()).filter((r) => r.ok).length;
    const failCount = results.size - successCount;

    this.logger.info(
      `System tasks completed: ${successCount} succeeded, ${failCount} failed`
    );

    return results;
  }

  /**
   * Topological sort of tasks based on dependencies and priority
   */
  private topologicalSort(): TaskId[] {
    const result: TaskId[] = [];
    const visited = new Set<TaskId>();
    const temp = new Set<TaskId>();

    const visit = (taskId: TaskId): void => {
      if (visited.has(taskId)) return;
      if (temp.has(taskId)) {
        throw new Error(`Cyclic task dependency detected: ${taskId}`);
      }

      const state = this.tasks.get(taskId);
      if (!state) throw new Error(`Missing task dependency: ${taskId}`);

      temp.add(taskId);

      // Visit dependencies first
      const deps = state.task.dependsOn ?? [];
      for (const depId of deps) {
        visit(depId);
      }

      temp.delete(taskId);
      visited.add(taskId);
      result.push(taskId);
    };

    // Get all task IDs sorted by priority (descending)
    const taskIds = Array.from(this.tasks.keys()).sort((a, b) => {
      const priorityA = this.tasks.get(a)?.task.priority ?? 0;
      const priorityB = this.tasks.get(b)?.task.priority ?? 0;
      return priorityB - priorityA;
    });

    for (const taskId of taskIds) {
      visit(taskId);
    }

    return result;
  }

  /**
   * Clear all tasks (useful for testing)
   */
  clear(): void {
    this.tasks.clear();
  }

  /**
   * Get all registered task IDs
   */
  getTaskIds(): TaskId[] {
    return Array.from(this.tasks.keys());
  }
}

