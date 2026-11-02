# Genesis plugins

The Go package `github.com/ossl-dev/genesis/packages/plugins` contains eleven
builtins and shared verified installers. `Load(config)` validates options and
dependencies; `LoadWithResolver` adds custom plugin resolution.

```go
nodes, err := plugins.Load(core.Config{Tools: []core.Instance{{
    ID: "node", Category: "tool", Module: "node",
    Options: map[string]any{"version": "22", "use_nvm": false},
}}})
```

Use the [plugin reference](../../apps/docs/plugins/overview.md) for exact options
and platform limits. Checksummed archives and npm prefixes use staged installs,
verification before/after promotion, destination locks, and recovery on failure.
Published metadata and release versions are validated before downloads. Package
manager operations retain their native host effects and have no general rollback.

Run `go test -race ./packages/plugins` from repository root. Real installer
checks use `go run ./scripts/smoke installers`, temporary prefixes, and network
access. Tests and native CI evidence are distinct from all-platform guarantees.
