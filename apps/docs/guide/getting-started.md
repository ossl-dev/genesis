# Getting Started

[Build Genesis from source](/guide/installation), then run its built entrypoint from your project directory.

```bash
bun /path/to/genesis/apps/cli/dist/index.js init
```

`init` prompts for TS or YAML format and writes missing config/package/README files. It does not ask which tools to install or support `--defaults`. Edit the generated config to describe your project.

```yaml
# genesis.config.yaml
tools:
  - type: node
    version: "22"
    use_nvm: true
env:
  NODE_ENV: development
scripts:
  - name: install-dependencies
    command: npm install
    when: after
```

Preview the planned actions, then apply:

```bash
bun /path/to/genesis/apps/cli/dist/index.js apply --dry-run
bun /path/to/genesis/apps/cli/dist/index.js apply
bun /path/to/genesis/apps/cli/dist/index.js validate
```

Apply installs missing tools through their plugins, runs scripts with configured environment values, and exits nonzero on failure. Scripts run on every apply; choose commands safe to repeat. Environment variables are not written into your parent shell.

Default discovery prefers `genesis.config.ts` over `genesis.config.yaml`. Use `apply --config ./dev.yaml` to select another file. Configs and plugins can execute code while loading, even during dry-run; use trusted sources.

See [Configuration](/guide/configuration), [Plugins](/plugins/overview), and [Troubleshooting](/guide/troubleshooting).
