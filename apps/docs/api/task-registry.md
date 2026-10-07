# Task Registry API

`TaskRegistry` deduplicates system operations by exact task ID within one execution context.

```typescript
interface Task {
  id: string;
  description: string;
  executor: () => Promise<TaskResult>;
  priority?: number;
  dependsOn?: string[];
}
interface TaskResult {
  ok: boolean;
  details?: string;
  error?: string;
}
```

| Method | Behavior |
| --- | --- |
| `register(task)` | Keep the first task for an ID; ignore duplicates |
| `has(id)` | Check registration |
| `getStatus(id)` | pending/running/completed/failed/not-found |
| `getResult(id)` | Completed result, if available |
| `getTaskIds()` | Registered IDs |
| `executeAll()` | Execute in dependency/priority order; return a result Map |
| `clear()` | Remove registrations/results |

Dependencies override priority. Higher priority runs first among the traversal's available task roots. Missing dependencies and cycles reject before task execution. Failed dependencies skip their child tasks. Thrown executor errors become failed results. Repeated executeAll calls reuse completed results; clear/create another registry for another run. Do not invoke executeAll concurrently on one registry.

## Task helpers

- `createPackageManagerTaskId(operation, packageName?)`
- `createPackageManagerUpdateTask(cwd, env)`
- `createPackageInstallTask(packageName, cwd, env)`
- `createCommandCheckTask(commandName, cwd, env)`
- `createCustomTask(id, description, executor, { priority?, dependsOn? })`
- `ensurePackageManagerUpdate()` and `ensurePackageInstalled(name)` return IDs only.

Install tasks depend on the update task: register both. Dependency IDs must match exactly; wildcards are not resolved. Custom task IDs are prefixed with the platform and `custom:`.

Shared tasks use brew on macOS, APT on Linux, and Chocolatey on Windows. Package names are not translated across distributions. The current Windows update helper upgrades all Chocolatey packages, rather than just refreshing metadata; built-in Windows installers skip this path.
