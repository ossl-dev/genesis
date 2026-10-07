# Python Plugin

Status: detection and package-manager provisioning implemented on macOS and Debian/Ubuntu. Windows installation is manual.

```yaml
languages:
  - type: python
    version: "3.11"
```

Import `python({ version })` from `@ossl/genesis-plugins/python`. The required version accepts numeric major, minor, or full versions. Detection checks `python3 --version`: `3.11` accepts `3.11.9`; `3.11.8` requires that patch.

Matching installations skip system prerequisites. macOS requests a formula such as `python@3.11`; Debian/Ubuntu requests a package such as `python3.11`. Your repositories must provide the version. Genesis does not add Python repositories or use pyenv/conda.

Apply fails if `python3` on the execution PATH still does not match after system tasks. Installing a versioned interpreter does not necessarily change the default `python3`. Set the appropriate PATH in your shell or config before rerunning. Package repositories may supply a different patch.

Fedora/Arch provisioning, virtual environments, and pip packages are not implemented by this plugin.
