# Standalone distribution

Genesis now builds as a Go binary. It needs no bootstrap runtime or workspace
orchestrator. `go run ./scripts/release` produces six binaries for macOS, Linux,
and Windows on amd64/arm64, with CGO disabled.

YAML configurations need only that binary. Executable Go configurations require
an installed Go compiler and emit one JSON configuration on stdout. External
plugins are ordinary platform executables using the versioned JSON protocol;
compiled Go plugins can also be registered through the library API.

Release automation uploads generated binaries for version tags. Package-manager
manifests, signed/notarized releases, automatic updates, and a public install
endpoint remain roadmap work. Runtime installation plugins provision user tools
independently of Genesis's own executable.
