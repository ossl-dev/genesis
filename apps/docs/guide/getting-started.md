# Getting Started

[Build Genesis](/guide/installation), then run `genesis init --format yaml` in an
environment repository. Existing files are preserved.

```yaml
tools:
  - type: node
    version: "22"
    use_nvm: false
  - type: git
scripts:
  - name: verify tools
    command: git --version
```

```sh
genesis apply --dry-run --json
genesis apply
genesis validate
genesis doctor
```

Preview loads and validates configuration without provisioning. Apply installs
plugins and runs scripts/repositories in order. Validate checks configured tools;
doctor combines detection and validation. Failed operations exit nonzero.
For explicit configs use `--config configs/dev.yml`. Independent plugins can run
with `apply --parallel 4`; prerequisites and dependencies remain ordered.
