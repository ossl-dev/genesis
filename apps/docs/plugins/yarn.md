# Yarn Plugin

Status: pinned npm-based installation of Yarn Classic and modern Yarn on macOS, Linux, and Windows with compatible Node/npm.

```yaml
tools:
  - type: node
    version: "22"
  - type: yarn
    version: "4.9.4"
```

Import `yarn(options)` from `@ossl/genesis-plugins/yarn`.

`version` requires a full numeric release. Yarn 1 installs `yarn@<version>`; modern releases install the upstream `@yarnpkg/cli-dist@<version>` package. Unpublished versions fail explicitly. `node_plugin` defaults to the configured ID `node`; set another ID for a custom instance or `null` to use an existing Node/npm runtime. `install_dir` is an optional absolute prefix owned by this plugin, defaulting to `~/.genesis/yarn`.

npm installs into a staged prefix with engine checks enabled and lifecycle scripts disabled. Genesis verifies the selected CLI before/after promotion, restores the old prefix on failure, and exposes the executable directory to later commands. Verification ignores project `yarnPath` delegation; project commands retain their normal Yarn configuration.

This provisions an explicit CLI without changing Corepack, project manifests, lockfiles, shell profiles, or global Yarn packages. Yarn recommends [Corepack](https://yarnpkg.com/corepack) for projects that want automatic package-manager selection; Genesis does not manage that integration. See [platform validation limits](/guide/platform-support).
