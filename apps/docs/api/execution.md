# Parallel Execution API

Core uses bounded goroutine workers and dependency phases. CLI apply defaults to one worker; `--parallel` enables parallel execution.

```go
package main

import (
    "log"
    "github.com/ossl-dev/genesis/packages/core"
)

func main() {
    engine, err := core.NewParallelExecutionEngine(2)
    if err != nil { log.Fatal(err) }
    plan, err := engine.CreateExecutionPlan(nil)
    if err != nil { log.Fatal(err) }
    results, err := engine.ExecuteParallel(plan, &core.Runtime{})
    if err != nil { log.Fatal(err) }
    for _, result := range results {
        if !result.Success { log.Fatal(result.Error) }
    }
}
```

Use loaded `[]core.Node` instead of `nil` for real work. Planning rejects graph errors before execution. Worker counts must be positive and are capped at `GOMAXPROCS`. A dependency phase finishes before its dependents begin. Results retain plan order, regardless of completion order.

Prepare hooks, task registration, and shared system tasks run sequentially before workers start. Parallel plugin errors and panics become failed results; dependent plugins are skipped, while independent work can finish. Runtime environment updates use synchronized `Environment` methods.

## Resource checks and cancellation

A phase becomes sequential when options overlap paths in `path`, `install_dir`, `home`, `prefix`, `target`, `destination`, `output_dir`, or `cache_dir`, or share a port. Execution checks relative paths again against the actual working directory.

These checks cannot infer symlink aliases, package-manager locks, shell-profile mutations, or undeclared resources. Keep plugins sharing those resources sequential. The engine does not enforce RAM/CPU thresholds.

`Runtime.Context` propagates cancellation into hooks and child commands. Plugins must honor it. Unix command cancellation kills the child process group; Windows cancellation requests tree termination and kills the child. `CommandOptions.Timeout` bounds individual commands. Worker goroutines finish before execution returns.

## Metrics

`AnalyzePerformance(plan, results)` compares summed plugin durations with measured execution time. `OptimizeExecution(nodes)` estimates improvement using prior durations; estimates do not guarantee speedup. Durations are Go `time.Duration` values.

`PluginMetrics.MemoryUsage` samples positive changes in the Genesis process's heap object bytes. `CPUUsage` estimates process user CPU seconds divided by plugin wall time using Go runtime counters. Samples overlap during concurrent work, may update coarsely, and exclude subprocess usage. Zero is possible without a sampled increase. These are process diagnostics, not isolated plugin accounting.

`GetMetrics()` returns the latest metrics per plugin. `GetExecutionHistory()` returns retained results; callers reusing an engine can release history through `ClearHistory()` and estimates through `ResetMetrics()`. The CLI creates an engine per invocation.
