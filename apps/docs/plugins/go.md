# Go Plugin

Status: detection and archive installation implemented on macOS/Linux. Windows installation is manual.

```yaml
languages:
  - type: go
    version: "1.22.5"
```

Import `go({ version })` from `@ossl/genesis-plugins/go`. Archive installation requires a complete numeric version, including the patch. Matching installations skip system prerequisites.

Detection checks `go version`. Installation downloads from `go.dev` for the host OS and architecture: x64 maps to amd64 and ARM64 to arm64. It extracts into `/usr/local/go`, which requires filesystem permissions. Genesis does not acquire elevated privileges for extraction.

The plugin prints PATH instructions for `/usr/local/go/bin`; it does not persist shell profile changes. An existing installation can be replaced, and failed extraction has no automatic rollback. Use detection/validation with preinstalled Go when elevated provisioning is unsuitable.
