# Genesis CLI

Build from the repository root with `bun run build`, then run `bun apps/cli/dist/index.js --help`. From another project directory, use the absolute path to the built entrypoint. YAML also works under Node; TS configs require runtime TypeScript support.

| Command | Options / behavior |
| --- | --- |
| `create`, `init` | Interactive scaffold; existing config/package/README files are preserved |
| `apply` | Apply the discovered local config |
| `apply --config <file>` | Load a selected `.ts`, `.yaml`, or `.yml` file |
| `apply --dry-run` | Print ordered actions without provisioning |
| `apply --dry-run --json` | Print JSON; environment variable names are included, values omitted |
| `diff` | Plugin presence/version detection |
| `validate` | Plugin validation; failed checks exit nonzero |
| `doctor` | Configured plugin detection and validation; failed checks exit nonzero |
| `list-plugins` | List built-in plugin exports |
| `list --format json` | Local config/cache metadata |
| `login --token <token> --url <url>` | Save a token locally; it is not verified by a backend |

`apply <environment-id>` and `apply --cloud` are unavailable and fail explicitly. `list --cloud` reports the unavailable backend. There are no restore, watch, completion, or cloud-management commands. `--verbose`, `--quiet`, `--cwd`, `--defaults`, and `--version` are not implemented. Enable debug logs with `GENESIS_DEBUG=1`.

The apply order is: before scripts, deduplicated system tasks, plugins in dependency order, repositories, after scripts. Scripts without `when` run after. Failures prevent a success message and produce a nonzero exit. No automatic rollback occurs.

`--json` currently requires `--dry-run`. Dry-run loads config/plugin modules but does not invoke lifecycle methods, repository commands, or scripts. Custom imports can still execute module-level code.

See [Configuration](../docs/guide/configuration.md), [Plugins](../docs/plugins/overview.md), and [Troubleshooting](../docs/guide/troubleshooting.md).
