# Git Plugin

Install and manage Git version control system for Genesis environments.

## Overview

The Git plugin provides comprehensive Git installation and management for Genesis environments. It supports multiple installation methods including package manager, source compilation, and binary installation across macOS, Linux, and Windows platforms.

## Features

- **Multiple Installation Methods**: Package manager, source, or binary installation
- **Cross-Platform Support**: Works on macOS, Linux, and Windows
- **Version Management**: Install specific Git versions or latest
- **Default Configuration**: Sets sensible Git defaults
- **Task Deduplication**: Optimized system package installation
- **Build Dependencies**: Automatically installs required build tools

## Installation

### Basic Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { git } from "@ossl/genesis-plugins/git";

export default defineConfig({
  tools: [
    git({
      version: "latest",
      install_method: "package",
    }),
  ],
});
```

### YAML Configuration

```yaml
tools:
  - type: git
    version: "latest"
    install_method: "package"
```

## Options

| Option | Type | Required | Default | Description |
|---------|--------|----------|-------------|
| `version` | string | No | "latest" | Git version to install (e.g., "2.40.0", "latest") |
| `install_method` | string | No | "package" | Installation method: "package", "source", or "binary" |

## Installation Methods

### Package Manager (Recommended)

**Advantages**:
- Fastest installation
- Automatic updates
- Managed dependencies
- Stable and tested

**Use Cases**:
- Development environments
- CI/CD pipelines
- Production servers

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

**Package Manager**: Uses Homebrew to install git
**Source**: Builds from source with Homebrew dependencies
**Binary**: Downloads official macOS binaries

### Linux

**Package Manager**: Uses APT/YUM to install git-all package
**Source**: Builds from source with APT dependencies
**Binary**: Downloads official Linux binaries

### Windows

Provides detailed installation guide:
- Recommends Git for Windows installer
- Includes Git Bash integration
- Manual installation required

## Default Configuration

The plugin automatically configures Git with sensible defaults:

```bash
git config --global init.defaultBranch main
git config --global pull.rebase false
```

## Usage Examples

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

## Development Workflow

After installation:

```bash
# Verify Git installation
git --version

# Configure your identity
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"

# Initialize a repository
git init

# Clone a repository
git clone https://github.com/user/repo.git

# Check status
git status

# Add files
git add .

# Commit changes
git commit -m "Initial commit"

# Push to remote
git push origin main
```

## Advanced Configuration

### Git Configuration

```bash
# Set your preferred editor
git config --global core.editor "vim"

# Enable color output
git config --global color.ui auto

# Set default push behavior
git config --global push.default simple

# Configure credential helper
git config --global credential.helper store

# Set up GPG signing
git config --global commit.gpgsign true
git config --global user.signingkey YOUR_KEY_ID
```

### SSH Setup

For Git operations over SSH:

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "your.email@example.com"

# Add SSH agent
ssh-add ~/.ssh/id_ed25519

# Test SSH connection
ssh -T git@github.com
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

```bash
# Fix Git permissions
sudo chown -R $(whoami) /usr/local/git  # macOS
# or
sudo chown -R $(whoami) /usr/local     # Linux
```

### Build Failures (Source Installation)

1. Install required build dependencies
2. Check system compatibility
3. Review build logs for errors

### Windows Installation

1. Download from https://git-scm.com/download/win
2. Run the installer with default settings
3. Verify installation: `git --version`

## Git Workflow

### Branching Strategy

```bash
# Create feature branch
git checkout -b feature/new-feature

# Switch to main branch
git checkout main

# Merge feature branch
git merge feature/new-feature

# Delete branch
git branch -d feature/new-feature
```

### Remote Management

```bash
# Add remote repository
git remote add origin https://github.com/user/repo.git

# View remotes
git remote -v

# Fetch from remote
git fetch origin

# Push to remote
git push origin main

# Pull from remote
git pull origin main
```

## Integration with Development Tools

The Git plugin integrates seamlessly with:
- **GitHub**: Web-based Git repository hosting
- **GitLab**: Self-hosted Git management
- **Bitbucket**: Atlassian's Git solution
- **VS Code**: Built-in Git integration
- **IntelliJ IDEA**: Advanced Git support
- **SourceTree**: GUI Git client
- **GitHub Desktop**: Official GitHub client

## Best Practices

1. **Commit Often**: Make small, frequent commits
2. **Write Good Messages**: Use clear, descriptive commit messages
3. **Branch Strategically**: Use feature branches for development
4. **Review Code**: Use pull requests for code review
5. **Backup Remotes**: Keep multiple remote backups

## Git Hooks

Automate Git operations with hooks:

```bash
# Pre-commit hook
#!/bin/sh
go test ./...
npm test
echo "Running tests before commit..."

# Pre-push hook
#!/bin/sh
go test ./...
npm run lint
echo "Running tests before push..."
```

## Security Considerations

- Downloads are from official Git sources
- HTTPS transfers ensure security
- Package manager installations are signed
- Regular security updates available

## Performance Notes

- Package manager installation is fastest
- Source installation takes more time but offers customization
- Binary installation balances speed and control
- Task deduplication prevents redundant operations

## Related Plugins

- **GitHub Plugin**: For GitHub-specific workflows
- **Node.js Plugin**: For frontend development
- **Docker Plugin**: For containerized applications
- **SSH Plugin**: For secure Git operations
