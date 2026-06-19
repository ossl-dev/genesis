# Go Plugin

Install and manage Go (Golang) programming language for Genesis environments.

## Overview

The Go plugin provides comprehensive Go language installation and management for Genesis environments. It downloads official Go distributions and sets up the development environment across macOS, Linux, and Windows platforms.

## Features

- **Official Downloads**: Downloads official Go distributions from go.dev
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Version Management**: Install specific Go versions (1.19, 1.20, 1.21, 1.22, etc.)
- **Environment Setup**: Automatic PATH configuration guidance
- **Task Deduplication**: Optimized system package installation
- **Clean Installation**: Removes previous versions before installing

## Installation

### Basic Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { go } from "@ossl/genesis-plugins/go";

export default defineConfig({
  languages: [
    go({
      version: "1.21",
    }),
  ],
});
```

### YAML Configuration

```yaml
languages:
  - type: go
    version: "1.21"
```

## Options

| Option | Type | Required | Default | Description |
|---------|--------|----------|-------------|
| `version` | string | Yes | - | Go version to install (e.g., "1.19", "1.20", "1.21", "1.22") |

## Supported Versions

- **Go 1.19**: Previous stable version
- **Go 1.20**: Previous stable version
- **Go 1.21**: Current stable version (recommended)
- **Go 1.22**: Latest stable version
- **Latest versions**: Automatically installs newest available

## Platform Support

### macOS & Linux

The plugin downloads official Go distributions:

1. Detects system architecture (x64 or ARM64)
2. Downloads appropriate tar.gz archive from go.dev
3. Extracts to `/usr/local/go`
4. Adds `/usr/local/go/bin` to PATH
5. Cleans up previous installations

### Windows

Provides detailed manual installation guidance:

1. Downloads Go MSI installer from go.dev
2. Step-by-step installation instructions
3. PATH configuration guidance
4. Verification steps

## Environment Variables

After installation, set up your Go environment:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export PATH="/usr/local/go/bin:$PATH"

# Optional: Set GOPATH (Go uses default ~/go)
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"
```

## Usage Examples

### Install Go 1.21 (Recommended)

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { go } from "@ossl/genesis-plugins/go";

export default defineConfig({
  languages: [
    go({
      version: "1.21",
    }),
  ],
});
```

### Install Latest Go

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { go } from "@ossl/genesis-plugins/go";

export default defineConfig({
  languages: [
    go({
      version: "1.22",
    }),
  ],
});
```

### Multiple Go Versions

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { go } from "@ossl/genesis-plugins/go";

export default defineConfig({
  languages: [
    go({
      version: "1.20",
    }),
    go({
      version: "1.21",
    }),
  ],
});
```

## Development Workflow

After installation:

```bash
# Verify Go installation
go version

# Check Go environment
go env

# Initialize a new module
go mod init myproject

# Build your project
go build

# Run tests
go test

# Install dependencies
go mod tidy

# Run your program
go run main.go
```

## Troubleshooting

### Go Not Found After Installation

1. Ensure `/usr/local/go/bin` is in your PATH
2. Restart your terminal or source your shell profile
3. Check installation directory: `ls -la /usr/local/go`

### Version Mismatch

1. Check which Go is active: `go version`
2. Verify only one Go installation exists
3. Remove previous versions before installing new ones

### Permission Issues

1. Use sudo for system-wide installation
2. Ensure write permissions to `/usr/local`
3. Consider user-specific installation if needed

### Windows Installation

1. Download from https://go.dev/dl/
2. Run the MSI installer
3. The installer automatically adds Go to PATH
4. Restart Command Prompt or PowerShell

### Architecture Support

The plugin automatically detects and supports:
- **x64 (amd64)**: Most common architecture
- **ARM64 (arm64)**: Apple Silicon, ARM servers

## Performance Notes

- Official Go distributions are optimized for performance
- Archive extraction is fast and efficient
- Task deduplication prevents redundant package manager updates
- Parallel installation with other plugins for optimal performance

## Go Modules

The installed Go version includes full Go module support:

```bash
# Initialize new module
go mod init github.com/username/project

# Add dependencies
go get github.com/gin-gonic/gin

# Update dependencies
go mod tidy

# Download dependencies
go mod download

# Verify dependencies
go mod verify
```

## Development Tools

### Popular Go Tools

```bash
# Install common development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/air-verse/air@latest

# Format code
go fmt ./...

# Lint code
golangci-lint run

# Hot reload for development
air
```

## Integration with Development Tools

The Go plugin integrates seamlessly with:
- **VS Code**: Go extension with IntelliSense
- **GoLand**: JetBrains IDE for Go
- **Vim/Neovim**: vim-go plugin
- **Emacs**: go-mode
- **Docker**: For containerized Go applications
- **GitHub Actions**: Go workflow support

## Best Practices

1. **Use Module Version**: Always use Go modules for dependency management
2. **Semantic Versioning**: Follow semantic versioning for releases
3. **Error Handling**: Use explicit error handling in Go
4. **Testing**: Write tests with the testing package
5. **Documentation**: Use godoc for code documentation

## Project Structure

Typical Go project structure:

```
myproject/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   └── myproject/
│       └── main.go
├── internal/
│   ├── app/
│   └── config/
├── pkg/
│   └── api/
├── docs/
└── tests/
    └── integration/
```

## Security Considerations

- Downloads are from official Go team servers
- HTTPS downloads ensure secure transfer
- Archive integrity is maintained
- Regular security updates available

## Related Plugins

- **Git Plugin**: For version control
- **Docker Plugin**: For containerization
- **Node.js Plugin**: For full-stack development
- **Homebrew Plugin**: For managing Go tools on macOS
