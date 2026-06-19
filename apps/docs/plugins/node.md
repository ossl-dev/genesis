# Node.js Plugin

Install and manage Node.js runtime with NVM or standalone installation.

## Overview

The Node.js plugin provides automated installation and management of Node.js across all platforms. By default, it uses **NVM (Node Version Manager)** for installation, which is the recommended approach for managing multiple Node.js versions.

## Installation

```bash
bun add -D @ossl/genesis-plugins
```

## Usage

### TypeScript

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({
      version: "20",
      use_nvm: true,
    }),
  ],
});
```

### YAML

```yaml
tools:
  - type: node
    version: "20"
    use_nvm: true
```

## Options

```typescript
interface NodeOptions {
  version: string; // Required: Node.js version to install
  use_nvm?: boolean; // Optional: Use NVM (default: true)
  global_packages?: string[]; // Optional: Global packages to install after setup
}
```

### `version` (required)

The Node.js version to install. Can be:

- Major version: `"20"`, `"18"`, `"16"`
- Full version: `"20.10.0"`, `"18.16.0"`

### `use_nvm` (optional)

Whether to use NVM for Node.js installation. Default: `true`.

- `true`: Install Node.js via NVM (recommended)
- `false`: Use standalone installation (not yet implemented)

### `global_packages` (optional)

List of global packages to install after Node.js setup is complete. Supports npm, yarn, pnpm, and bun packages.

```typescript
node({
  version: "20",
  global_packages: [
    "typescript", // TypeScript compiler
    "ts-node", // TypeScript runtime
    "nodemon", // Auto-restart for Node.js
    "eslint", // Code linting
    "prettier", // Code formatting
  ],
});
```

The plugin will automatically detect which package managers are available and install packages using the first one found (npm → yarn → pnpm → bun).

## Installation Methods

### NVM (Recommended, Default)

**Advantages:**

- ✅ Manage multiple Node.js versions
- ✅ Easy version switching
- ✅ Per-project version configuration
- ✅ Isolated installations
- ✅ Shell integration

**Platforms:**

- macOS: Automatic installation
- Linux: Automatic installation
- Windows: Manual installation guide provided

**Example:**

```typescript
node({
  version: "20",
  use_nvm: true, // Default
});
```

### Standalone

**Advantages:**

- ✅ System-wide installation
- ✅ Simpler setup
- ✅ No version manager overhead

**Platforms:**

- All platforms supported

**Example:**

```typescript
node({
  version: "18",
  use_nvm: false,
});
```

## Platform-Specific Behavior

### macOS

**NVM Installation:**

1. Registers system tasks (package manager update, curl installation)
2. Downloads NVM install script from official repository
3. Executes installation script
4. Installs Node.js using `nvm install <version>`
5. Sets installed version as default: `nvm alias default <version>`
6. Integrates with shell profile (`.bashrc`, `.zshrc`)

**Standalone Installation:**

- Downloads Node.js installer from nodejs.org
- Runs installer package

### Linux

**NVM Installation:**

1. Registers system tasks (apt-get update, curl installation)
2. Downloads NVM install script from official repository
3. Executes installation script
4. Installs Node.js using `nvm install <version>`
5. Sets installed version as default: `nvm alias default <version>`
6. Integrates with shell profile (`.bashrc`, `.zshrc`)

**Standalone Installation:**

- Uses package manager (apt, yum, etc.)
- Or downloads binary from nodejs.org

### Windows

**NVM Installation:**

- Detects Windows platform
- Displays comprehensive installation guide for nvm-windows
- Provides download link and step-by-step instructions
- **Does NOT attempt automatic installation**

**Standalone Installation:**

- Downloads Node.js installer from nodejs.org
- Runs MSI installer

## How It Works

The Node.js plugin uses Genesis's three-phase execution model:

### Phase 1: Task Registration

Registers system-level prerequisites:

**macOS/Linux with NVM:**

```
Registering system tasks...
  - Package manager update (deduplicated!)
  - curl installation
```

**Windows or Standalone:**

- No system tasks registered

### Phase 2: System Task Execution

System tasks execute (deduplicated):

```
Executing system tasks...
  ✓ Update APT package index (runs once!)
  ✓ Install curl
```

### Phase 3: Plugin Installation

Performs NVM and Node.js installation:

**macOS/Linux:**

```
Installing plugins...
  ℹ node: Checking if NVM is installed...
  ℹ node: Installing NVM...
  ✓ node: NVM installed successfully
  ℹ node: Installing Node.js 20...
  ✓ node: Node.js 20.10.0 installed
  ✓ node: Set Node.js 20 as default
```

**Windows:**

```
Installing plugins...
  ℹ node: Windows detected
  ℹ node: Please install nvm-windows manually:

  1. Download from: https://github.com/coreybutler/nvm-windows/releases
  2. Run the installer
  3. Open new terminal
  4. Run: nvm install 20
  5. Run: nvm use 20
```

## Examples

### Single Version

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({
      version: "20",
      use_nvm: true,
    }),
  ],
});
```

### Multiple Versions

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }), // Latest Node 20
    node({ version: "18", use_nvm: true }), // Latest Node 18
  ],
});
```

### With Other Plugins

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [node({ version: "20", use_nvm: true })],
  languages: [python({ version: "3.11" })],
});
```

**Optimization:** Both plugins register package manager updates, but it runs only **once** thanks to task deduplication!

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

## Advanced Usage

### Project-Specific Version

Create `.nvmrc` in your project:

```
20.10.0
```

Then use:

```bash
nvm use
```

### Environment Variables

```typescript
const nodeVersion = process.env.NODE_VERSION || "20";

export default defineConfig({
  tools: [node({ version: nodeVersion, use_nvm: true })],
});
```

## Related Documentation

- [NVM Official Documentation](https://github.com/nvm-sh/nvm)
- [nvm-windows Documentation](https://github.com/coreybutler/nvm-windows)
- [Node.js Official Website](https://nodejs.org/)
- [Task Registry](/guide/task-registry) - Learn about task deduplication
