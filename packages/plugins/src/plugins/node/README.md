# Node.js Plugin

> Install and manage Node.js runtime with NVM or standalone installation

The Node.js plugin for Genesis provides automated installation and management of Node.js across all platforms (macOS, Linux, Windows).

---

## 📑 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Installation Methods](#installation-methods)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Platform-Specific Behavior](#platform-specific-behavior)
- [How It Works](#how-it-works)
- [Troubleshooting](#troubleshooting)

---

## Overview

The Node.js plugin automatically installs and configures Node.js on your system. By default, it uses **NVM (Node Version Manager)** for installation, which is the recommended approach for managing multiple Node.js versions.

---

## Features

- ✅ **NVM Installation** (default, recommended)
  - Automatic NVM installation on macOS/Linux
  - Guided installation for Windows (nvm-windows)
  - Version management and switching
  - Shell profile integration

- ✅ **Standalone Installation** (optional)
  - Direct Node.js installation without NVM
  - System-wide installation

- ✅ **Cross-Platform Support**
  - macOS (via NVM)
  - Linux (via NVM)
  - Windows (via nvm-windows with manual installation guide)

- ✅ **Automatic Detection**
  - Checks if Node.js is already installed
  - Verifies correct version
  - Skips installation if already present

- ✅ **Comprehensive Logging**
  - Debug, info, warn, and error levels
  - Color-coded output
  - Detailed progress information

- ✅ **Optimized System Operations**
  - Task registry for deduplication
  - System dependencies installed once across all plugins
  - Three-phase execution model

---

## Installation Methods

### NVM (Recommended, Default)

**Advantages:**
- Manage multiple Node.js versions
- Easy version switching
- Per-project version configuration
- Isolated installations
- Shell integration

**Platforms:**
- macOS: Automatic installation
- Linux: Automatic installation
- Windows: Manual installation guide provided

### Standalone

**Advantages:**
- System-wide installation
- Simpler setup
- No version manager overhead

**Platforms:**
- All platforms supported

---

## Configuration

### Options

```typescript
interface NodeOptions {
  version: string;      // Required: Node.js version to install
  use_nvm?: boolean;    // Optional: Use NVM (default: true)
}
```

#### `version` (required)

The Node.js version to install. Can be:
- Major version: `"20"`, `"18"`, `"16"`
- Full version: `"20.10.0"`, `"18.16.0"`

#### `use_nvm` (optional)

Whether to use NVM for installation.
- Default: `true`
- Set to `false` for standalone installation

---

## Usage Examples

### TypeScript Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    // Default: NVM installation
    node({
      version: "20",
    }),
    
    // Explicit NVM installation
    node({
      version: "20",
      use_nvm: true,
    }),
    
    // Standalone installation
    node({
      version: "18",
      use_nvm: false,
    }),
  ],
});
```

### YAML Configuration

```yaml
tools:
  # Default: NVM installation
  - type: node
    version: "20"
  
  # Explicit NVM installation
  - type: node
    version: "20"
    use_nvm: true
  
  # Standalone installation
  - type: node
    version: "18"
    use_nvm: false
```

---

## Platform-Specific Behavior

### macOS

**NVM Installation:**
1. Downloads NVM install script from official repository
2. Executes installation script
3. Installs Node.js using `nvm install <version>`
4. Sets installed version as default: `nvm alias default <version>`
5. Integrates with shell profile (`.bashrc`, `.zshrc`)

**Standalone Installation:**
- Downloads Node.js installer from nodejs.org
- Runs installer package

### Linux

**NVM Installation:**
1. Downloads NVM install script from official repository
2. Executes installation script
3. Installs Node.js using `nvm install <version>`
4. Sets installed version as default: `nvm alias default <version>`
5. Integrates with shell profile (`.bashrc`, `.zshrc`)

**Standalone Installation:**
- Uses package manager (apt, yum, etc.)
- Or downloads binary from nodejs.org

### Windows

**NVM Installation:**
- Detects Windows platform
- Displays comprehensive installation guide for nvm-windows
- Provides download link and step-by-step instructions
- **Does NOT attempt automatic installation** (as requested)
- Logs guide to console for user to follow

**Standalone Installation:**
- Downloads Node.js installer from nodejs.org
- Runs MSI installer

---

## How It Works

The Node.js plugin uses Genesis's **three-phase execution model** for optimal performance:

### Phase 1: Task Registration

The plugin registers system-level prerequisites that need to be installed before NVM can be used:

**macOS/Linux with NVM:**
1. Registers package manager update task (e.g., `apt-get update`)
2. Registers `curl` installation task (needed to download NVM install script)

**Windows or Standalone:**
- No system tasks registered (manual installation or direct download)

**Benefits:**
- System tasks are deduplicated across all plugins
- If multiple plugins need `curl`, it's only installed once
- Package manager updates run only once, not per plugin

### Phase 2: System Task Execution

Genesis executes all registered system tasks (deduplicated):
1. Package manager update runs once
2. System packages (like `curl`) are installed
3. All plugins' system dependencies are now available

### Phase 3: Plugin Installation

The plugin performs its specific installation work:

#### Detection Check

1. Checks if `node` command is available
2. Runs `node --version` to get installed version
3. Compares installed version with requested version
4. Skips installation if already correct

#### With NVM (`use_nvm: true`)

**macOS/Linux:**
1. Check if NVM is installed
2. If not, download and install NVM (using `curl` from Phase 2)
3. Source NVM in current shell
4. Install Node.js: `nvm install <version>`
5. Set as default: `nvm alias default <version>`
6. Verify installation

**Windows:**
1. Detect Windows platform
2. Log comprehensive installation guide
3. Provide nvm-windows download link
4. Explain manual installation steps
5. Do NOT attempt automatic installation

#### Without NVM (`use_nvm: false`)

1. Download Node.js installer for platform
2. Run installer
3. Verify installation

### Phase 4: Validation

1. Reuses detection logic
2. Verifies Node.js is installed and correct version

---

## Troubleshooting

### NVM not found after installation

**Problem:** NVM command not available after installation.

**Solution:**
```bash
# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc

# Or restart your terminal
```

### Node version mismatch

**Problem:** Different Node.js version installed than requested.

**Solution:**
```bash
# With NVM
nvm install <version>
nvm alias default <version>
nvm use <version>

# Verify
node --version
```

### Windows NVM installation

**Problem:** Automatic installation not supported on Windows.

**Solution:**
1. Follow the installation guide logged by Genesis
2. Download nvm-windows from: https://github.com/coreybutler/nvm-windows/releases
3. Run the installer
4. Open new terminal
5. Run: `nvm install <version>`
6. Run: `nvm use <version>`

### Permission errors on macOS/Linux

**Problem:** Permission denied during installation.

**Solution:**
```bash
# NVM should NOT require sudo
# If you see permission errors, check NVM installation

# Reinstall NVM
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash

# Reload shell
source ~/.bashrc  # or ~/.zshrc
```

### Network errors

**Problem:** Failed to download NVM or Node.js.

**Solution:**
- Check internet connection
- Check firewall settings
- Try again (Genesis is idempotent)
- Use VPN if behind corporate firewall

### Shell profile not updated

**Problem:** NVM not available in new terminals.

**Solution:**
```bash
# Manually add to ~/.bashrc or ~/.zshrc
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
```

---

## Advanced Usage

### Multiple Node.js Versions

```typescript
// Install multiple versions
export default defineConfig({
  tools: [
    node({ version: "20" }),  // Latest Node 20
    node({ version: "18" }),  // Latest Node 18
  ],
});
```

### Project-Specific Version

Create `.nvmrc` in your project:
```
20.10.0
```

Then use:
```bash
nvm use
```

---

## Related Documentation

- [NVM Official Documentation](https://github.com/nvm-sh/nvm)
- [nvm-windows Documentation](https://github.com/coreybutler/nvm-windows)
- [Node.js Official Website](https://nodejs.org/)
- [Genesis Plugin Architecture](../../README.md#plugin-architecture)

---

**Part of the [@ossl/genesis-plugins](../../README.md) package**
