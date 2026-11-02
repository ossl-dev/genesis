# Genesis CLI

Build from repository root: `go build -o bin/genesis ./apps/cli`.
Run `bin/genesis --help`, or invoke its absolute path from an environment project.
YAML requires no Go compiler at runtime; executable `.go` configs use `go run`.

Commands: `create`, `init`, `apply`, `doctor`, `diff`, `validate`, `list-plugins`,
`login`, and `list`. Scaffolding supports `--format yaml|go` and `--name` for
noninteractive use and preserves existing files.

`apply --config path.yml --dry-run --json` exports a plan without provisioning.
`apply --parallel 4` bounds independent plugin execution by available CPUs.
`--timeout 15m` sets a deadline for config loading and environment operations;
apply, doctor, diff, and validate accept it. Cancellation stops active child
commands and downloads; completed host changes are not rolled back.
Inspection commands accept `--config` and `--json`. Failed apply, validation,
and diagnostics exit nonzero. Cloud apply fails explicitly; login only stores
private local token metadata. `list --format json --cloud` keeps warnings on
stderr and emits valid JSON on stdout.
