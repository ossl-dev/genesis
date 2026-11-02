# Go Plugin

Status: archive installation for macOS, Linux, and Windows on x64/ARM64. Unsupported architectures and unavailable releases fail explicitly.

```yaml
languages:
  - type: go
    version: "1.22.5"
    # install_dir: "${HOME}/.local/share/genesis/go"
```

Go API: `plugins.Go()` from `github.com/ossl-dev/genesis/packages/plugins`.

`version` requires a full numeric release. `install_dir` is an optional absolute directory, owned by this installation. The default is `/usr/local/go` on macOS/Linux and `%USERPROFILE%/.genesis/go` on Windows. System paths require write permissions; Genesis does not elevate archive filesystem operations.

Release metadata is decoded one release at a time, with a 32 MiB response limit and a 30-second timeout. The complete response is validated before installation, including entries after the selected archive.

The installer resolves the official Go archive and SHA-256, downloads into unique staging, extracts through Go archive readers, and verifies `bin/go version` before replacing an existing install. Promotion or relocated verification failure restores the previous directory. Recovery failure retains the backup path in the error; a process crash can leave staging and an installation lock for manual recovery.

Managed installations are rediscovered on later runs. Genesis sets runtime `GOROOT` and adds the bin directory to `PATH`; add it to your shell PATH for future sessions. No shell profiles are edited. Matching versions skip installation entirely. Tests include real temporary archive extraction and recovery; see [Platform Support](/guide/platform-support) for installer validation limits.
