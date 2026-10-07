# Docker Plugin

Status: detection, Colima setup, and Linux installer paths implemented. Docker Desktop requires interactive completion; Windows installation is manual.

```yaml
tools:
  - type: docker
    version: latest
    include_compose: true
    install_desktop: false
```

Import `docker(options = {})` from `@ossl/genesis-plugins/docker`.

| Option | Default | Meaning |
| --- | --- | --- |
| `version` | `latest` | Accept any detected version, or require numeric components |
| `include_compose` | `true` | Check Compose alongside the Docker CLI |
| `install_desktop` | `false` | Download Desktop on macOS instead of using Colima |

Compose detection tries `docker compose version`, then legacy `docker-compose --version`. Matching installations skip prerequisites. Detection checks commands and versions; it does not prove the daemon is running.

Colima setup registers brew packages for Colima, Docker CLI, and optionally Compose. Desktop downloads an ARM64/x64 DMG and retains it for manual installation. Apply fails until installation and license acceptance are complete.

Linux has Debian/Ubuntu and Fedora/CentOS installer branches. Debian uses its own repository URLs. Shared prerequisites still use APT, so Fedora/CentOS support is incomplete. Installer commands inherit the configured environment, and shell pipelines propagate failure.

After automatic installation, `docker info` checks the daemon without pulling a test image. A failed check fails apply. Installation version pinning is incomplete: installers can select the repository's current release. There is no rollback.
