# Homebrew Plugin

Status: implemented for macOS. This plugin does not provision Homebrew on Linux or Windows.

```yaml
tools:
  - type: homebrew
    update_packages: false
    install_cask: false
    add_to_path: true
    global_packages: [jq]
```

Import `homebrew(options = {})` from `@ossl/genesis-plugins/homebrew`.

| Option | Default | Meaning |
| --- | --- | --- |
| `update_packages` | `true` | Upgrade installed formulae during apply |
| `install_cask` | `true` | Upgrade casks when package upgrades are enabled |
| `add_to_path` | `true` | Print shell setup instructions |
| `global_packages` | omitted | Packages installed after Homebrew is ready |

Detection checks `brew --version`. The official install script runs with pipeline failure propagation. The installer selects `/opt/homebrew` on ARM64 or `/usr/local` on Intel and adds its binary directory to the environment for later Genesis commands. Profile changes are printed for you to apply.

Global packages are preserved by the helper factory and installed after apply. The plugin does not invoke brew to install its own prerequisites.

Install Homebrew before other plugins register brew system tasks: those tasks run before plugin apply. A Homebrew entry in the same config does not resolve initial bootstrap ordering. Some update failures are currently logged rather than propagated.
