# Troubleshooting

## Configuration errors

Select the intended file with `genesis apply --config configs/dev.yml`. Errors
include config paths and fields. Quote version strings. Unknown plugin options,
duplicate IDs, missing dependencies, cycles, and unset `${NAME}` references fail
before provisioning. Go configs require an installed Go compiler and exactly one
JSON value on stdout; use stderr for diagnostics.

Set `GENESIS_DEBUG=1` for configuration and task execution diagnostics on stderr.
JSON commands suppress diagnostic chatter.

## Home and executable paths

Default managed installations and local credentials require an available,
absolute HOME (USERPROFILE on Windows). Invalid/relative home paths fail before
writing a default prefix. An explicit absolute `install_dir` remains usable
without a home; matching host tools can still satisfy detection. NVM_DIR and
XDG_CONFIG_HOME must also be absolute when used for NVM setup.

Relative PATH entries resolve against the command's working directory, including
paths containing spaces. Genesis does not fall back to a different host PATH
when the configured environment cannot find a command.

## Missing tools or wrong versions

Run `genesis doctor --config configs/dev.yml`. Managed runtime paths are exposed
to later commands in the same execution. Future shells need their own PATH setup;
Genesis does not modify every shell profile. Numeric versions match components,
so 3.1 does not match 3.11. JDK detection requires both java and javac.

## Stalled workflows

Use `genesis apply --timeout 15m` to bound the whole workflow, or interrupt with
Ctrl-C. Deadlines propagate through config programs, installers, and child
commands. Completed host changes remain; inspect reported state before retrying.

## Failed installers

Check published release availability, target architecture/libc, checksum metadata,
network access, and destination permissions. Existing unrelated nonempty folders
are protected. Prefix locks prevent concurrent replacement; after a crash, inspect
that no installer is running before manually removing a stale lock. Recovery errors
report a retained backup path; preserve it until the installation is recovered.

## Docker

CLI presence alone is insufficient. Validation checks daemon readiness with a
15-second deadline and requires requested client/server versions to match. Start
the daemon or Colima and check socket permissions. Linux systemd startup requires
a real systemd host. Desktop is manual, and automatic macOS numeric pins are not
supported. See [Docker reference](/plugins/docker).

## Platform prerequisites

System tasks select Linux package managers from distro metadata. Unsupported
systems fail explicitly. Arch hosts need a current coherent package database;
run the host's full maintenance update first. Homebrew bootstrap requires trusted
network access and host permissions. Windows tools without automatic installers
report manual setup requirements. See [platform support](/guide/platform-support).

## Cloud/cache

Cloud apply, OAuth verification, and full environment restore are not implemented.
`login --token` saves private metadata only. `list --cloud --format json` sends the
backend warning to stderr. Metadata cache exports are not executable backups.
