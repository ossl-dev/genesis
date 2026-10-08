# Genesis Roadmap

Things to build, fix, and improve. Checked boxes mean implemented in this repository, not a published release. Installer tests mostly mock commands; OS CI coverage does not certify fresh-machine provisioning.

## Next priorities

1. Extend verified staged installation beyond Go/Java; correct Git release resolution and version pinning in Docker installers.
2. Validate Homebrew bootstrap and update behavior on disposable macOS hosts.
3. Add real installer checks in disposable environments, then implement Linux package-manager selection and Windows installers.
4. Finish local cache persistence/restore before cloud sync or standalone distribution.

Recent work adds trustworthy config/failure handling, validated lifecycle hooks, a tested parallel core API, and real repository/script apply with dry-run plans. CLI plugin execution remains sequential.

**Want to contribute?** Pick an unchecked box, open an issue saying you're on it, send a PR. Keep one item per PR. If something's unclear, open an issue and ask — don't guess.

---

## Phase 1 — Ship what we have

Stuff that's built but not fully finished, tested, or released.

### Tests

Unit tests cover configs, plugins, task scheduling, and CLI behavior. Local integration tests exercise scripts and Git repositories in temporary directories.

- [x] Add unit tests for config loader — YAML parsing, TS config loading, validation edge cases
- [x] Add unit tests for task registry — dedup, topological sort, priority ordering
- [x] Add unit tests for plugin executor — 3-phase lifecycle, error propagation
- [x] Add unit tests for each plugin's `detect`, `apply`, `validate` methods (mock the shell)
- [x] Add integration tests — execute real scripts and local Git clones in temp directories; cover failures and preservation of existing files
- [x] Add platform-specific test runs (macOS, Linux, Windows) in CI

### CI / infra

- [x] Wire up GitHub Actions — lint, build, test on push/PR
- [x] Add Bun test runs across three OS; Node versions are not separately pinned in CI
- [x] Add CI badge to README
- [x] Add per-platform CI jobs (macOS, Ubuntu, Windows runners)
- [x] Add `turbo run lint` to CI — currently TypeScript checks, not ESLint rules
- [x] Add advisory `bun audit` job (failures do not block CI)

### Finish stubs

Code exists but doesn't actually do the thing yet. Finish it.

- [x] **ParallelExecutionEngine** — preserve executable plugin nodes, schedule dependency layers with bounded workers, clamp concurrency to available CPUs, serialize overlapping configured paths/ports, and propagate failures. Resource checks are conservative heuristics; CLI apply remains sequential.
- [ ] **EnvironmentCacheManager** — metadata/compression prototype exists; filesystem/network restore and durable reload are unfinished, and "remote" sync only writes local JSON. Implement persistence, safe artifact restore, and round-trip tests.
- [ ] **genesis login** — token metadata can be saved locally, but backend verification and OAuth are missing.
- [ ] **genesis list --cloud** — reports an unavailable backend. Wire to a real service without contaminating JSON output.
- [ ] **genesis apply <env-id>** — fails explicitly. Implement authenticated environment resolution and apply.

### Docs accuracy

Keep shipped behavior distinct from experiments and proposals; remove unsupported commands and fabricated API examples.

- [x] Audit every doc page against real implemented code — remove or mark aspirational pages clearly
- [x] Add a "status" badge per plugin doc page (implemented / planned / in progress)
- [x] Add missing API reference pages for modules that do exist but aren't documented
- [x] Add a troubleshooting page (common issues, platform quirks, config gotchas)

---

## Phase 2 — Core improvements

Make existing stuff faster, safer, more portable.

### Plugin lifecycle

- [x] Add `preApply` / `postApply` hooks per plugin (some plugins need to run before/after others)
- [ ] Add plugin rollback on failure — if plugin C fails, undo plugin B's changes
- [x] Add plugin config validation at load time, not just at apply time
- [x] Support plugin dependencies — plugin A must apply before plugin B

### Platform support

- [ ] **Windows** — Go/Java archive installers are implemented. Add native installers for Node, Python, and Git through supported Windows package/version managers.
- [ ] **Linux** — test on Debian, Fedora, Arch. Shared system tasks currently hardcode APT; implement distribution/package-manager selection and package mappings.
- [ ] **aarch64 / ARM** — Go and Docker Desktop map ARM64 correctly. Add real ARM installer checks and reject unsupported architectures rather than falling back to x64.
- [x] `genesis doctor` detects and validates configured plugins with nonzero failure exits
- [ ] Add config-free host diagnostics and prerequisite inventory to doctor

### Execution

- [ ] Make `ParallelExecutionEngine` configurable — max concurrency, resource thresholds, timeout per task
- [ ] Add task progress reporting — show what's running, what's done, what failed
- [x] Add dry-run mode that shows what would happen without executing anything
- [x] Export execution plan as JSON for CI tooling to consume

### Config

- [ ] Add config file merging — load `genesis.config.yaml` + override with env-specific partials
- [ ] Add config file watching / hot-reload for `genesis apply --watch`
- [x] Allow config values to reference environment variables (`${HOME}`, `${USER}`, etc.)
- [ ] Add `$schema` field to generated config for editor autocomplete

---

## Phase 3 — More plugins

Plugins that exist as doc pages but have zero implementation code.

### Languages & runtimes

- [ ] **Bun plugin** — detect existing install, install via curl/brew, set up shell completions
- [ ] **Deno plugin** — detect, install via curl/brew, manage DENO_INSTALL
- [ ] **Rust plugin** — detect rustup/rustc/cargo, install via rustup, manage toolchains
- [ ] **pnpm / Yarn plugin** — install via npm/brew, manage global packages
- [ ] **C++ build tools** — detect g++/clang/MSVC, install Xcode CLT / build-essential
- [ ] **Swift plugin** — detect swiftc, install Xcode or Swift toolchain for Linux

### Databases & services

- [ ] **PostgreSQL plugin** — detect psql/postgres, install via brew/apt/choco, create default user/db
- [ ] **Redis plugin** — detect redis-server, install via brew/apt/choco
- [ ] **MongoDB plugin** — detect mongod, install via brew/apt/choco
- [ ] **Docker Compose plugin** — detect compose, install alongside Docker or standalone
- [ ] **Nginx plugin** — detect nginx, install, set up basic config

### Mobile & frontend

- [ ] **Android SDK plugin** — detect Java + Android SDK, install via sdkmanager or Android Studio CLI
- [ ] **Expo / React Native plugin** — detect expo-cli, install Node + Watchman + Xcode tools

### DevOps & observability

- [ ] **FFmpeg plugin** — detect ffmpeg, install via brew/apt
- [ ] **Grafana / Prometheus plugin** — detect, install, configure basic dashboards
- [ ] **Terraform / OpenTofu plugin** — detect, install, manage versions

---

## Phase 4 — Cloud & collaboration

Genesis as a shared dev environment platform.

### Cloud environments

- [ ] Design and build the cloud backend (API server, DB, auth)
- [ ] **genesis push** — upload current env config + state to cloud
- [ ] **genesis pull** — download and apply a teammate's shared env
- [ ] **genesis apply <env-id>** — apply a cloud-hosted env by short ID
- [ ] **genesis list --cloud** — list shared environments with metadata
- [ ] **genesis diff <env-id>** — diff local vs cloud env

### Auth & teams

- [ ] **genesis login** — real OAuth flow (GitHub, GitLab, or email)
- [ ] Team workspaces — share envs with team members, not just public
- [ ] Environment versioning — each push creates a versioned snapshot

### Sharing

- [ ] **genesis create --share** — interactive create that publishes to cloud
- [ ] Environment templates — "start from this team's standard dev setup"
- [ ] Public gallery — browse community env configs (keep it simple, no social features)

---

## Phase 5 — Polished experience

Make Genesis feel like a mature tool.

### Standalone binary

- [ ] Ship `genesis` as a single binary (Bun compile or `pkg`), no Node/Bun required
- [ ] Self-bootstrapping — the binary should install its own runtime if needed
- [ ] Homebrew formula / winget manifest / apt repo for distribution

### IDE integration

- [ ] VS Code extension — detect `.genesis/`, show current env status in status bar
- [ ] Dev Containers integration — generate `.devcontainer.json` from genesis config
- [ ] GitHub Codespaces / Gitpod support — bootstrap dev env from genesis config on workspace start

### CI/CD integration

- [ ] **genesis ci** — non-interactive mode for CI pipelines (no prompts, colored but pipe-safe output)
- [ ] **genesis ci --check** — verify CI env matches config, exit non-zero on mismatch
- [ ] GitHub Action — setup genesis as a CI step, sync env before running tests
- [ ] Docker image — `genesis apply` inside a container to provision build tools

### Quality of life

- [ ] Shell completion — `genesis completion` for bash, zsh, fish
- [ ] Progress bars and spinners during apply (not just log lines)
- [ ] `--verbose` / `--quiet` / `--json` output flags
- [ ] Colored `genesis diff` — green for new packages, red for removed, yellow for version bumps
- [ ] Telemetry (opt-in) — track apply success/failure rates to surface flaky plugins

---

## Bugs & known issues

Not triaged into phases. Fix anytime.

- [x] Skip system prerequisite tasks when the requested runtime is already installed.
- [x] Compare version components instead of accepting misleading string prefixes.
- [x] Make installed NVM Node available to subsequent commands and install global packages after runtime setup.
- [x] Detect modern Docker Compose, use Debian repository URLs, and check daemon availability without pulling a test image.
- [x] Resolve Java archive releases and build numbers; stage extraction without selecting unrelated JDK directories.
- [x] Replace Go installations through staged extraction with recovery on failure.
- [ ] Enforce version selection in Git source/binary and Docker installation paths.
- [x] Propagate Homebrew update/upgrade failures and bootstrap brew before shared system tasks.

- [x] **Homebrew on Apple Silicon**: choose `/opt/homebrew`, use `/usr/local` on Intel, and expose the install bin directory to later commands.
- [x] **Docker Desktop on macOS**: retain the downloaded DMG and report manual completion instead of claiming a successful install.
- [x] **Go arch detection**: map ARM64 to arm64 and x64 to amd64 in archive URLs.
- [ ] **Node standalone install**: `use_nvm: false` prints "standalone installation not yet supported" and skips. Should at least try fnm or a direct download.
- [x] **Parallel execution with one core**: `ParallelExecutionEngine` doesn't check available CPUs — could oversubscribe a low-resource machine.
- [x] **Config validation error messages**: Zod errors are printed raw without context. "Expected number, got string" on a deeply nested field is hard to debug — need path + friendly message.
- [x] **Plugin loading errors**: report the plugin ID and module when import fails; load-time option errors also include the ID.
- [ ] **No config file caching**: every `genesis apply` re-parses and re-validates the config. Add hash-based skip when nothing changed.
- [ ] **`genesis diff` output**: currently reports plugin detection status. Add desired/current version and package changes.
- [x] **Windows test paths**: real integration tests use native paths; mocked config fixtures use portable virtual paths. The old `download.ts` utility was removed.

---

## Ideas / maybe someday

Not committed. Brainstorming parking lot.

- **Dotfile management**: manage `.zshrc`, `.gitconfig`, `.editorconfig` from genesis config
- **Project templates**: `genesis init --template nextjs-starter` that sets up Node, Postgres, Redis, and clones a repo in one command
- **Dev environment as a service**: ephemeral cloud dev environments spun up from a genesis config (competitor to Gitpod/CodeSandbox but CLI-driven)
- **Plugin marketplace**: community-contributed plugins installable via `genesis plugin add <name>`
- **VS Code Remote Containers interop**: generate `.devcontainer/devcontainer.json` with all the tools in the genesis config
- **Ansible/Puppet/Chef export**: `genesis export --format ansible` to generate a playbook from your env state
- **Dockerfile generation**: `genesis export --dockerfile` to produce a dev container Dockerfile
- **System-wide rollback**: "undo last apply" that uninstalls everything that was added. Track per-package install evidence to make this safe.
- **Mac App Store / Windows Store distribution**: sandboxed entry point for non-CLI users
- **AI-assisted config generation**: describe your stack in plain English, get a `genesis.config.yaml` back
- **Nix flake backend**: use Nix as the underlying installer instead of system package managers, for reproducibility
