# Architecture

One Go module keeps applications in `apps` and reusable code in `packages`.

- `apps/cli`: command parsing, interactive scaffolding, output, local auth/listing.
- `apps/docs`: independent VitePress documentation, with its own manifest and lockfile.
- `packages/core`: configuration, plugin/task graphs, subprocesses, execution,
  repository/scripts, metadata cache APIs.
- `packages/plugins`: eleven builtins, strict options, release resolution, verified
  archive installs, package-manager installers.
- `packages/extension`: Go configuration execution and portable executable plugins.
- `scripts`: release cross builds and disposable installer smoke checks.

Configuration normalizes YAML shorthand and validates IDs/options. Plugin loading
resolves builtins or explicit external executables. Dependency graphs reject
missing/cyclic edges before lifecycle execution. Prepare runs across all plugins,
then shared tasks register and execute, then pre/apply/post run by dependency.

Default apply is sequential. `--parallel N` uses bounded dependency layers;
shared paths/ports serialize conflicting work. Environment access is synchronized,
command output and downloads are bounded, and context cancellation terminates
child processes and waits for workers. Plugins must cooperate with cancellation.

Archive promotion verifies both staged and installed executables. It restores
previous prefixes on failure, retaining backup evidence if recovery fails.
System package operations and arbitrary scripts do not have general rollback.

Diff reports detection, doctor detects and validates configured plugins. Cache
metadata import/export/compression/snapshots remain available; filesystem/network
restoration and cloud APIs are explicitly unfinished.
