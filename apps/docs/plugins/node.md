# Node.js Plugin

Status: detection and NVM installation implemented on macOS/Linux. Windows and standalone installation are incomplete.

```yaml
tools:
  - type: node
    version: "22"
    use_nvm: true
    global_packages: [typescript]
```

Import `node(options)` from `@ossl/genesis-plugins/node` in TypeScript.

| Option | Default | Meaning |
| --- | --- | --- |
| `version` | required | Numeric major, minor, or full version |
| `use_nvm` | `true` | Install through NVM when the requested version is missing |
| `global_packages` | omitted | npm package specs installed after Node is ready |

Detection compares requested version components, so `2` does not match `22`. Matching installations skip system prerequisites. Global packages use the requested NVM runtime when NVM exists; otherwise they use npm on PATH. Package failures stop apply.

NVM installation resolves the Node executable and adds its directory to the environment for subsequent Genesis commands. It also sets the NVM default alias. Your calling shell still needs its NVM profile setup or a reload.

`use_nvm: false` validates existing installations but cannot install a missing runtime. Windows displays manual instructions and fails apply until the requested version is available.
