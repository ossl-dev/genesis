# Task Registry

Plugins often share prerequisites such as a package-manager update or curl installation. Genesis registers those operations by exact ID and executes each once per registry. Duplicate registrations retain the first task.

```typescript
async registerTasks(runtime) {
  const { cwd, env, taskRegistry } = runtime.context;
  taskRegistry.register(createPackageManagerUpdateTask(cwd, env));
  taskRegistry.register(createPackageInstallTask("curl", cwd, env));
}
```

The install helper depends on the update ID, so both must be registered. Missing dependencies and cycles fail before any task executes. A failed task blocks its dependents; any failed system prerequisite stops plugin apply.

Task dependencies take precedence over priority. Completed results are reused if executeAll is called again. Create a fresh registry for each apply, and do not run a registry concurrently.

Shared Linux helpers currently use APT. macOS helpers require brew to already be available. Platform-specific installers must choose appropriate package names; the registry does not discover distribution package mappings.

Global Node/Homebrew packages run after those tools are ready, rather than as prerequisites. See [Lifecycle](/plugins/lifecycle) and [Task API](/api/task-registry).
