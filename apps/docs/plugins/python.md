# Python Plugin

Install and manage Python runtime with system package managers.

## Overview

The Python plugin provides automated installation and management of Python across all platforms using the system package manager (Homebrew on macOS, APT on Debian/Ubuntu, etc.).

## Installation

```bash
bun add -D @ossl/genesis-plugins
```

## Usage

### TypeScript

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";

export default defineConfig({
  languages: [
    python({
      version: "3.11",
    }),
  ],
});
```

### YAML

```yaml
languages:
  - type: python
    version: "3.11"
```

## Options

```typescript
interface PythonOptions {
  version: string;      // Required: Python version to install
}
```

### `version` (required)

The Python version to install. Can be:
- Major version: `"3"`, `"3.11"`, `"3.10"`
- Full version: `"3.11.5"`, `"3.10.12"`

**Note:** The exact version format depends on what's available in your system's package manager.

## Platform-Specific Behavior

### macOS

**Installation:**
1. Registers Homebrew update task (deduplicated!)
2. Installs Python via: `brew install python@<major-version>`
3. Example: `brew install python@3` for Python 3.x

**Package Name Format:**
- `python@3` for Python 3.x
- `python@3.11` for Python 3.11.x

### Linux (Debian/Ubuntu)

**Installation:**
1. Registers APT update task (deduplicated!)
2. Installs Python via: `apt-get install python<version>`
3. Example: `apt-get install python3.11` for Python 3.11

**Package Name Format:**
- `python3.11` for Python 3.11
- `python3.10` for Python 3.10

### Linux (RedHat/Fedora)

**Installation:**
1. Registers YUM/DNF update task
2. Installs Python via: `yum install python<version>` or `dnf install python<version>`

### Windows

**Installation:**
- Detects Windows platform
- Displays installation guide
- Provides download link to python.org
- **Does NOT attempt automatic installation**

## How It Works

The Python plugin uses Genesis's three-phase execution model:

### Phase 1: Task Registration

Registers system-level prerequisites:

**macOS:**
```
Registering system tasks...
  - Homebrew update (deduplicated!)
  - Python installation via Homebrew
```

**Linux:**
```
Registering system tasks...
  - Package manager update (deduplicated!)
  - Python installation via APT/YUM
```

### Phase 2: System Task Execution

System tasks execute (deduplicated):

```
Executing system tasks...
  ✓ Update APT package index (runs once!)
  ✓ Install python3.11
```

### Phase 3: Plugin Installation

Verifies the installation:

**macOS/Linux:**
```
Installing plugins...
  ℹ python: Verifying Python installation...
  ✓ python: Python 3.11.5 verified
```

**Windows:**
```
Installing plugins...
  ℹ python: Windows detected
  ℹ python: Please install Python manually:
  
  1. Visit: https://www.python.org/downloads/
  2. Download Python installer
  3. Run the installer
  4. Check "Add Python to PATH"
  5. Verify: python --version
```

## Examples

### Single Version

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";

export default defineConfig({
  languages: [
    python({
      version: "3.11",
    }),
  ],
});
```

### Multiple Versions

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";

export default defineConfig({
  languages: [
    python({ version: "3.11" }),
    python({ version: "3.10" }),
  ],
});
```

### With Other Plugins

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
  languages: [
    python({ version: "3.11" }),
  ],
});
```

**Optimization:** Both plugins register package manager updates (on Linux), but it runs only **once** thanks to task deduplication!

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

## Advanced Usage

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

### Environment Variables

```typescript
const pythonVersion = process.env.PYTHON_VERSION || "3.11";

export default defineConfig({
  languages: [
    python({ version: pythonVersion }),
  ],
});
```

### Conditional Installation

```typescript
const isDevelopment = process.env.NODE_ENV === "development";

export default defineConfig({
  languages: [
    // Only install Python in development
    ...(isDevelopment ? [python({ version: "3.11" })] : []),
  ],
});
```

## Related Documentation

- [Python Official Website](https://www.python.org/)
- [Python Package Index (PyPI)](https://pypi.org/)
- [Virtual Environments Documentation](https://docs.python.org/3/library/venv.html)
- [Task Registry](/guide/task-registry) - Learn about task deduplication

