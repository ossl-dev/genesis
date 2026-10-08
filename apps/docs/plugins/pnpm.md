# pnpm Plugin

Status: pinned npm-based installation on macOS, Linux, and Windows with a compatible Node/npm runtime.

```yaml
tools:
  - type: node
    version: "22"
  - type: pnpm
    version: "10.19.0"
```

Import `pnpm(options)` from `@ossl/genesis-plugins/pnpm`.

- `version`: required full numeric release.
- `node_plugin`: configured Node plugin ID, default `node`. Set `null` to use Node/npm already on PATH.
- `install_dir`: optional absolute npm prefix owned by this plugin; defaults to `~/.genesis/pnpm`.

The plugin installs an exact npm package into a staged prefix with engine checks enabled and lifecycle scripts disabled. It verifies `pnpm --version` before and after promotion and restores the previous prefix on failure. A missing/incompatible Node runtime or failed npm operation is reported as failure. pnpm runs after its configured Node dependency; custom Node IDs must match `node_plugin`.

Genesis rediscovers managed executables and adds the prefix's executable directory to runtime PATH (`bin` on Unix, the prefix itself on Windows). An explicit installation directory selects that managed prefix. Shell PATH changes, project dependencies, and global pnpm packages are not managed. See [upstream installation](https://pnpm.io/installation) and [platform validation limits](/guide/platform-support).
