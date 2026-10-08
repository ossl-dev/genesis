# Node.js Plugin

Status: NVM installation on macOS/Linux and checksummed standalone archives on macOS, glibc Linux, and Windows for published x64/ARM64 releases.

```yaml
tools:
  - type: node
    version: "22"
    use_nvm: false
    global_packages: [typescript]
    # install_dir: "${HOME}/.local/share/genesis/node"
```

Import `node(options)` from `@ossl/genesis-plugins/node`.

| Option | Default | Meaning |
| --- | --- | --- |
| `version` | required | Numeric major, minor, or full release |
| `use_nvm` | `true` | Use NVM on Unix when no `install_dir` is specified |
| `install_dir` | omitted | Absolute managed directory; selects standalone installation |
| `global_packages` | omitted | npm package specs installed after Node is ready |

Standalone installation resolves the newest stable release matching the requested components from Node's official index and verifies its published SHA-256. It extracts into unique staging, verifies the executable before/after promotion, and restores the previous directory on promotion failure. A nonempty destination without the expected executable is refused. Unsupported architectures, missing assets, and musl Linux fail explicitly. The default directory is `~/.genesis/node`; Windows uses the native ZIP path regardless of `use_nvm`.

Managed standalone installs are rediscovered on later runs and exposed to subsequent Genesis commands through PATH. Unix archives require `tar`; Windows requires ZIP-capable `tar`. Add the printed executable directory to your shell PATH for future sessions. Archive replacement owns the whole configured directory, including packages installed there.

NVM installation resolves and verifies the selected Node executable, adds it to runtime PATH, and sets the default alias. Failed installation, alias updates, or version verification report failure. Your calling shell still needs its NVM profile setup or a reload. Global packages use npm from the selected runtime PATH, without requiring an NVM shell when an existing Node already matches. Package failures stop apply.

Matching installations skip system prerequisite work. See [Platform Support](/guide/platform-support) for real installer coverage and remaining limits. Official archives are available from the [Node distribution index](https://nodejs.org/dist/).
