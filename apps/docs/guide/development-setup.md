# Development Setup

```sh
git clone https://github.com/ossl-dev/genesis.git
cd genesis
go mod download
go test ./...
go test -race ./...
go vet ./...
go build -o bin/genesis ./apps/cli
```

One Go module retains the apps/packages structure for Genesis. Documentation
remains an independent VitePress project: run `cd apps/docs && bun install
--frozen-lockfile` before its dev/build commands. No generated package output
is needed before tests. Source uses standard formatting (`gofmt`); `make check`
runs vet, tests, and formatting checks. Archive tests extract real fixtures;
command mocks cover installer permutations, while Git/scripts and cancellation
also have real subprocess integration tests.

```sh
cd apps/docs && bun run dev
cd apps/docs && bun run build
go run ./scripts/release
```

Installer smoke: `go run ./scripts/smoke installers`. It uses disposable prefixes
and published downloads. Package and Docker modes require disposable Linux root
hosts with `GENESIS_DISPOSABLE_HOST=1`; never run them on a workstation.
