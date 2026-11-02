# Contributing

Keep changes focused, preserve platform behavior, update relevant documentation,
and use the existing conventional commit style.

```sh
go test ./...
go test -race ./...
go vet ./...
go run ./apps/docs -output dist/docs
```

Format Go with `gofmt`. Prefer argument arrays, context cancellation, streamed
I/O, bounded goroutines, and explicit ownership of files/processes. Validate
plugin options before changes. Preserve shared-task deduplication and staged
installer recovery. Add regression tests for meaningful failure modes; use real
temporary-file/subprocess integration checks where mocks could hide behavior.

Native platform CI and cross builds are separate evidence. Explain installer
coverage limits; do not label mocked commands as successful host provisioning.
See [plugin development](/guide/plugin-development) and [architecture](/guide/architecture).
