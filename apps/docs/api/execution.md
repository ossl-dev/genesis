# Parallel Execution API

The CLI applies plugins sequentially. Core API consumers can opt into parallel execution for plugins whose installers can safely run together.

```typescript
import { ParallelExecutionEngine, Logger, TaskRegistry } from "@ossl/genesis-core";

const logger = new Logger({ level: "info" });
const engine = new ParallelExecutionEngine(logger, 2);
const plan = await engine.createExecutionPlan(nodes);
const results = await engine.executeParallel(plan, {
  cwd: process.cwd(),
  env: { ...process.env },
  logger,
  taskRegistry: new TaskRegistry(logger),
});

if (results.some(result => !result.success)) {
  throw new Error("Environment apply failed");
}
```

`nodes` are loaded `PluginExecutionNode` objects. Planning rejects duplicate IDs, missing dependencies, and cycles. Independent plugins share a phase; dependents run in later phases. The worker limit must be a positive integer and is capped at the host's available CPU count.

Execution registers and runs system prerequisites once, then calls `preApply`, `apply`, and `postApply`. Failed prerequisites reject execution. Failed plugin results, exceptions, and hook failures are recorded in results and block transitive dependents; independent plugins can finish.

## Resource checks

A phase runs sequentially when plugin options share a port or overlapping paths in `path`, `install_dir`, `home`, `prefix`, `target`, `destination`, `output_dir`, or `cache_dir`. Execution checks paths again relative to the actual working directory.

These checks do not discover package-manager locks, symlink aliases, shared shell profiles, or undeclared resources. Use sequential execution for installers with those risks. Resource thresholds, cancellation, and per-task timeouts are not implemented.

## Metrics

`analyzePerformance(plan, results)` compares summed plugin durations with the measured execution wall time. Empty plans yield finite values. `optimizeExecution(nodes)` estimates improvement from dependency grouping and historical durations; it does not guarantee a speedup.

CPU and heap deltas sample the Genesis process during a plugin's execution. Concurrent plugins overlap these samples, and subprocess resource usage is not included. Metrics are diagnostic estimates, not isolated per-plugin measurements. `getMetrics()` returns the latest result per plugin; `getExecutionHistory()` returns this engine instance's execution history.
