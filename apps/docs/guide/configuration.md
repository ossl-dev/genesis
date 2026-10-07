# Configuration

Genesis discovers `genesis.config.ts` first, then `genesis.config.yaml` in the current directory. Select another file with `genesis apply --config ./configs/dev.yaml`; explicit paths accept `.ts`, `.yaml`, or `.yml` and resolve from the current directory.

TypeScript configs execute through the host runtime's module loader. Use Bun for TypeScript configs; Node requires a version with TypeScript support or a suitable loader. Only load configs and plugins you trust.

## YAML

```yaml
tools:
  - type: node
    version: "22"
    use_nvm: true
languages:
  - type: python
    version: "3.11"
env:
  NODE_ENV: development
  WORKSPACE: "${HOME}/projects"
```

Built-in types are `node`, `python`, `go`, `java`, `git`, `docker`, and `homebrew`. YAML entries may also use the full plugin instance format for custom modules:

```yaml
tools:
  - id: team-tool
    category: tool
    module: team-genesis-plugin
    options:
      version: "1"
```

Both entry formats can appear in the same list. Each plugin ID must be unique across `tools`, `sdks`, and `languages`; duplicate IDs are rejected rather than silently dropping one instance.

## TypeScript

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [node({ version: "22", use_nvm: true })],
  languages: [python({ version: "3.11" })],
  env: { NODE_ENV: "development" },
});
```

A default export or named `config` export is accepted. `defineConfig` supplies TypeScript types; loading the file performs runtime validation.

## Environment references

`${NAME}` in any string value is replaced with the invoking process's environment value. Missing variables stop loading and identify the affected field. References are expanded once; shell expressions and fallback syntax such as `${NAME:-value}` are not evaluated.

The config's `env` values override inherited environment variables for Genesis plugin commands. They do not persist into your calling shell, and references do not resolve against other entries in the config's `env` section.

## Validation

All top-level sections are optional. Plugin instances require a nonempty `id`, `module`, and a category: `tool`, `sdk`, `language`, `library`, or `framework`.

Repository specs use `url`, `folder` (or YAML `path`), and optional `branch`. Scripts use `name`, `command`, optional `description`, and optional `when: before | after`.

Invalid input is reported with the config filename and field path, for example `tools.0.module: Invalid input: expected string, received number`. Unsupported plugin types and duplicate IDs fail before provisioning begins.

See [Plugin Overview](/plugins/overview) for plugin options and platform limitations.
