# Python Plugin

> Install and manage Python runtime with system package managers

The Python plugin for Genesis provides automated installation and management of Python across all platforms (macOS, Linux, Windows).

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

The Python plugin automatically installs and configures Python on your system using the system package manager (Homebrew on macOS, APT on Debian/Ubuntu, etc.).

---

## Features

- ✅ **System Package Manager Installation**
  - Homebrew on macOS
  - APT on Debian/Ubuntu Linux
  - YUM/DNF on RedHat/Fedora Linux
  - Manual installation guide for Windows

- ✅ **Cross-Platform Support**
  - macOS (via Homebrew)
  - Linux (via APT/YUM/DNF)
  - Windows (manual installation guide)

- ✅ **Automatic Detection**
  - Checks if Python is already installed
  - Verifies correct version
  - Skips installation if already present

- ✅ **Optimized System Operations**
  - Task registry for deduplication
  - Package manager updates run only once across all plugins
  - Three-phase execution model

- ✅ **Comprehensive Logging**
  - Debug, info, warn, and error levels
  - Color-coded output
  - Detailed progress information

---

## Installation Methods

### System Package Manager (Default)

**Advantages:**
- System-wide installation
- Managed by OS package manager
- Easy updates via package manager
- Consistent with system packages

**Platforms:**
- macOS: Automatic installation via Homebrew
- Linux: Automatic installation via APT/YUM/DNF
- Windows: Manual installation guide provided

---

## Configuration

### Options

```typescript
interface PythonOptions {
  version: string;      // Required: Python version to install
}
```

#### `version` (required)

The Python version to install. Can be:
- Major version: `"3"`, `"3.11"`, `"3.10"`
- Full version: `"3.11.5"`, `"3.10.12"`

**Note:** The exact version format depends on what's available in your system's package manager.

---

## Usage Examples

### TypeScript Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";

export default defineConfig({
  languages: [
    // Install Python 3.11
    python({
      version: "3.11",
    }),
    
    // Install Python 3.10
    python({
      version: "3.10",
    }),
  ],
});
```

### YAML Configuration

```yaml
languages:
  # Install Python 3.11
  - type: python
    version: "3.11"
  
  # Install Python 3.10
  - type: python
    version: "3.10"
```

---

## Platform-Specific Behavior

### macOS

**Installation:**
1. Uses Homebrew package manager
2. Installs Python via: `brew install python@<major-version>`
3. Example: `brew install python@3` for Python 3.x

**Package Name Format:**
- `python@3` for Python 3.x
- `python@3.11` for Python 3.11.x

### Linux (Debian/Ubuntu)

**Installation:**
1. Updates APT package index (deduplicated across plugins)
2. Installs Python via: `apt-get install python<version>`
3. Example: `apt-get install python3.11` for Python 3.11

**Package Name Format:**
- `python3.11` for Python 3.11
- `python3.10` for Python 3.10

### Linux (RedHat/Fedora)

**Installation:**
1. Updates YUM/DNF package index
2. Installs Python via: `yum install python<version>` or `dnf install python<version>`

### Windows

**Installation:**
- Detects Windows platform
- Displays installation guide
- Provides download link to python.org
- **Does NOT attempt automatic installation**
- User must manually download and install

---

## How It Works

The Python plugin uses Genesis's **three-phase execution model** for optimal performance:

### Phase 1: Task Registration

The plugin registers system-level prerequisites:

**macOS:**
1. Registers Homebrew update task: `brew update`
2. Registers Python installation task: `brew install python@<version>`

**Linux:**
1. Registers package manager update task: `apt-get update` or `yum update`
2. Registers Python installation task: `apt-get install python<version>`

**Windows:**
- No system tasks registered (manual installation required)

**Benefits:**
- System tasks are deduplicated across all plugins
- If multiple plugins need package manager updates, it runs only once
- Package manager updates run only once, not per plugin

### Phase 2: System Task Execution

Genesis executes all registered system tasks (deduplicated):
1. Package manager update runs once (shared across all plugins)
2. Python package is installed via system package manager
3. All plugins' system dependencies are now available

### Phase 3: Plugin Installation

The plugin verifies the installation:

#### Detection Check

1. Checks if `python3` or `python` command is available
2. Runs `python3 --version` to get installed version
3. Compares installed version with requested version
4. Returns success if correct version is installed

#### Installation Verification

**macOS/Linux:**
1. System package manager has already installed Python (from Phase 2)
2. Plugin verifies Python is available
3. Logs success message

**Windows:**
1. Displays manual installation guide
2. Provides download link: https://www.python.org/
3. User must install manually

### Phase 4: Validation

1. Reuses detection logic
2. Verifies Python is installed and correct version
3. Runs `python3 --version` to confirm

---

## Troubleshooting

### Python not found after installation

**Problem:** Python command not available after installation.

**Solution:**
```bash
# Verify installation
python3 --version

# Check if python3 is in PATH
which python3

# On macOS with Homebrew
brew info python@3

# On Linux
apt list --installed | grep python
```

### Version mismatch

**Problem:** Different Python version installed than requested.

**Solution:**
```bash
# Check installed version
python3 --version

# On macOS
brew uninstall python@<old-version>
brew install python@<new-version>

# On Linux
sudo apt-get remove python<old-version>
sudo apt-get install python<new-version>
```

### Windows installation

**Problem:** Automatic installation not supported on Windows.

**Solution:**
1. Visit https://www.python.org/downloads/
2. Download Python installer for Windows
3. Run the installer
4. **Important:** Check "Add Python to PATH" during installation
5. Verify: Open new terminal and run `python --version`

### Permission errors on Linux

**Problem:** Permission denied during installation.

**Solution:**
```bash
# APT requires sudo
sudo apt-get update
sudo apt-get install python3.11

# Or run Genesis with sudo (not recommended)
```

### Package not found

**Problem:** Python version not available in package manager.

**Solution:**
```bash
# Check available versions
# On macOS
brew search python

# On Debian/Ubuntu
apt-cache search python3

# Use deadsnakes PPA for more Python versions (Ubuntu)
sudo add-apt-repository ppa:deadsnakes/ppa
sudo apt-get update
sudo apt-get install python3.11
```

---

## Advanced Usage

### Multiple Python Versions

```typescript
// Install multiple Python versions
export default defineConfig({
  languages: [
    python({ version: "3.11" }),  // Python 3.11
    python({ version: "3.10" }),  // Python 3.10
  ],
});
```

### Virtual Environments

After installing Python, create virtual environments:

```bash
# Create virtual environment
python3 -m venv myenv

# Activate virtual environment
source myenv/bin/activate  # macOS/Linux
myenv\Scripts\activate     # Windows

# Install packages
pip install requests flask
```

### Using with Node.js Plugin

The Python and Node.js plugins work together efficiently:

```typescript
export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
  languages: [
    python({ version: "3.11" }),
  ],
});
```

**Optimization:** Both plugins register `apt-get update` (on Linux), but it only runs **once** thanks to task deduplication!

---

## Related Documentation

- [Python Official Website](https://www.python.org/)
- [Python Package Index (PyPI)](https://pypi.org/)
- [Virtual Environments Documentation](https://docs.python.org/3/library/venv.html)
- [Genesis Plugin Architecture](../../README.md#plugin-architecture)
- [Task Deduplication Guide](../../../../docs/TASK_DEDUPLICATION.md)

---

**Part of the [@ossl/genesis-plugins](../../README.md) package**

