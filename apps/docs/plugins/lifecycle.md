# Plugin Lifecycle

Genesis loads every configured plugin and validates its options before executing any provisioning commands. `createPlugin(instance)` must return a plugin with the instance's ID and category. Import failures include the plugin ID and module in the error.

## Options and dependencies

A plugin may expose `parseOptions(value: unknown)` to validate and normalize its configuration. The loader stores the returned options in the execution node. Built-in plugins use this to apply defaults consistently in YAML and TypeScript and reject invalid or unknown options.

`dependsOn` lists other configured plugin IDs. The executor sorts dependencies before dependents. Duplicate IDs, missing dependencies, and cycles fail before task registration.

```typescript
export function createPlugin(instance) {
  return {
    id: instance.id,
    category: instance.category,
    dependsOn: ["node"],
    async preApply(runtime) {
      // Prepare plugin-specific state after system prerequisites succeed.
    },
    async apply(runtime) {
      return { ok: true, didChange: false };
    },
    async postApply(runtime) {
      // Run follow-up work only after this plugin's apply succeeds.
    },
  };
}
```

## Apply order

1. `prepare` runs for every plugin in dependency order, before any task registration. Use it only for bootstrap work that shared prerequisites require. A failure stops the run.
2. `registerTasks` runs for each plugin in dependency order.
3. The task registry executes deduplicated system prerequisites. Missing task dependencies and cycles fail before any task runs. A failed system task stops plugin apply.
4. For each plugin, Genesis runs `preApply`, `apply`, then `postApply`. A returned `ok: false` skips that plugin's post hook and blocks its dependents. Unrelated plugins may continue. Thrown errors stop the run.

Hooks receive the same runtime as `apply`: `instance`, normalized `options`, and `context` (`cwd`, `env`, `logger`, `taskRegistry`). Hooks run even when `apply` reports no change, so make them safe to repeat. Register prerequisites in `registerTasks`; the task execution phase is already over when hooks run.

Node's global npm packages and Homebrew's global packages install after their runtime or package manager is ready. They are not system prerequisites. Failed package installation stops the run.

## Inspection

`detect` reports whether the requested tool is present. `validate` checks the desired state. `doctor` calls both and reports failures with a nonzero exit status. `diff` calls detection without applying plugins.

## Limits

Automatic rollback is not implemented. A failed apply can leave earlier changes in place. Hooks do not imply transactional installs. The default CLI uses sequential plugin execution; the separate parallel engine is a core API.

On macOS, including Homebrew in the config bootstraps and verifies it during `prepare`, before any plugin registers brew tasks.
