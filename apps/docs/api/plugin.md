# Plugin API

```typescript
interface GenesisPlugin<TOptions = unknown> {
  id: string;
  category: GenesisPluginCategory;
  dependsOn?: string[];
  parseOptions?(options: unknown): TOptions;
  registerTasks?(runtime: PluginRuntime<TOptions>): Promise<void>;
  preApply?(runtime: PluginRuntime<TOptions>): Promise<void>;
  apply?(runtime: PluginRuntime<TOptions>): Promise<ApplyResult>;
  postApply?(runtime: PluginRuntime<TOptions>): Promise<void>;
  detect?(runtime: PluginRuntime<TOptions>): Promise<DetectResult>;
  validate?(runtime: PluginRuntime<TOptions>): Promise<ValidateResult>;
}
```

Categories are `tool`, `sdk`, `language`, `library`, and `framework`. `createPlugin(instance)` must return the configured ID/category. `parseOptions` validates/defaults options at load time. Dependencies refer to configured instance IDs, not package names.

## Runtime

```typescript
interface PluginRuntime<TOptions> {
  instance: GenesisPluginInstance<TOptions>;
  options: TOptions;
  context: GenesisPluginContext;
}
interface GenesisPluginContext {
  cwd: string;
  env: NodeJS.ProcessEnv;
  logger: Logger;
  taskRegistry: TaskRegistry;
}
```

Pass `cwd` and `env` to child commands so config environment values and selected runtimes reach them. Register system prerequisites in `registerTasks`; that phase is over when apply hooks run.

## Results

| Method | Result |
| --- | --- |
| `detect` | `{ ok: boolean, details?: string }` |
| `apply` | `{ ok: boolean, didChange: boolean, details?: string }` |
| `validate` | `{ ok: boolean, message?: string }` |

A successful subprocess is not enough to establish the desired state; verify the requested executable/version. Returning `ok: false` blocks dependents and skips postApply. A thrown error stops sequential execution. Hooks run even after an unchanged successful apply and must be repeatable.

Missing detection reports `unknown`; missing validation currently reports success. A missing apply method reports successful/no-change execution. Supply methods when the tool needs meaningful verification.

See [Lifecycle](/plugins/lifecycle) for ordering and failure semantics. Rollback is not implemented.
