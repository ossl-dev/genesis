# Git Plugin

Status: detection and package provisioning implemented. Source/binary paths are experimental; Windows installation is manual.

```yaml
tools:
  - type: git
    version: latest
    install_method: package
```

Import `git(options = {})` from `@ossl/genesis-plugins/git`.

| Option | Default | Meaning |
| --- | --- | --- |
| `version` | `latest` | Accept any installed version, or require numeric version components |
| `install_method` | `package` | `package`, `source`, or `binary` |

Detection checks `git --version`. Matching installations skip package-manager work. Package installation uses brew on macOS and APT on Debian/Ubuntu. A package install that leaves the requested version unavailable fails apply.

`latest` accepts any detected version; it does not force an upgrade. Package repositories determine available versions. The source path needs Git to clone sources and does not enforce the requested release. Binary asset resolution is incomplete. Prefer package installation or preinstall Git.

Source/binary paths may set global `init.defaultBranch` and `pull.rebase` after installation. User name/email options are unsupported. Configure them explicitly in a setup script.

Repository cloning uses the config's [repositories section](/guide/configuration).
