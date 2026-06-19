# Go Plugin

Install and manage Go (Golang) programming language for Genesis environments.

## Features

- **Official Downloads**: Downloads official Go distributions from go.dev
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Version Management**: Install specific Go versions (1.19, 1.20, 1.21, 1.22, etc.)
- **Environment Setup**: Automatic PATH configuration guidance
- **Task Deduplication**: Optimized system package installation
- **Clean Installation**: Removes previous versions before installing

## Installation

Add to your Genesis configuration:

### TypeScript Configuration

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

- `version` (string, required): Go version to install (e.g., "1.19", "1.20", "1.21", "1.22")

## Platform Support

### macOS & Linux

- Downloads official Go tar.gz archives from go.dev
- Extracts to `/usr/local/go`
- Automatically adds `/usr/local/go/bin` to PATH
- Cleans up previous installations

### Windows

- Provides detailed installation guide
- Recommends downloading official MSI installer
- Instructions for PATH configuration

## Environment Variables

After installation, set up your Go environment:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export PATH="/usr/local/go/bin:$PATH"
```

## Supported Versions

- Go 1.19
- Go 1.20
- Go 1.21
- Go 1.22
- Latest versions

## Examples

### Install Go 1.21

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
4. Verify installation: `go version`

### Architecture Support

The plugin automatically detects and supports:
- x64 (amd64) - Most common
- ARM64 (arm64) - Apple Silicon, ARM servers

## Performance Notes

- Official Go distributions are optimized for performance
- Archive extraction is fast and efficient
- Task deduplication prevents redundant package manager updates
- Parallel installation with other plugins for optimal performance

## Development Workflow

After installation:

```bash
# Verify installation
go version

# Initialize a new module
go mod init myproject

# Build your project
go build

# Run tests
go test

# Install dependencies
go mod tidy
```

## Integration with Other Tools

The Go plugin integrates seamlessly with:
- Gin web framework
- gRPC development
- Docker containerization
- VS Code Go extension
- GoLand IDE

## Security Considerations

- Downloads are from official Go team servers
- HTTPS downloads ensure secure transfer
- Archive integrity is maintained
- Regular security updates available

## GOPATH and GOROOT

The plugin sets up:
- **GOROOT**: `/usr/local/go` (Go installation directory)
- **GOPATH**: Uses Go defaults (`~/go` or `$HOME/go`)
- **PATH**: Includes `$GOROOT/bin` automatically

## Module Support

The installed Go version includes full Go module support:
- `go mod init` for new modules
- `go get` for dependencies
- `go mod tidy` for cleanup
- `go mod vendor` for vendoring

---

**Part of the [Genesis](../../../README.md) project**
