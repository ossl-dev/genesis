# Task Registry API

`core.TaskRegistry` deduplicates shared system operations by exact ID within one runtime.

```go
package example

import (
    "context"
    "github.com/ossl-dev/genesis/packages/core"
)

func Register(registry *core.TaskRegistry) error {
    return registry.Register(core.Task{
        ID: "example:check",
        Description: "Check prerequisite",
        Priority: 100,
        Execute: func(ctx context.Context) (core.TaskResult, error) {
            return core.TaskResult{OK: true}, ctx.Err()
        },
    })
}
```

`Task` has `ID`, `Description`, `Priority`, `DependsOn []string`, and `Execute func(context.Context) (TaskResult, error)`. `TaskResult` has `OK`, `Details`, and `Error`.

| Method | Behavior |
| --- | --- |
| `NewTaskRegistry()` | Create an empty registry |
| `Register(task)` | Keep the first task per ID; reject missing ID/executor |
| `Has(id)` | Check registration |
| `Status(id)` | pending/running/completed/failed/not-found |
| `Result(id)` | Result plus availability boolean |
| `IDs()` | Registration order |
| `ExecuteAll(ctx)` | Dependency/priority order; results map plus error |
| `Clear()` | Remove registrations and results |

Dependencies override priority. Higher priority starts traversal earlier; ties preserve registration order. Missing dependencies and cycles fail before any task executes. Failed dependencies skip child tasks. Executor errors and panics become failed task results. Context cancellation returns an error. Repeated execution reuses results; concurrent execution calls serialize. Register tasks before execution, then create/clear a registry for a new run.

## System helpers

```go
func RegisterPackageTasks(r *Runtime, packages ...string) error
func CreatePackageManagerUpdateTask(r *Runtime) (Task, error)
func CreatePackageInstallTask(r *Runtime, name string) (Task, error)
func PackageManagerTaskID(platform, manager, operation, packageName string) string
func CreateCommandCheckTask(r *Runtime, name string) Task
```

`RegisterPackageTasks` registers both metadata and install tasks; exact IDs deduplicate shared requests. Install tasks depend on metadata tasks. `CreateCustomTask` prefixes IDs with `<platform>:custom:` and accepts explicit priority/dependencies. Dependency IDs do not support wildcards.

Shared packages use brew on macOS, Chocolatey on Windows, or APT/DNF/pacman/APK selected from Linux `os-release`. Common compiler/TLS/cURL/zlib/Expat packages map to each distribution's names. Unknown distributions and malformed package names fail early. Linux root runs without sudo.

Metadata tasks refresh brew/APT/DNF/APK. pacman keeps its existing database: maintain Arch hosts with `pacman -Syu` before provisioning. Chocolatey resolves metadata during installs. Shared tasks never perform host-wide upgrades.
