# Genesis core

Go configuration, plugin lifecycle, task scheduling, scripts/repositories, and command utilities. Import `github.com/ossl-dev/genesis/packages/core`. See [Core API](../../apps/docs/api/core.md) and [Architecture](../../apps/docs/guide/architecture.md).

```go
package main

import (
    "log"
    "os"
    "github.com/ossl-dev/genesis/packages/core"
    "github.com/ossl-dev/genesis/packages/plugins"
)

func main() {
    cwd, err := os.Getwd()
    if err != nil { log.Fatal(err) }
    config, err := core.LoadConfig(cwd, "genesis.config.yaml")
    if err != nil { log.Fatal(err) }
    nodes, err := plugins.Load(config)
    if err != nil { log.Fatal(err) }
    results, err := core.ApplyEnvironment(config, nodes, &core.Runtime{Cwd: cwd})
    if err != nil { log.Fatal(err) }
    for _, result := range results {
        if !result.OK { log.Fatal(result.Details) }
    }
}
```

`ApplyEnvironment` runs before scripts, shared prerequisites and plugin hooks, repositories, then after scripts. Graph errors fail before provisioning. System-task failures stop apply; returned plugin failures block dependents. Existing repositories keep local work. System-wide rollback remains unfinished.

`ApplyEnvironmentParallel` and `ParallelExecutionEngine` offer bounded goroutine execution with dependency layers, context cancellation, synchronized runtime environment updates, and configured path/port conflict checks. Undeclared shared resources still require sequential execution.

`EnvironmentCacheManager` stores metadata in memory, with JSON/gzip transport and SHA-256 validation. Filesystem/network restore, encryption, remote synchronization, and durable reload remain unfinished and return explicit unsupported errors. CLI apply does not populate backups.

From the repository root: `go test -race ./...`, `go vet ./...`, and `go build ./apps/cli`.
