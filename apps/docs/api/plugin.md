# Plugin API

Plugins are Go values in `packages/core`. Optional hooks are nil function fields.

```go
package example

import "github.com/ossl-dev/genesis/packages/core"

func Plugin() core.Plugin {
    return core.Plugin{
        ID: "example",
        Category: "tool",
        Detect: func(r *core.Runtime, i core.Instance) (core.DetectResult, error) {
            result, err := r.Run("example", "--version")
            return core.DetectResult{OK: result.Code == 0, Details: result.Stdout}, err
        },
    }
}
```

`Plugin` fields are `ID`, `Category`, `DependsOn`, `ParseOptions`, `Prepare`, `RegisterTasks`, `PreApply`, `Apply`, `PostApply`, `Detect`, and `Validate`. Categories are `tool`, `sdk`, `language`, `library`, and `framework`. Dependencies identify configured instances. `ParseOptions` accepts and returns `map[string]any` with an error; loaders validate/default options before machine changes.

Hook signatures:

```go
func(*core.Runtime, core.Instance) error // Prepare/RegisterTasks/PreApply/PostApply
func(*core.Runtime, core.Instance) (core.DetectResult, error)
func(*core.Runtime, core.Instance) (core.ApplyResult, error)
func(*core.Runtime, core.Instance) (core.ValidateResult, error)
```

## Runtime

`Runtime` carries `Context`, `Cwd`, synchronized `Env`, `Logger`, `Tasks`, `Runner`, `Platform`, and `Arch`. `Ensure()` fills absent defaults before execution. Use `Env.Get`, `Set`, `Snapshot`, and `PrependPath`; never mutate an environment map shared across workers. Context cancellation is cooperative inside plugin code.

Use `Run(command, args...)` for commands with runtime cwd/environment, or `RunWithOptions` for a timeout override. Keep system prerequisites in `RegisterTasks`; task execution finishes before apply hooks run. Pass user values as arguments, not interpolated shell source.

## Results and failures

| Method | Result fields |
| --- | --- |
| Detect | `OK`, `Details` |
| Apply | `OK`, `DidChange`, `Details` |
| Validate | `OK`, `Message` |

Verify installed versions after commands succeed. An apply result with `OK: false` skips its post hook and blocks dependents. Sequential hook errors stop execution and retain a failed current summary alongside earlier results. An apply error preserves its returned `DidChange`; a post-apply error preserves the apply result's change flag. Pre-apply errors report no changes. Parallel errors/panics are recorded as failed plugin results. Successful unchanged applies still run their pre/post hooks, so hooks must be repeatable.

A missing detect hook reports `unknown`; missing validation reports success. A missing apply hook succeeds without changes. Supply those methods when meaningful verification is required.

Custom executable Go plugins use [the extension protocol](/guide/plugin-development), which works across supported operating systems. Automatic system-wide rollback remains unfinished; archive installers provide their own staged recovery.
