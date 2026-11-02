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
remains an independent VitePress project: run `bun install --cwd apps/docs --frozen-lockfile` before its dev/build commands. No generated package output
is needed before tests. Source uses standard formatting (`gofmt`); `make check`
runs vet, tests, and formatting checks. Archive tests extract real fixtures;
command mocks cover installer permutations, while Git/scripts and cancellation
also have real subprocess integration tests.

```sh
bun run --cwd apps/docs dev
bun run --cwd apps/docs build
go run ./scripts/release
```

Installer smoke: `go run ./scripts/smoke installers`. It uses disposable prefixes
and published downloads. Package and Docker modes require disposable Linux root
hosts with `GENESIS_DISPOSABLE_HOST=1`; never run them on a workstation.

## Profiling

Core and release-parser benchmarks isolate Go overhead from downloads and child
programs. Compare several runs on the same host before changing concurrency or
memory limits:

```sh
go test ./packages/core -run '^$' -bench 'Benchmark(BuildPluginGraph|ApplyEnvironment|ParallelDependencyChain)$' -benchmem -count=5
go test ./packages/plugins -run '^$' -bench '^BenchmarkGoReleaseManifest$' -benchmem -count=5
```

The release fixture is 2.16 MiB: streaming reduced allocated bytes by about 97%
and allocation count by 20%, with about 4% more decoder time than buffered
parsing in local measurements. It validates the entire manifest and retains
size/deadline limits. These are parser measurements, not installer speed claims.
See [execution benchmarks](/api/execution) for graph and worker measurements.
