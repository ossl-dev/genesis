# Docker Plugin

Install and manage Docker containerization platform for Genesis environments.

## Features

- **Multiple Installation Options**: Docker Desktop or Docker Engine
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Docker Compose**: Automatic Docker Compose installation
- **Platform-Specific**: Optimized installation for each OS
- **Task Deduplication**: Optimized system package installation
- **Verification Testing**: Automatic post-installation testing

## Installation

Add to your Genesis configuration:

### TypeScript Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { docker } from "@ossl/genesis-plugins/docker";

export default defineConfig({
  tools: [
    docker({
      version: "latest",           // Optional: specific version or "latest"
      include_compose: true,       // Optional: include Docker Compose
      install_desktop: false,      // Optional: install Docker Desktop (macOS/Windows)
    }),
  ],
});
```

### YAML Configuration

```yaml
tools:
  - type: docker
    version: "latest"              # Optional: specific version or "latest"
    include_compose: true          # Optional: include Docker Compose
    install_desktop: false         # Optional: install Docker Desktop (macOS/Windows)
```

## Options

- `version` (string, optional): Docker version to install (e.g., "24.0.0", "latest", default: "latest")
- `include_compose` (boolean, optional): Install Docker Compose (default: true)
- `install_desktop` (boolean, optional): Install Docker Desktop on macOS/Windows (default: false)

## Platform Support

### macOS

**Docker Engine (Default)**:
- Uses Colima for lightweight Docker Engine
- Faster and less resource-intensive
- Command-line focused

**Docker Desktop (Optional)**:
- Full GUI application
- Integrated development experience
- Requires manual installation steps

### Linux

**Docker Engine**:
- Official Docker repository installation
- Systemd service integration
- User group management
- Supports Ubuntu, Debian, Fedora, CentOS/RHEL

### Windows

**Docker Desktop**:
- Provides detailed installation guide
- WSL 2 integration
- Manual installation required
- Windows 10/11 Pro, Enterprise, or Education

## Installation Methods

### Docker Engine (Recommended for Servers)

**Advantages**:
- Lightweight and fast
- Command-line interface
- Better for CI/CD and servers
- Less resource usage

**Use Cases**:
- Development servers
- CI/CD pipelines
- Production environments
- Headless systems

### Docker Desktop (Recommended for Development)

**Advantages**:
- User-friendly GUI
- Integrated development tools
- Easy container management
- Built-in Kubernetes

**Use Cases**:
- Local development
- GUI-based workflows
- Kubernetes development
- Team collaboration

## Supported Versions

- Docker 20.10+
- Docker 24.x (latest)
- Docker Compose 2.x
- Latest stable releases

## Examples

### Basic Docker Installation

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { docker } from "@ossl/genesis-plugins/docker";

export default defineConfig({
  tools: [
    docker(),  // Uses defaults: latest version, include Compose, no Desktop
  ],
});
```

### Docker Desktop on macOS

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { docker } from "@ossl/genesis-plugins/docker";

export default defineConfig({
  tools: [
    docker({
      install_desktop: true,
      include_compose: true,
    }),
  ],
});
```

### Server-Ready Docker on Linux

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { docker } from "@ossl/genesis-plugins/docker";

export default defineConfig({
  tools: [
    docker({
      version: "24.0.0",
      include_compose: true,
      install_desktop: false,
    }),
  ],
});
```

## Post-Installation Setup

After Docker installation:

```bash
# Verify Docker installation
docker --version
docker-compose --version

# Test Docker with hello-world
docker run hello-world

# Manage Docker as non-root user (Linux)
# You may need to log out and log back in
docker ps
```

## User Permissions

### Linux

The plugin automatically adds your user to the `docker` group:

```bash
# Add user to docker group (done automatically)
sudo usermod -aG docker $USER

# Apply group changes
newgrp docker

# Or log out and log back in
```

### macOS/Windows

Docker Desktop handles user permissions automatically.

## Troubleshooting

### Docker Not Found After Installation

1. **Linux**: Ensure user is in docker group or use sudo
2. **macOS**: Restart Docker Desktop or restart Colima
3. **Windows**: Restart Docker Desktop service

### Permission Denied

```bash
# Linux: Add user to docker group
sudo usermod -aG docker $USER

# Then log out and log back in
```

### Docker Daemon Not Running

```bash
# Linux: Start Docker service
sudo systemctl start docker
sudo systemctl enable docker

# macOS: Restart Docker Desktop or Colima
colima restart
```

### Windows Installation

1. Enable WSL 2: `wsl --install`
2. Install Docker Desktop from official website
3. Restart your computer
4. Verify installation: `docker --version`

### macOS Colima Issues

```bash
# Restart Colima
colima restart

# Check Colima status
colima status

# Reset Colima if needed
colima delete --force
colima start
```

## Performance Notes

- Docker Engine is lighter and faster than Docker Desktop
- Colima provides good performance on macOS without Desktop
- Linux installation is optimized for production use
- Task deduplication prevents redundant package operations

## Security Considerations

- Docker group membership is equivalent to root access
- Use Docker Desktop for development, Engine for production
- Regular security updates available through package managers
- Container security best practices still apply

## Integration with Development Workflow

After installation:

```bash
# Build a Docker image
docker build -t myapp .

# Run a container
docker run -p 8080:80 myapp

# Use Docker Compose
docker-compose up -d

# View running containers
docker ps

# View logs
docker logs <container-id>

# Stop containers
docker-compose down
```

## Docker Compose

The plugin automatically installs Docker Compose when `include_compose: true`:

```yaml
# docker-compose.yml
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:latest
    environment:
      POSTGRES_PASSWORD: secret
```

## Container Registry Integration

Configure Docker registry access:

```bash
# Login to Docker Hub
docker login

# Login to private registry
docker login registry.example.com

# Pull images
docker pull ubuntu:latest

# Push images
docker push myusername/myapp:latest
```

## Development Tips

### Useful Docker Commands

```bash
# Clean up unused resources
docker system prune -a

# View disk usage
docker system df

# Inspect containers
docker inspect <container-id>

# Execute commands in running container
docker exec -it <container-id> bash
```

### Development Workflow

```bash
# Development with hot reload
docker-compose up --build

# Production build
docker-compose -f docker-compose.prod.yml up -d

# Run tests
docker-compose run --rm app npm test
```

## Integration with Other Tools

The Docker plugin integrates seamlessly with:
- Kubernetes (via Docker Desktop or k3s)
- CI/CD platforms (GitHub Actions, GitLab CI)
- Development environments (VS Code, IntelliJ)
- Container registries (Docker Hub, ECR, GCR)
- Monitoring tools (Prometheus, Grafana)

---

**Part of the [Genesis](../../../README.md) project**
