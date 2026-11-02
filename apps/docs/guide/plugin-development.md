# Plugin Development

Use `core.Plugin` for compiled Go plugins. Hooks receive `*core.Runtime` and
`core.Instance`; implement strict `ParseOptions` before any changes. Set
`DependsOn` to configured instance IDs. `plugins.LoadWithResolver` resolves
custom modules; library consumers can also create `core.Node` directly.

```go
plugin := core.Plugin{
    ID: "example", Category: "tool",
    Detect: func(r *core.Runtime, i core.Instance) (core.DetectResult, error) {
        result, err := r.Run("example", "--version")
        return core.DetectResult{OK: result.Code == 0, Details: result.Stdout}, err
    },
}
```

Use runtime cwd/env for commands, context for HTTP/filesystem work, and task
registry for shared prerequisites. Avoid global process environment mutation.
Use `Runtime.RunWithOptions` for explicit timeouts. Archive installs should use
the shared staged installer and published checksums. See [best practices](/plugins/best-practices).

## Executable plugins

A config module `exec:./path/to/plugin` runs an ordinary platform executable,
avoiding platform-specific shared libraries. Build plugins in Go or any language
available on the target host. The executable reads one JSON request from stdin
and emits one JSON response on stdout; logs belong on stderr.

Request fields: `protocol: 1`, `method`, `instance`, `cwd`, and `env`. Supported
methods: `describe`, `parseOptions`, `prepare`, `registerTasks`, `preApply`,
`detect`, `apply`, `postApply`, and `validate`. Every response requires
`protocol: 1`; an `error` string fails the call.

`describe` returns `dependsOn` and a `methods` array advertising implemented optional
hooks; omitted hooks retain the core defaults. `parseOptions` returns normalized `options`.
Lifecycle responses may return an `env` delta. Detection/apply/validation return
`detect: {ok, details}`, `apply: {ok, didChange, details}`, or
`validate: {ok, message}`. Empty hook responses still include protocol version.
`registerTasks` can return a `tasks` array with `id`, `description`, `priority`,
`dependsOn`, `command`, `args`, and optional `timeoutSeconds` (0–86400).

Responses are bounded to 8 MiB; each call has a deadline and respects caller
cancellation. Calls use separate processes, so preserve state through files or
explicit output rather than process globals. Configs and plugins are trusted code;
loading metadata and parsing options also execute the plugin during dry-run.
