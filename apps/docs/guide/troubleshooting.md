# Troubleshooting

## Config not found or wrong config selected

Default names are `genesis.config.ts` and `genesis.config.yaml`, with TS taking precedence. Run from the project directory or select a file explicitly:

```bash
bun /path/to/genesis/apps/cli/dist/index.js apply --config ./configs/dev.yml
```

Relative repository paths and script working directories still use the invoking directory, not the config file's directory.

## Invalid config or plugin options

Errors include the config filename/field path or plugin ID. Check types, required version values, unknown options, and duplicate IDs. One plugin ID may appear only once across all sections. Go archive installation requires a complete version such as `1.22.5`.

Unset `${NAME}` references fail at load time. Export the variable before invoking Genesis. Expansion uses process environment values, not other entries in the config's `env` section.

## TypeScript loading fails

Use Bun for TypeScript configs. Node needs runtime TypeScript support or a configured loader. Import Genesis helpers from `@ossl/genesis-core` and plugin factories from `@ossl/genesis-plugins`; ensure those packages are available to the project runtime.

## System task failed

A failed system prerequisite stops plugin apply. Debug logs can show registered tasks:

```bash
GENESIS_DEBUG=1 bun /path/to/genesis/apps/cli/dist/index.js apply
```

Check the reported package command, permissions, network access, and package availability. Shared Linux tasks select APT/DNF/pacman/APK from `os-release`. Unknown distributions fail explicitly. On macOS, include Homebrew in the config or preinstall it before tools that need brew tasks. Archive extraction into system directories also requires filesystem permissions.

## Tool installed but validation still fails

Check which executable is on PATH and its version. A versioned Python package may not change default `python3`. Go/Java profile changes are printed as instructions. Config `env` affects child commands, not your invoking shell. NVM installation adds its selected executable to subsequent Genesis commands; reload/source NVM to use it in your shell.

`doctor` and `validate` require a project config. `diff` reports detection status. There is no separate `detect` command.

## Repository destination already exists

Genesis requires the destination to be a repository root with the configured origin and requested branch. It refuses mismatches without changing branches or overwriting files. Correct the config or choose another destination. Existing worktrees are not pulled or reset.

## Script failed

The error identifies the script name and exit code. Before scripts run before tool installation; after scripts run after successful plugins and repository setup. Scripts use `sh` on Unix and `cmd.exe` on Windows. Write commands compatible with the target shell and safe to repeat.

## Docker Desktop or daemon unavailable

Desktop requires manual installation and license acceptance; Genesis reports the required setup without downloading an installer. Apply and doctor check the daemon with a 15-second timeout. A failing `docker info` check indicates startup, socket access, or connection problems; a daemon version mismatch can also fail a pin. Linux package changes have no automatic rollback.

## Cloud or restore commands

Cloud apply is unavailable. Login only stores an unverified token. No restore command exists, and the cache prototype does not restore the filesystem. Use local version-controlled configs.

Report reproducible problems at [GitHub Issues](https://github.com/ossl-dev/genesis/issues), including the command, OS/architecture, redacted config, and error output.

## Interrupted staged installation

Installers use an adjacent `.genesis-<random>` staging directory and `<install_dir>.genesis-lock`. A crash can leave both. Before removing a stale lock, confirm no install is running and inspect retained `previous` contents. Recovery failures report the backup path; preserve it until the intended installation has been restored. Nonempty directories without the expected executable are refused to avoid replacing unrelated files.
