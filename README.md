# Genesis

[![CI](https://github.com/ossl-dev/genesis/actions/workflows/ci.yml/badge.svg)](https://github.com/ossl-dev/genesis/actions/workflows/ci.yml)

Rebuild developer environments from a versioned configuration: tools, runtimes,
repositories, environment variables, and setup scripts. Genesis is written in Go
and builds into a standalone binary. Local provisioning is the current focus;
cloud services and complete environment restoration remain unfinished.

```sh
go build -trimpath -o bin/genesis ./apps/cli
./bin/genesis --help
```

Go 1.23 or newer builds the source; use a supported Go release for production.
The resulting binary needs no language runtime. Cross builds cover macOS, Linux,
and Windows on amd64 and arm64. Native installer support varies by plugin.

```yaml
tools:
  - type: node
    version: "22"
    use_nvm: false
  - type: git
languages:
  - type: go
    version: "1.22.5"
repositories:
  - url: https://github.com/example/project.git
    folder: project
scripts:
  - name: verify
    command: git --version
    when: after
env:
  PROJECT_ROOT: "${HOME}/projects"
```

From the environment directory, run the built binary by its absolute path:

```sh
/path/to/genesis/bin/genesis apply --dry-run --json
/path/to/genesis/bin/genesis apply
/path/to/genesis/bin/genesis doctor
```

`apply --config path.yml` selects an explicit config. `--parallel 4` enables
bounded goroutines for independent plugins; the default remains sequential.
Dependencies, overlapping installation paths, and shared ports constrain
parallelism. System package prerequisites are deduplicated before plugin apply.
Failed dependencies block dependent plugins. Scripts and repositories run in
configured order; repository reapply preserves existing work and checks origin,
branch, and repository root before accepting an existing destination.

All eleven builtins remain: Node, Bun, Deno, pnpm, Yarn, Python, Go, Java, Git,
Docker, and Homebrew. Archive installers verify published SHA256 checksums,
stage and verify installations, and retain recoverable backups when rollback
fails. Linux shared tasks support APT, DNF, pacman, and APK; Docker official
repositories support Ubuntu, Debian, Fedora, CentOS, and RHEL. Docker Desktop
and unavailable Windows tools report manual setup requirements.

YAML is the portable default. Executable Go configurations emit JSON; compiled
Go plugins and executable plugins replace runtime module imports. See
[configuration](apps/docs/guide/configuration.md) and
[plugin development](apps/docs/guide/plugin-development.md).

Only run trusted configurations and plugins. Dry-run skips provisioning hooks;
Go configs and external plugin validation still execute configured code.
Installers modify the host. System package operations have no general rollback.

```sh
go test ./...
go test -race ./...
go vet ./...
go run ./apps/docs -output dist/docs
go run ./apps/docs -serve localhost:8080
go run ./scripts/release
```

Source layout: `apps/cli`, `apps/docs`, `packages/core`, `packages/plugins`,
`packages/extension`, and `scripts`. One Go module replaces workspace build
orchestration. Documentation builds and serves through Go; server search works
without browser scripting. Static exports support navigation and browser find.

Real downloads and installer reapply checks: `go run ./scripts/smoke installers`.
They use temporary prefixes and need network access plus source-build tools for
Git. Linux package/Docker smoke modes require an explicitly disposable root
host. CI exercises three operating systems and Linux distro containers; local
cross-compilation is not evidence of native installer execution.

See [roadmap](ROADMAP.md), [migration record](MIGRATION.md), and
[platform support](apps/docs/guide/platform-support.md). Cache metadata APIs remain
available, but artifact/network restoration, OAuth, and cloud apply are not
implemented. Login stores local token metadata without verifying cloud access.
