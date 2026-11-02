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

Use loaded `[]core.Node` instead of `nil` for real work. Planning rejects graph errors before execution. Worker counts must be positive and are capped at `GOMAXPROCS`. A dependency phase finishes before its dependents begin. Results retain plan order, regardless of completion order. Single-worker phases execute directly; worker goroutines and job channels are created only when a phase can run concurrently.

Prepare hooks, task registration, and shared system tasks run sequentially before workers start. Parallel plugin errors and panics become failed results; dependent plugins are skipped, while independent work can finish. Runtime environment updates use synchronized `Environment` methods.

## Resource checks and cancellation

A phase becomes sequential when options overlap paths in `path`, `install_dir`, `home`, `prefix`, `target`, `destination`, `output_dir`, or `cache_dir`, or share a port. Execution checks relative paths again against the actual working directory.

These checks cannot infer symlink aliases, package-manager locks, shell-profile mutations, or undeclared resources. Keep plugins sharing those resources sequential. The engine does not enforce RAM/CPU thresholds.

`Runtime.Context` propagates cancellation into hooks and child commands. Plugins must honor it. Unix command cancellation kills the child process group; Windows cancellation requests tree termination and kills the child. `CommandOptions.Timeout` bounds individual commands. Worker goroutines finish before execution returns. Cancellation returns the original context error even in the final task or plugin phase, so callers can use `errors.Is` with `context.Canceled` or `context.DeadlineExceeded`. Completed plugin results remain available alongside that error.

## Metrics

`AnalyzePerformance(plan, results)` compares summed plugin durations with measured execution time. `OptimizeExecution(nodes)` estimates improvement using prior durations; estimates do not guarantee speedup. Durations are Go `time.Duration` values.

`PluginMetrics.MemoryUsage` samples positive changes in the Genesis process's heap object bytes. `CPUUsage` estimates process user CPU seconds divided by plugin wall time using Go runtime counters. Samples overlap during concurrent work, may update coarsely, and exclude subprocess usage. Zero is possible without a sampled increase. These are process diagnostics, not isolated plugin accounting.

`GetMetrics()` returns the latest metrics per plugin. `GetExecutionHistory()` returns retained results; callers reusing an engine can release history through `ClearHistory()` and estimates through `ResetMetrics()`. The CLI creates an engine per invocation.

## Scheduling overhead

Graph traversal indexes nodes once and keeps visit state in a byte slice. Sequential apply reuses validated order between its internal lifecycle steps; public entry points still validate dependencies before provisioning.

Local medians from five benchmark runs on an Apple M2 Pro, darwin/arm64, Go 1.27.3:

| Operation | Before | After | Allocated bytes before → after | Allocations before → after |
| --- | --- | --- | --- | --- |
| Graph, 11 nodes | 1,836 ns | 595 ns | 4,896 → 2,504 | 18 → 4 |
| Apply, 11 nodes | 6,375 ns | 2,098 ns | 16,056 → 6,376 | 63 → 17 |
| Graph, 1,000 nodes | 232,999 ns | 52,071 ns | 573,745 → 235,858 | 1,041 → 7 |
| Apply, 1,000 nodes | 791,710 ns | 189,491 ns | 1,887,616 → 638,091 | 3,149 → 40 |

These benchmarks use dependency chains with no-op plugins and an empty runtime environment. They measure core overhead, excluding installers, network, and child processes. Allocation profiling attributed 91% of baseline apply allocations to graph construction. Actual provisioning time depends on the requested tools and host.

Run `go test ./packages/core -run '^$' -bench 'Benchmark(BuildPluginGraph|ApplyEnvironment)$' -benchmem -count=5` to compare on another host.

A separate 100-node dependency-chain benchmark of parallel execution (four-worker limit, three-run medians on the same host) improved from 272,943 to 116,048 ns/op, from 237,656 to 215,246 B/op, and from 1,538 to 1,238 allocations/op after removing worker/channel setup for serialized phases. This fixture includes metrics collection, reuses its plan, and clears history each iteration. Run `go test ./packages/core -run '^$' -bench '^BenchmarkParallelDependencyChain$' -benchmem -count=3` to reproduce.
