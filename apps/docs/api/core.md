# Core API

Exports from `@ossl/genesis-core`.

## Configuration

```typescript
loadConfig(cwd: string, configPath?: string, env?: NodeJS.ProcessEnv): Promise<GenesisConfig>
validateConfig(value: unknown): GenesisConfig
defineConfig(config: GenesisConfig): GenesisConfig
```

`loadConfig` discovers TS then YAML, normalizes YAML entries, expands environment references, and reports errors with file/field context. `validateConfig` validates the normalized structure and rejects duplicate IDs. `defineConfig` is a typed identity helper.

`GenesisConfig` has optional `tools`, `sdks`, `languages`, `repositories`, `scripts`, and `env` sections. Repository specs have `url`, `folder`, and optional `branch`; the loader also accepts `path` as an input alias. Script specs have `name`, `command`, optional `description`, and optional `when: before | after`.

## Plugins

```typescript
collectPluginInstances(config: GenesisConfig): GenesisPluginInstance[]
loadPlugin(instance: GenesisPluginInstance): Promise<PluginExecutionNode>
loadPlugins(instances: GenesisPluginInstance[]): Promise<PluginExecutionNode[]>
buildPluginGraph(nodes: PluginExecutionNode[]): PluginExecutionNode[]
```

The loader imports `createPlugin`, checks identity/category, and runs optional `parseOptions`. Graph building rejects duplicates, missing dependencies, and cycles. See [Plugin API](/api/plugin).

```typescript
runDetect(nodes, context): Promise<DetectSummary[]>
runDiff(nodes, context): Promise<DetectSummary[]>
runApply(nodes, context): Promise<ApplySummary[]>
runValidate(nodes, context): Promise<ValidateSummary[]>
```

`runDiff` currently delegates to detection. `runApply` sorts dependencies, runs deduplicated prerequisites, then pre/apply/post hooks. Failed prerequisites reject. Returned plugin failures block dependents; thrown errors stop the run. These functions do not themselves set a process exit status.

## Environment execution

```typescript
applyEnvironment(config, nodes, context): Promise<ApplySummary[]>
createEnvironmentPlan(config, nodes): { environment: string[]; actions: EnvironmentAction[] }
```

`applyEnvironment` merges config environment values into a copy of context, runs before scripts, applies plugins, checks/clones repositories, then runs after scripts. Plugin failures return summaries and prevent repository/after-script work. Script/repository errors reject.

The plan contains ordered script/plugin/repository actions and environment variable names. It does not call lifecycle methods or inspect installed state, and excludes environment values.

## Tasks and utilities

See [Task Registry](/api/task-registry), [Parallel Execution](/api/execution), and [Utilities](/api/utilities).

`EnvironmentCacheManager` and its snapshot/artifact types are exported but unfinished. Compression stores metadata in memory, optional "remote" sync writes local JSON, restore does not restore files/network, and encryption is not implemented. The CLI has no usable restore workflow. These APIs are not backup guarantees.
