# Configuration

Genesis discovers `genesis.config.go`, `.yaml`, `.yml`, then `.json`, in that
order. Use `--config path` to select another file. YAML and JSON require only
the binary; Go configs execute trusted code through an installed Go compiler.

## YAML

```yaml
tools:
  - type: node
    version: "22"
    use_nvm: false
languages:
  - type: python
    version: "3.11"
env:
  WORKSPACE: "${HOME}/work"
```

Builtin `type` entries supply ID, category, and module. Full entries support
custom instance IDs and executable plugins:

```yaml
tools:
  - id: project-node
    category: tool
    module: node
    options:
      version: "22"
      use_nvm: false
  - id: project-pnpm
    category: tool
    module: pnpm
    options:
      version: "10.19.0"
      node_plugin: project-node
```

Builtin module names accept `node`, `builtin:node`, and existing qualified
builtin aliases. Categories are tool, sdk, language, library, and framework.
IDs must be unique across all sections. Dependencies reference configured IDs.
Unknown builtin options and malformed values fail before provisioning.

`${NAME}` references expand recursively from the process environment. Unset
variables fail with field/file context. Config `env` entries merge into command
environments; they do not change the caller's shell or supply interpolation
values to other config entries. JSON plans show names, not secret values.

## Go configs

A Go config is a standalone program that prints exactly one JSON configuration.
It can compute values, read files, or use imported Go packages. Keep diagnostics
on stderr. For a dependency-free example:

```go
//go:build ignore

package main

import "fmt"

func main() {
    fmt.Println(`{"tools":[{"id":"git","category":"tool","module":"git"}]}`)
}
```

Save as `genesis.config.go`. The build constraint excludes it from ordinary
package builds; an explicit `go run genesis.config.go` still executes it.
`genesis init --format go` generates this program shape. To use the typed
`core.Config` API, initialize a normal Go module requiring Genesis and encode
the configuration using `encoding/json`.

## Repositories and scripts

```yaml
repositories:
  - url: https://github.com/example/project.git
    folder: project
    branch: main
scripts:
  - name: prepare
    command: echo preparing
    when: before
  - name: check
    command: git --version
```

Repository `path` remains an alias for `folder`. Relative paths resolve against
execution cwd. Existing destinations must be repository roots with matching
origin and requested branch. Genesis does not reset, pull, or overwrite work.
Scripts use `sh -c` on Unix and `cmd.exe` on Windows. Missing `when` means after.
Order: before scripts, plugins, repositories, after scripts. Plugin failure skips
repositories and after scripts.

## Preview

```sh
genesis apply --config configs/dev.yml --dry-run --json
```

Dry-run invokes no provisioning lifecycle hooks. Loading Go configs and external
plugin metadata/option validation executes configured programs; trust those
sources. JSON is an action plan, not proof a host needs changes. Apply JSON
requires dry-run; diagnostics/diff/validate support their own JSON output.

## Deadlines

`apply`, `doctor`, `diff`, and `validate` accept `--timeout 15m` (or Go duration
units such as `30s`). The default is no overall deadline. The deadline covers
config programs, plugin loading, prerequisites, scripts, repositories, and
installers. Context cancellation stops active commands/downloads and prevents
later actions. Completed system/package changes are retained.

```sh
genesis apply --parallel 4 --timeout 15m
```
