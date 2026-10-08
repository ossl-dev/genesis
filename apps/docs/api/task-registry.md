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

Shared tasks use brew on macOS, Chocolatey on Windows, and APT/DNF/pacman/APK selected from Linux `os-release`. Common build dependencies are mapped to distribution packages. Unknown distributions and invalid package names are rejected. Linux root processes run without sudo.

Update tasks refresh metadata for brew/APT/DNF/APK. pacman uses its existing database; update Arch hosts with `pacman -Syu` before provisioning. Chocolatey needs no separate refresh. These tasks never upgrade the whole host.
