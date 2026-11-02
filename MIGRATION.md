# Go migration

Goal: preserve Genesis behavior in a standalone Go CLI, retaining `apps` and
`packages`, with no JavaScript build or runtime dependency for Genesis itself.
Documentation is independent: retain VitePress, `config.mts`, its theme, and its
own manifest/lockfile under `apps/docs`. Tool plugins can still
provision JavaScript runtimes for user projects.

## Phases and acceptance gates

1. Inventory the existing source and 431-test baseline. Record every command,
   option, lifecycle hook, platform installer, safety boundary, and unfinished API.
2. Port core configuration, task deduplication, dependency scheduling, command
   execution, environment application, and cache metadata APIs. Exercise failures,
   cancellation, concurrency, repository preservation, and environment isolation.
3. Port all eleven plugins and shared installers. Preserve strict options,
   published checksums, stable version selection, platform mappings, staged
   verification, rollback, and subsequent-command environment propagation.
4. Port CLI commands, interactive scaffolding, custom configurations and plugin
   extensibility. Preserve YAML shorthand and full entries, JSON output and exit
   codes, local credentials and environment metadata, and truthful cloud errors.
5. Replace smoke scripts, application CI, and release builds with Go. Keep
   VitePress documentation independent, with its own build and dependencies.
   Remove source, generated output, manifests, dependencies, and caches belonging
   to the old toolchain only after replacement checks pass.
6. Run unit/integration/race tests, vet, six-platform cross builds, real disposable
   installer checks, VitePress validation, and runtime forbidden-artifact scans.

Each completed phase is committed using the repository's conventional messages.
Native cross-platform CI results remain distinct from local cross-compilation.

## Compatibility decisions

- YAML remains the portable default. Executable TypeScript configuration becomes
  executable Go configuration, printing a JSON configuration on stdout. Explicit
  `.go` configurations require a Go compiler; YAML needs only the Genesis binary.
- Dynamic JavaScript module plugins become compiled Go plugins registered through
  the public API, or external executable plugins over a versioned JSON protocol.
  Go shared libraries are avoided because they are not portable to Windows.
- Existing builtin module identifiers remain accepted configuration aliases.
- Documentation remains Markdown in VitePress. Restore the existing `config.mts`
  and theme; isolate its package manager dependencies from the Go application.
- Cloud/OAuth and filesystem/network cache restoration were unimplemented before
  migration. Preserve metadata APIs and explicit limitations; never claim those
  operations succeeded.
- Bounded goroutines operate on dependency-ready plugins only. Shared package
  tasks remain deduplicated; environment mutation is synchronized. Cancellation
  waits for workers and terminates child commands before returning.

## Feature inventory

| Area | Required behavior |
| --- | --- |
| Config | YAML/full and shorthand entries, explicit paths, Go configs, environment interpolation, duplicate IDs and contextual validation |
| Plugins | Strict options/defaults, prepare/register/pre/apply/post/detect/validate, dependency ordering, missing/cyclic dependency errors |
| Tasks | Exact deduplication, priorities, dependency ordering, blocked dependents, failure details |
| Execution | Sequential and bounded parallel modes, resource conflicts, plans, metrics/history, cancellation |
| Environment | Before/after scripts, configured env, safe repository clone/origin/branch/root checks, dry run |
| Installers | Node/NVM, Bun, Deno, Go, Java/JDK, pnpm, Yarn Classic/modern, Git/source/MinGit, Python/venv/pip, Homebrew, Docker/Compose/Colima |
| Platforms | macOS/Linux/Windows, amd64/arm64, APT/DNF/pacman/APK, musl constraints, explicit unsupported cases |
| Archive safety | Streaming SHA256, bounded downloads/extraction, traversal/link defenses, locking, staged/promotion verification and recoverable backups |
| CLI | create/init/apply/doctor/diff/validate/list-plugins/login/list, flags, JSON, honest nonzero exits |
| Metadata | Private token files, local listings, cache import/export/compression/snapshot/statistics and explicit restore limitations |
| Delivery | Go binary, independent VitePress docs, Go smoke runners, three-OS CI, Linux distro fixtures, release cross builds |

## Verification record

Implementation and verification evidence is appended as phases complete. The Git
history retains the original implementation and tests for audit.

## Documentation scope correction

The initial migration overreached by replacing VitePress and deleting
`apps/docs/.vitepress/config.mts`. User clarified that documentation is independent
of Genesis implementation. Restore VitePress configuration/theme from the previous
commit, retain updated Markdown, and keep all docs dependencies under `apps/docs`.
The Go migration and source-artifact checks exclude this independent project.
