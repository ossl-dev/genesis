# Git Plugin

Install and manage Git version control system for Genesis environments.

## Features

- **Multiple Installation Methods**: Package manager, source, or binary installation
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Version Management**: Install specific Git versions or latest
- **Default Configuration**: Sets sensible Git defaults
- **Task Deduplication**: Optimized system package installation
- **Build Dependencies**: Automatically installs required build tools

## Installation

Add to your Genesis configuration:

### TypeScript Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { git } from "@ossl/genesis-plugins/git";

export default defineConfig({
  tools: [
    git({
      version: "latest",           // Optional: specific version or "latest"
      install_method: "package",    // Optional: "package", "source", or "binary"
    }),
  ],
});
```

### YAML Configuration

```yaml
tools:
  - type: git
    version: "latest"              # Optional: specific version or "latest"
    install_method: "package"      # Optional: "package", "source", or "binary"
```

## Options

- `version` (string, optional): Git version to install (e.g., "2.40.0", "latest", default: "latest")
- `install_method` (string, optional): Installation method ("package", "source", "binary", default: "package")

## Installation Methods

### Package Manager (Recommended)

**macOS**: Uses Homebrew to install Git
**Linux**: Uses APT/YUM to install git-all package

**Advantages**:
- Fastest installation
- Automatic updates
- Managed dependencies
- Stable and tested

### Source Installation

Builds Git from source code for maximum control.

**Advantages**:
- Latest features
- Custom build options
- Optimized for system
- Educational value

**Requirements**:
- Build tools (make, gcc, autoconf)
- Development libraries
- More time and resources

### Binary Installation

Downloads pre-compiled Git binaries.

**Advantages**:
- Faster than source compilation
- Latest versions available
- No compilation required
- Cross-platform consistency

## Platform Support

### macOS

- **Package Manager**: `brew install git`
- **Source**: Builds from source with Homebrew dependencies
- **Binary**: Downloads official macOS binaries

### Linux

- **Package Manager**: `sudo apt install git-all` or equivalent
- **Source**: Builds from source with APT dependencies
- **Binary**: Downloads official Linux binaries

### Windows

- Provides detailed installation guide
- Recommends Git for Windows installer
- Includes Git Bash integration

## Default Configuration

The plugin automatically configures Git with sensible defaults:

```bash
git config --global init.defaultBranch main
git config --global pull.rebase false
```

## Supported Versions

- Git 2.30+
- Latest releases
- Specific version numbers

## Examples

### Basic Git Installation

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { git } from "@ossl/genesis-plugins/git";

export default defineConfig({
  tools: [
    git(),  // Uses defaults: latest version, package manager
  ],
});
```

### Specific Version Installation

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { git } from "@ossl/genesis-plugins/git";

export default defineConfig({
  tools: [
    git({
      version: "2.40.0",
      install_method: "binary",
    }),
  ],
});
```

### Source Installation

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { git } from "@ossl/genesis-plugins/git";

export default defineConfig({
  tools: [
    git({
      install_method: "source",
    }),
  ],
});
```

## Troubleshooting

### Git Not Found After Installation

1. Ensure Git is in your PATH
2. Restart your terminal
3. Check installation: `which git`

### Version Mismatch

1. Check active version: `git --version`
2. Verify installation method
3. Remove old versions if necessary

### Permission Issues

1. Use sudo for system-wide installation
2. Ensure proper permissions
3. Consider user-specific installation

### Windows Installation

1. Download from https://git-scm.com/download/win
2. Run the installer with default settings
3. Verify installation: `git --version`

### Build Failures (Source Installation)

1. Install required build dependencies
2. Check system compatibility
3. Review build logs for errors

## Performance Notes

- Package manager installation is fastest
- Source installation takes more time but offers customization
- Binary installation balances speed and control
- Task deduplication prevents redundant operations

## Security Considerations

- Downloads are from official Git sources
- HTTPS transfers ensure security
- Package manager installations are signed
- Regular security updates available

## Integration with Development Workflow

After installation:

```bash
# Verify installation
git --version

# Configure your identity
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"

# Initialize a repository
git init

# Clone a repository
git clone https://github.com/user/repo.git
```

## Advanced Configuration

The plugin sets basic defaults, but you can customize further:

```bash
# Set your preferred editor
git config --global core.editor "vim"

# Enable color output
git config --global color.ui auto

# Set default push behavior
git config --global push.default simple

# Configure credential helper
git config --global credential.helper store
```

## SSH Setup

For Git operations over SSH:

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "your.email@example.com"

# Add SSH agent
ssh-add ~/.ssh/id_ed25519

# Test SSH connection
ssh -T git@github.com
```

## Integration with Other Tools

The Git plugin integrates seamlessly with:
- GitHub, GitLab, Bitbucket
- IDEs (VS Code, IntelliJ, etc.)
- CI/CD platforms
- Code review tools
- Git hosting services

---

**Part of the [Genesis](../../../README.md) project**
