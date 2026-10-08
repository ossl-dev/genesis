# Docker Plugin

```yaml
tools:
  - type: docker
    version: "29.1" # Linux release prefix; use latest for Colima
    include_compose: true
    install_desktop: false
```

Import `docker(options = {})` from `@ossl/genesis-plugins/docker`.

| Option | Default | Meaning |
| --- | --- | --- |
| `version` | `latest` | Accept an installed version, or require numeric components in both client and daemon |
| `include_compose` | `true` | Install and check Compose alongside Docker |
| `install_desktop` | `false` | Require manually installed Docker Desktop instead of automatic Engine/Colima setup |

Detection checks the client and, when requested, Compose. It tries `docker compose version`, then legacy `docker-compose --version`. Apply and doctor also check the connected daemon's version through `docker info`, with a 15-second timeout. A stopped daemon, inaccessible socket, or mismatched remote daemon fails validation even when the client matches.

On macOS, automatic setup uses brew packages for Colima, Docker CLI, and optionally Compose, then starts Colima. A stopped Colima instance is started even when the CLI is already installed. Automatic macOS release pinning is unsupported: preinstall a matching client and daemon, or use `version: latest`. Unsupported pins fail before this plugin registers package tasks.

Docker Desktop installation and license acceptance are manual on macOS and Windows. Genesis directs you to the [official setup instructions](https://docs.docker.com/desktop/) and reports failure until the requested client, Compose, and daemon are available. It does not download an unchecked Desktop installer.

Linux automatic setup supports Ubuntu, Debian, Fedora, CentOS, and RHEL on x64/ARM64. It uses the distribution's [official Docker repository](https://docs.docker.com/engine/install/), writes repository files atomically, and resolves matching stable Engine/CLI packages before installing them. Numeric prefixes select the newest matching release; full versions select that release. An unavailable pin fails before Engine package installation, though repository setup may already have changed files. Existing distro packages can conflict and require manual resolution.

Compose is omitted when disabled, including optional package-manager recommendations. If a matching client only lacks Compose, setup installs the Compose package without reinstalling Engine packages or changing the service. Dependencies such as containerd and Buildx use repository versions. Setup enables and starts the systemd Docker service; hosts without systemd need manual service setup. User group membership is not changed automatically: configure socket access using Docker's [Linux post-install instructions](https://docs.docker.com/engine/install/linux-postinstall/), then rerun Genesis.

A failed installation reports whether setup already changed the host. System package and service changes have no automatic rollback. Genesis never pulls a test image or upgrades every installed host package during setup.
