# Genesis Roadmap

Things to build, fix, and improve. Checked boxes mean shipped.

**Want to contribute?** Pick an unchecked box, open an issue saying you're on it, send a PR. Keep one item per PR. If something's unclear, open an issue and ask — don't guess.

---

## Phase 1 — Ship what we have

Stuff that's built but not fully finished, tested, or released.

### Tests

Genesis has zero tests. That's the first thing to fix.

- [ ] Add unit tests for config loader — YAML parsing, TS config loading, validation edge cases
- [ ] Add unit tests for task registry — dedup, topological sort, priority ordering
- [ ] Add unit tests for plugin executor — 3-phase lifecycle, error propagation
- [ ] Add unit tests for each plugin's `detect`, `apply`, `validate` methods (mock the shell)
- [ ] Add integration tests — apply a real config in a temp dir, verify side effects
- [ ] Add platform-specific test runs (macOS, Linux, Windows) in CI

### CI / infra

- [ ] Wire up GitHub Actions — lint, build, test on push/PR
- [ ] Add test matrix for Node 18/20/22 and Bun 1.x
- [ ] Add CI badge to README
- [ ] Add per-platform CI jobs (macOS, Ubuntu, Windows runners)
- [ ] Add `turbo run lint` to CI — eslint is configured but nothing enforces it
- [ ] Add dependency vulnerability check (e.g. `npm audit` or `bun audit`)

### Finish stubs

Code exists but doesn't actually do the thing yet. Finish it.

- [ ] **ParallelExecutionEngine** — the class is wired up but `getMemoryUsage` returns 0, `detectResourceConflicts` returns empty, `isCriticalPath` always returns true. Make it do real resource-aware parallel scheduling.
- [ ] **EnvironmentCacheManager** — full interface exists, but `restore`, `decompress`, `sync`, `snapshot` methods only log. Implement actual I/O so `genesis` can cache and restore dev environments.
- [ ] **genesis login** — CLI command exists, emits placeholder output. Implement OAuth flow and token storage.
- [ ] **genesis list --cloud** — exists, prints hardcoded example output. Wire to real backend.
- [ ] **genesis apply <env-id>** — exists, prints hardcoded output. Wire to real cloud environment apply.

### Docs accuracy

The docs site lists 12+ plugins in the sidebar that don't exist yet. Fix the mismatch.

- [ ] Audit every doc page against real implemented code — remove or mark aspirational pages clearly
- [ ] Add a "status" badge per plugin doc page (implemented / planned / in progress)
- [ ] Add missing API reference pages for modules that do exist but aren't documented
- [ ] Add a troubleshooting page (common issues, platform quirks, config gotchas)

---

## Phase 2 — Core improvements

Make existing stuff faster, safer, more portable.

### Plugin lifecycle

- [ ] Add `preApply` / `postApply` hooks per plugin (some plugins need to run before/after others)
- [ ] Add plugin rollback on failure — if plugin C fails, undo plugin B's changes
- [ ] Add plugin config validation at load time, not just at apply time
- [ ] Support plugin dependencies — plugin A must apply before plugin B

### Platform support

- [ ] **Windows** — every plugin currently falls back to printing manual install guides. Auto-install for at least Homebrew (via winget), Node (via nvm-windows or fnm), Python, and Git.
- [ ] **Linux** — test on Debian, Fedora, Arch. Package manager detection works (apt/yum/dnf/pacman) but only apt is actually used in plugin code.
- [ ] **aarch64 / ARM** — Docker plugin downloads x86 binaries on ARM. Go download URL doesn't auto-detect arch. Fix arch detection everywhere.
- [ ] Add `genesis doctor` to actually check that required system tools exist and report what's missing (currently just prints config example)

### Execution

- [ ] Make `ParallelExecutionEngine` configurable — max concurrency, resource thresholds, timeout per task
- [ ] Add task progress reporting — show what's running, what's done, what failed
- [ ] Add dry-run mode that shows what would happen without executing anything
- [ ] Export execution plan as JSON for CI tooling to consume

### Config

- [ ] Add config file merging — load `genesis.config.yaml` + override with env-specific partials
- [ ] Add config file watching / hot-reload for `genesis apply --watch`
- [ ] Allow config values to reference environment variables (`${HOME}`, `${USER}`, etc.)
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

- [ ] **Homebrew on Apple Silicon**: install path is `/opt/homebrew`, not `/usr/local`. Plugin assumes the latter for cask installs.
- [ ] **Docker Desktop on macOS**: plugin offers Colima then Docker Desktop as alternatives, but Docker Desktop requires user to accept license interactively — breaks non-interactive apply.
- [ ] **Go arch detection**: download URL hardcodes `amd64`. On Apple Silicon it downloads the wrong binary.
- [ ] **Node standalone install**: `use_nvm: false` prints "standalone installation not yet supported" and skips. Should at least try fnm or a direct download.
- [ ] **Parallel execution with one core**: `ParallelExecutionEngine` doesn't check available CPUs — could oversubscribe a low-resource machine.
- [ ] **Config validation error messages**: Zod errors are printed raw without context. "Expected number, got string" on a deeply nested field is hard to debug — need path + friendly message.
- [ ] **Plugin loading silently fails**: if a plugin module throws during import, it's caught and logged as debug. Should surface in `genesis doctor` at minimum.
- [ ] **No config file caching**: every `genesis apply` re-parses and re-validates the config. Add hash-based skip when nothing changed.
- [ ] **`genesis diff` output**: currently prints the full config on both sides. Should show only what changed.
- [ ] **Windows path separators**: code uses `path.join` in some places but string concatenation in others (e.g. `download.ts`). Will break on Windows.

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
