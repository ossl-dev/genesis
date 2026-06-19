# Task Registry

Learn how Genesis's task registry eliminates redundant operations and optimizes environment setup.

## The Problem

When installing multiple plugins, many require the same system-level operations:

```bash
# Installing Node.js plugin
sudo apt-get update  # ← Run 1
sudo apt-get install curl
# ... install NVM

# Installing Python plugin  
sudo apt-get update  # ← Run 2 (redundant!)
sudo apt-get install python3.11
```

**Problems:**
- ⏰ Wastes time running the same command multiple times
- 🔄 Inefficient, especially on slow connections
- 📦 Package manager updates can take 30+ seconds each

## The Solution

Genesis's **Task Registry** deduplicates system-level operations:

```bash
# Phase 1: Register tasks
# - Node plugin registers: apt-update, install-curl
# - Python plugin registers: apt-update, install-python3.11

# Phase 2: Execute tasks (deduplicated!)
sudo apt-get update  # ← Runs ONCE!
sudo apt-get install curl
sudo apt-get install python3.11

# Phase 3: Plugin installation
# - Install NVM using curl
# - Verify Python installation
```

**Benefits:**
- ⚡ **50% faster** - Package manager updates run once
- 🎯 **Guaranteed dependencies** - System packages available when needed
- 🧹 **Clean execution** - No redundant operations

## How It Works

### Three-Phase Execution Model

Genesis uses a three-phase execution model:

#### Phase 1: Task Registration

Plugins register system-level prerequisites:

```typescript
// Node.js plugin
async registerTasks(runtime) {
  const { taskRegistry } = runtime.context;
  
  // Register package manager update
  taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)
  );
  
  // Register curl installation
  taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}
```

```typescript
// Python plugin
async registerTasks(runtime) {
  const { taskRegistry } = runtime.context;
  
  // Register package manager update (same ID!)
  taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)
  );
  
  // Register Python installation
  taskRegistry.register(
    createPackageInstallTask("python3.11", cwd, env)
  );
}
```

#### Phase 2: Task Execution

Task registry executes all tasks with automatic deduplication:

```
Executing system tasks...
  ✓ linux:package-manager:apt-update (deduplicated - runs once!)
  ✓ linux:package:install:curl
  ✓ linux:package:install:python3.11
```

#### Phase 3: Plugin Installation

Plugins perform their specific installation work:

```
Installing plugins...
  ✓ node: NVM installed using curl
  ✓ python: Python 3.11 verified
```

## Task Identification

Tasks are identified by unique IDs:

### Task ID Format

```
platform:category:operation[:parameter]
```

**Examples:**
- `linux:package-manager:apt-update`
- `linux:package:install:curl`
- `macos:package-manager:brew-update`
- `macos:package:install:python@3`

### Deduplication Logic

Tasks with the same ID are deduplicated:

```typescript
// Both plugins register this task
const taskId = "linux:package-manager:apt-update";

// Task registry ensures it runs only once
taskRegistry.register({
  id: taskId,
  description: "Update APT package index",
  executor: async () => {
    await runCommand("sudo", ["apt-get", "update"], { cwd, env });
  },
});
```

## Task Properties

### Task Interface

```typescript
interface Task {
  id: TaskId;                    // Unique identifier
  description: string;           // Human-readable description
  executor: TaskExecutor;        // Function to execute
  priority?: number;             // Execution priority (higher = first)
  dependsOn?: TaskId[];         // Task dependencies
}
```

### Task Priorities

Tasks execute in priority order:

```typescript
const PRIORITIES = {
  CHECK: 200,      // Checks (e.g., command exists)
  UPDATE: 100,     // Package manager updates
  INSTALL: 50,     // Package installations
  CUSTOM: 0,       // Custom tasks
};
```

**Example execution order:**
1. Check if curl exists (priority: 200)
2. Update package manager (priority: 100)
3. Install curl (priority: 50)
4. Install python3.11 (priority: 50)

### Task Dependencies

Tasks can depend on other tasks:

```typescript
taskRegistry.register({
  id: "custom:install-nvm",
  description: "Install NVM",
  executor: async () => { /* ... */ },
  dependsOn: [
    "linux:package-manager:apt-update",
    "linux:package:install:curl",
  ],
});
```

## System Task Helpers

Genesis provides pre-built helpers for common operations:

### Package Manager Update

```typescript
import { createPackageManagerUpdateTask } from "@ossl/genesis-core";

const task = createPackageManagerUpdateTask(cwd, env);
// Creates: linux:package-manager:apt-update (on Linux)
// Creates: macos:package-manager:brew-update (on macOS)
```

### Package Installation

```typescript
import { createPackageInstallTask } from "@ossl/genesis-core";

const task = createPackageInstallTask("curl", cwd, env);
// Creates: linux:package:install:curl (on Linux)
// Creates: macos:package:install:curl (on macOS)
```

### Command Check

```typescript
import { createCommandCheckTask } from "@ossl/genesis-core";

const task = createCommandCheckTask("curl", cwd, env);
// Creates: system:command:check:curl
```

### Custom Task

```typescript
import { createCustomTask } from "@ossl/genesis-core";

const task = createCustomTask(
  "custom:my-task",
  "My custom task",
  async () => {
    // Your custom logic
  },
  {
    priority: 100,
    dependsOn: ["linux:package-manager:apt-update"],
  }
);
```

## Real-World Example

### Configuration

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

### Execution Flow

```
Phase 1: Registering tasks...
  ℹ node: Registered system tasks
    - linux:package-manager:apt-update
    - linux:package:install:curl
  ℹ python: Registered system tasks
    - linux:package-manager:apt-update (duplicate!)
    - linux:package:install:python3.11

Phase 2: Executing system tasks...
  ✓ linux:package-manager:apt-update (deduplicated - runs once!)
  ✓ linux:package:install:curl
  ✓ linux:package:install:python3.11

Phase 3: Installing plugins...
  ℹ node: Installing NVM...
  ✓ node: NVM installed successfully
  ✓ node: Node.js 20 installed
  ✓ python: Python 3.11 verified

✅ Environment setup complete!
```

### Time Savings

**Without task registry:**
- apt-get update: 30s × 2 = 60s
- Install curl: 5s
- Install python3.11: 10s
- **Total: 75s**

**With task registry:**
- apt-get update: 30s × 1 = 30s
- Install curl: 5s
- Install python3.11: 10s
- **Total: 45s**

**Savings: 40% faster!** (And scales with more plugins)

## Platform Support

Task registry works across all platforms:

### Linux (Debian/Ubuntu)
- Package manager: `apt-get`
- Update task: `linux:package-manager:apt-update`
- Install task: `linux:package:install:<package>`

### Linux (RedHat/Fedora)
- Package manager: `yum` or `dnf`
- Update task: `linux:package-manager:yum-update`
- Install task: `linux:package:install:<package>`

### macOS
- Package manager: `brew`
- Update task: `macos:package-manager:brew-update`
- Install task: `macos:package:install:<package>`

### Windows
- Currently: Manual installation guides
- Future: Chocolatey support

## Best Practices

### 1. Use System Tasks for System Operations

```typescript
// ✅ Good: Use task registry for system packages
async registerTasks(runtime) {
  taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}

// ❌ Bad: Install system packages in apply()
async apply(runtime) {
  await runCommand("sudo", ["apt-get", "install", "curl"]);
}
```

### 2. Use apply() for User-Space Operations

```typescript
// ✅ Good: User-space installations in apply()
async apply(runtime) {
  // Install NVM (user-space)
  await installNvm();
}
```

### 3. Set Appropriate Priorities

```typescript
// ✅ Good: Higher priority for checks
createCommandCheckTask("curl", cwd, env);  // Priority: 200

// ✅ Good: Medium priority for updates
createPackageManagerUpdateTask(cwd, env);  // Priority: 100

// ✅ Good: Lower priority for installs
createPackageInstallTask("curl", cwd, env);  // Priority: 50
```

## What's Next?

- [Plugin Development](/guide/plugin-development) - Create plugins using task registry
- [Architecture](/guide/architecture) - Deep dive into Genesis architecture
- [API Reference](/api/task-registry) - Complete task registry API

