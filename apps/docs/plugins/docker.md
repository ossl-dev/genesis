# Docker Plugin

Install and manage Docker containerization platform for Genesis environments.

## Overview

The Docker plugin provides comprehensive Docker installation and management for Genesis environments. It supports both Docker Engine and Docker Desktop installations across macOS, Linux, and Windows platforms, with automatic Docker Compose integration.

## Features

- **Multiple Installation Options**: Docker Engine or Docker Desktop
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Docker Compose**: Automatic Docker Compose installation
- **Platform-Specific**: Optimized installation for each OS
- **Verification Testing**: Automatic post-installation testing
- **Task Deduplication**: Optimized system package installation

## Installation

### Basic Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { docker } from "@ossl/genesis-plugins/docker";

export default defineConfig({
  tools: [
    docker({
      version: "latest",
      include_compose: true,
      install_desktop: false,
    }),
  ],
});
```

### YAML Configuration

```yaml
tools:
  - type: docker
    version: "latest"
    include_compose: true
    install_desktop: false
```

## Options

| Option | Type | Required | Default | Description |
|---------|--------|----------|-------------|
| `version` | string | No | "latest" | Docker version to install (e.g., "24.0.0", "latest") |
| `include_compose` | boolean | No | true | Install Docker Compose |
| `install_desktop` | boolean | No | false | Install Docker Desktop on macOS/Windows |

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

## Usage Examples

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

## Development Workflow

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

## Docker Best Practices

### Image Optimization

```dockerfile
# Use multi-stage builds
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

FROM node:18-alpine AS production
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY . .
EXPOSE 3000
CMD ["node", "server.js"]
```

### Security

```dockerfile
# Use non-root user
RUN addgroup -g 1000 -S nodejs
RUN adduser -S nextjs -u 1000
USER nextjs

# Minimal base images
FROM alpine:latest
```

### Resource Management

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    image: myapp:latest
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
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

# View container logs in real-time
docker logs -f <container-id>
```

### Development Workflow

```bash
# Development with hot reload
docker-compose up --build

# Production build
docker-compose -f docker-compose.prod.yml up -d

# Run tests
docker-compose run --rm app npm test

# View logs
docker-compose logs -f app
```

## Integration with Development Tools

The Docker plugin integrates seamlessly with:
- **Kubernetes**: For orchestration
- **CI/CD platforms**: GitHub Actions, GitLab CI
- **Development environments**: VS Code, IntelliJ
- **Container registries**: Docker Hub, ECR, GCR
- **Monitoring tools**: Prometheus, Grafana

## Performance Notes

- Docker Engine is lighter and faster than Docker Desktop
- Colima provides good performance on macOS without Desktop
- Linux installation is optimized for production use
- Task deduplication prevents redundant operations

## Security Considerations

- Docker group membership is equivalent to root access
- Use Docker Desktop for development, Engine for production
- Regular security updates available through package managers
- Container security best practices still apply

## Advanced Configuration

### Docker Daemon Configuration

```json
// /etc/docker/daemon.json
{
  "registry-mirrors": ["https://mirror.gcr.io"],
  "insecure-registries": ["registry.mycorp.com"],
  "debug": true,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

### Docker Compose Overrides

```yaml
# docker-compose.override.yml
version: '3.8'
services:
  app:
    environment:
      - DEBUG=true
    volumes:
      - ./src:/app/src
```

## Related Plugins

- **Kubernetes Plugin**: For orchestration
- **Nginx Plugin**: For reverse proxy
- **PostgreSQL Plugin**: For database containers
- **Redis Plugin**: For caching
