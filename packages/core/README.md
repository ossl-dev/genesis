# @ossl/genesis-core

Core configuration, plugin lifecycle, task scheduling, and shell utilities for Genesis. See the [Core API](../../apps/docs/api/core.md) and [Architecture](../../apps/docs/guide/architecture.md).

```typescript
import {
  loadConfig, collectPluginInstances, loadPlugins,
  applyEnvironment, Logger, TaskRegistry,
} from "@ossl/genesis-core";

const cwd = process.cwd();
const config = await loadConfig(cwd);
const nodes = await loadPlugins(collectPluginInstances(config));
const logger = new Logger({ level: "info" });
const results = await applyEnvironment(config, nodes, {
  cwd, env: { ...process.env }, logger, taskRegistry: new TaskRegistry(logger),
});
if (results.some(result => !result.ok)) throw new Error("Apply failed");
```

`applyEnvironment` executes configured scripts, plugin prerequisites/lifecycle, and repositories. `runApply` executes only the plugin lifecycle. Graph errors fail before provisioning. System task failures reject apply; returned plugin failures block dependents. There is no rollback.

[ParallelExecutionEngine](../../apps/docs/api/execution.md) is an opt-in API with bounded workers and dependency layers. Resource checks inspect configured paths/ports, not every system resource.

`EnvironmentCacheManager` is an unfinished prototype. It does not provide usable filesystem restore, network restore, encryption, or remote synchronization. It is not wired into CLI apply. Do not rely on its snapshot/restore results as backups.

From the repository root: `bun run build`, `bun run lint`, `bun run test`.
