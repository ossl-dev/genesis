# Homebrew Plugin

Install and manage Homebrew package manager for macOS environments.

## Features

- **macOS Native**: Designed specifically for macOS systems
- **Architecture Support**: Supports both Intel and Apple Silicon Macs
- **Automatic Updates**: Updates Homebrew and installed packages
- **Cask Support**: Includes Homebrew Cask for application management
- **PATH Configuration**: Automatic PATH setup guidance
- **Task Deduplication**: Optimized system package installation

## Installation

Add to your Genesis configuration:

### TypeScript Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { homebrew } from "@ossl/genesis-plugins/homebrew";

export default defineConfig({
  tools: [
    homebrew({
      update_packages: true,    // Optional: update installed packages
      install_cask: true,       // Optional: include Homebrew Cask
      add_to_path: true,        // Optional: add Homebrew to PATH
    }),
  ],
});
```

### YAML Configuration

```yaml
tools:
  - type: homebrew
    update_packages: true       # Optional: update installed packages
    install_cask: true          # Optional: include Homebrew Cask
    add_to_path: true           # Optional: add Homebrew to PATH
```

## Options

- `update_packages` (boolean, optional): Update installed packages after installation (default: true)
- `install_cask` (boolean, optional): Include Homebrew Cask for GUI applications (default: true)
- `add_to_path` (boolean, optional): Add Homebrew to PATH automatically (default: true)

## Platform Support

### macOS

**Apple Silicon (M1/M2/M3)**:
- Installs to `/opt/homebrew`
- Uses ARM64-optimized packages
- Automatic PATH configuration

**Intel Macs**:
- Installs to `/usr/local`
- Uses x86_64 packages
- Automatic PATH configuration

### Linux & Windows

- Provides alternative package manager recommendations
- Suggests appropriate alternatives for each platform
- Manual installation guidance

## Architecture Detection

The plugin automatically detects your Mac's architecture:

```bash
# Check your architecture
uname -m

# Output:
# arm64 = Apple Silicon
# x86_64 = Intel
```

## Installation Paths

### Apple Silicon Macs

- **Homebrew**: `/opt/homebrew`
- **PATH**: `/opt/homebrew/bin`
- **Shell Environment**: `eval "$(/opt/homebrew/bin/brew shellenv)"`

### Intel Macs

- **Homebrew**: `/usr/local/Homebrew`
- **PATH**: `/usr/local/bin`
- **Shell Environment**: `eval "$(/usr/local/bin/brew shellenv)"`

## Examples

### Basic Homebrew Installation

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { homebrew } from "@ossl/genesis-plugins/homebrew";

export default defineConfig({
  tools: [
    homebrew(),  // Uses defaults: update packages, include cask, add to PATH
  ],
});
```

### Minimal Installation

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { homebrew } from "@ossl/genesis-plugins/homebrew";

export default defineConfig({
  tools: [
    homebrew({
      update_packages: false,
      install_cask: false,
      add_to_path: false,
    }),
  ],
});
```

## Post-Installation Setup

After Homebrew installation, the plugin provides shell configuration commands:

### For Apple Silicon Macs

Add to your shell profile (`~/.zshrc` or `~/.bashrc`):

```bash
if [[ "$(uname -m)" == "arm64" ]]; then
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi
```

### For Intel Macs

Add to your shell profile:

```bash
eval "$(/usr/local/bin/brew shellenv)"
```

Then restart your shell or run:

```bash
source ~/.zshrc  # or ~/.bashrc
```

## Usage Examples

After installation:

```bash
# Verify Homebrew installation
brew --version

# Update Homebrew
brew update

# Install a package
brew install node

# Install a GUI application
brew install --cask visual-studio-code

# List installed packages
brew list

# Search for packages
brew search git

# Upgrade all packages
brew upgrade

# Remove a package
brew uninstall node
```

## Homebrew vs Homebrew Cask

### Homebrew (Command Line Tools)

For command-line tools and utilities:

```bash
# Development tools
brew install git
brew install node
brew install python

# System utilities
brew install curl
brew install wget
brew install htop
```

### Homebrew Cask (GUI Applications)

For graphical applications:

```bash
# Development tools
brew install --cask visual-studio-code
brew install --cask intellij-idea

# Productivity
brew install --cask slack
brew install --cask zoom

# Browsers
brew install --cask google-chrome
brew install --cask firefox
```

## Troubleshooting

### Homebrew Not Found After Installation

1. Ensure PATH is configured correctly
2. Restart your terminal or source your shell profile
3. Check installation directory: `ls -la /opt/homebrew` or `ls -la /usr/local`

### Permission Issues

```bash
# Fix Homebrew permissions
sudo chown -R $(whoami) /opt/homebrew  # Apple Silicon
# or
sudo chown -R $(whoami) /usr/local     # Intel
```

### Architecture Mismatch

If you're using Rosetta on an Apple Silicon Mac:

```bash
# Check if running under Rosetta
uname -m

# If it shows x86_64 on an ARM Mac, you're in Rosetta
# Exit Rosetta and use native ARM64 Homebrew
arch -arm64 brew install package-name
```

### Update Issues

```bash
# Reset Homebrew if updates fail
brew doctor
brew update-reset

# Or clean up and reinstall
brew cleanup
```

## Performance Notes

- Native ARM64 packages are faster on Apple Silicon
- Homebrew automatically detects architecture
- Package updates are optimized for speed
- Task deduplication prevents redundant operations

## Security Considerations

- Downloads are from official Homebrew sources
- HTTPS transfers ensure security
- Package integrity is verified
- Regular security updates available

## Integration with Development Workflow

Homebrew is essential for macOS development:

```bash
# Common development setup
brew install git
brew install node
brew install python
brew install docker
brew install --cask visual-studio-code

# Database setup
brew install postgresql
brew install redis
brew install mongodb

# Mobile development
brew install --cask xcode
brew install --cask android-studio
```

## Advanced Configuration

### Custom Taps

Add additional package repositories:

```bash
# Add custom tap
brew tap some-user/some-repo

# Install from tap
brew install some-formula
```

### Environment Variables

Configure Homebrew behavior:

```bash
# Set custom install prefix
export HOMEBREW_PREFIX="/opt/homebrew"

# Disable analytics
export HOMEBREW_NO_ANALYTICS=1

# Set custom cache directory
export HOMEBREW_CACHE="$HOME/.cache/homebrew"
```

## Package Management

### Finding Packages

```bash
# Search for packages
brew search git
brew search --cask chrome

# Get package information
brew info git
brew info --cask visual-studio-code
```

### Dependency Management

```bash
# See package dependencies
brew deps git

# Install with dependencies
brew install --include-test git

# Check for outdated packages
brew outdated
```

## Integration with Other Tools

The Homebrew plugin integrates seamlessly with:
- Development environments (VS Code, IntelliJ)
- Container tools (Docker, Colima)
- Version control (Git)
- Programming languages (Node.js, Python, Go)
- Database systems (PostgreSQL, Redis, MongoDB)

---

**Part of the [Genesis](../../../README.md) project**
