# Genesis

[![CI](https://github.com/ossl-dev/genesis/actions/workflows/ci.yml/badge.svg)](https://github.com/ossl-dev/genesis/actions/workflows/ci.yml)

Genesis provisions development tools, repositories, and setup scripts from a version-controlled config. The goal is to replace repeated onboarding instructions with a reproducible team setup.

It is an early-stage TypeScript/Bun monorepo. Local provisioning is the current focus. Cloud environments, complete environment isolation, cache restore, and published standalone binaries are unfinished.

## Run from source

```bash
bun install --frozen-lockfile
bun run build
bun apps/cli/dist/index.js --help
```

From a project directory, run the built entrypoint by its absolute path:

```bash
bun /path/to/genesis/apps/cli/dist/index.js apply --dry-run
bun /path/to/genesis/apps/cli/dist/index.js apply
```

The entrypoint can also run under Node for YAML configs. TypeScript configs require a runtime with TypeScript loading support; Bun is the supported development path. There is no install endpoint or binary release maintained in this repository.

## Configuration

```yaml
# genesis.config.yaml
tools:
  - type: node
    version: "22"
    use_nvm: true
  - type: git
    install_method: package
repositories:
  - url: https://github.com/your-team/project.git
    folder: ./project
    branch: main
env:
  NODE_ENV: development
  WORKSPACE: "${HOME}/projects"
scripts:
  - name: install-dependencies
    command: cd project && npm install
    when: after
```

Genesis discovers `genesis.config.ts` before `genesis.config.yaml`. `apply --config ./configs/dev.yml` selects an explicit file. YAML supports built-in `type` entries or full plugin instances with `id`, `category`, `module`, and `options`. IDs must be unique. `${NAME}` references use the invoking process's environment; unset variables fail with a field path.

`env` applies to Genesis commands and scripts, not the parent shell. `before` scripts run before plugin provisioning. Repositories clone after successful plugins; `after` scripts run last. Scripts execute every time, so make them safe to repeat. Existing repositories must match the requested origin and branch; local edits are preserved.

## Commands

| Command | Current behavior |
| --- | --- |
| `init`, `create` | Prompt for TS/YAML format and scaffold missing files |
| `apply` | Run scripts, deduplicated system tasks, plugins, and repository setup |
| `apply --dry-run` | List planned actions without provisioning |
| `apply --dry-run --json` | Export the action plan as JSON |
| `diff` | Report plugin detection status; not a package/version change diff |
| `validate` | Run plugin validation; exit nonzero on a failed check |
| `doctor` | Run plugin detection and validation for the current config |
| `list-plugins` | List the seven built-in plugins |
| `list --format json` | List local config/cache metadata |
| `login --token <token>` | Store a token locally; no backend verification or OAuth |

Cloud apply is unavailable and exits nonzero. `list --cloud` only reports the unavailable backend. `GENESIS_DEBUG=1` enables debug logs.

## Plugins and limits

Built-ins: Node, Python, Go, Java, Git, Docker, and Homebrew. Options are validated when plugins load. Missing dependencies, cycles, and duplicate IDs fail before provisioning. Plugins can declare `dependsOn`, `preApply`, and `postApply` hooks.

- macOS uses Homebrew for shared package tasks; install brew before provisioning other tools. Node uses NVM, Docker uses Colima by default, and Go uses an archive.
- Linux shared package tasks use APT. Fedora/Arch support is incomplete.
- Windows detects installed tools, but missing-tool installation generally requires manual steps.
- Java archive release resolution and Git source/binary paths are experimental. Go/Java extraction requires permissions for system directories.
- Docker Desktop needs manual installation and license acceptance. Downloads are retained and apply reports that completion is required.
- Apply is sequential in the CLI. Core consumers can opt into the tested parallel engine; configured path/port checks cannot infer every installer resource conflict.
- Rollback and environment isolation are not implemented. A failed run can leave earlier changes in place.

Installers change the host environment. Only run configs and plugin modules you trust. Dry-run skips provisioning methods, but imported TypeScript configs and plugin modules can execute code while loading.

## Development

```bash
bun run test
bun run lint
bun run build
bun run docs:build
bun run docs:dev
```

Tests cover config parsing, plugin loading/lifecycle, task dependencies, parallel scheduling, mocked installers, real setup scripts, and local Git clones. CI runs across macOS, Linux, and Windows; these tests do not certify every real installer on each platform.

The recent work established unit tests and CI, removed dead utilities, and repaired build/type generation. The current improvements make config selection and failure reporting reliable, add lifecycle validation and hooks, repair parallel scheduling, and execute repositories/scripts with dry-run plans. See [ROADMAP.md](ROADMAP.md) for unfinished work.

- [Configuration](apps/docs/guide/configuration.md)
- [CLI reference](apps/cli/README.md)
- [Core API](apps/docs/api/core.md)
- [Plugins](apps/docs/plugins/overview.md)
- [Architecture](apps/docs/guide/architecture.md)

Report issues at [ossl-dev/genesis](https://github.com/ossl-dev/genesis/issues).
