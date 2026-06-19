# Task Registry API

Complete API reference for the Task Registry system.

## Overview

The Task Registry provides task deduplication and execution management.

## TaskRegistry Class

### Constructor

```typescript
class TaskRegistry {
  constructor();
}
```

### Methods

#### `register(task: Task): void`

Register a task for execution.

**Parameters:**
- `task`: Task to register

**Behavior:**
- If task ID already exists, the task is ignored (deduplication)
- If task ID is new, the task is added to the registry

**Example:**

```typescript
taskRegistry.register({
  id: "linux:package-manager:apt-update",
  priority: 100,
  execute: async () => {
    await runCommand("sudo", ["apt-get", "update"], { cwd, env });
  },
});
```

#### `executeAll(): Promise<Map<TaskId, TaskResult>>`

Execute all registered tasks.

**Returns:** Map of task IDs to results

**Behavior:**
1. Deduplicates tasks by ID
2. Sorts by priority (higher first)
3. Resolves dependencies
4. Executes sequentially

**Example:**

```typescript
const results = await taskRegistry.executeAll();

for (const [taskId, result] of results) {
  if (result.ok) {
    console.log(`✓ ${taskId}`);
  } else {
    console.log(`✗ ${taskId}: ${result.error}`);
  }
}
```

## Task Interface

```typescript
interface Task {
  id: TaskId;
  priority: number;
  dependsOn?: TaskId[];
  execute: () => Promise<void>;
}
```

### Properties

#### `id: TaskId`

Unique task identifier.

**Format:** `"platform:category:operation[:parameter]"`

**Examples:**
- `"linux:package-manager:apt-update"`
- `"macos:package:install:curl"`
- `"linux:package:install:python3.11"`

#### `priority: number`

Task execution priority (higher = earlier).

**Standard Priorities:**
- `200` - Checks and validations
- `100` - Package manager updates
- `50` - Package installations
- `0` - Custom tasks

#### `dependsOn?: TaskId[]`

Optional array of task IDs this task depends on.

**Example:**

```typescript
{
  id: "linux:package:install:curl",
  priority: 50,
  dependsOn: ["linux:package-manager:apt-update"],
  execute: async () => { /* ... */ },
}
```

#### `execute: () => Promise<void>`

Async function that performs the task.

## TaskId Type

```typescript
type TaskId = string;
```

Task IDs follow the format: `"platform:category:operation[:parameter]"`

## TaskResult Interface

```typescript
interface TaskResult {
  ok: boolean;
  error?: string;
}
```

## Helper Functions

### `createPackageManagerUpdateTask()`

Create a package manager update task.

```typescript
function createPackageManagerUpdateTask(
  cwd: string,
  env: NodeJS.ProcessEnv
): Task
```

**Returns:** Task that updates the package manager

**Platform Behavior:**
- **macOS**: `brew update`
- **Linux (Debian/Ubuntu)**: `apt-get update`
- **Linux (RedHat/Fedora)**: `yum update` or `dnf update`
- **Windows**: No-op

**Example:**

```typescript
taskRegistry.register(
  createPackageManagerUpdateTask(cwd, env)
);
```

### `createPackageInstallTask()`

Create a package installation task.

```typescript
function createPackageInstallTask(
  packageName: string,
  cwd: string,
  env: NodeJS.ProcessEnv
): Task
```

**Parameters:**
- `packageName`: Name of package to install

**Returns:** Task that installs the package

**Platform Behavior:**
- **macOS**: `brew install <package>`
- **Linux (Debian/Ubuntu)**: `apt-get install <package>`
- **Linux (RedHat/Fedora)**: `yum install <package>` or `dnf install <package>`
- **Windows**: No-op

**Example:**

```typescript
taskRegistry.register(
  createPackageInstallTask("curl", cwd, env)
);
```

## Task Execution Flow

### 1. Registration Phase

```typescript
// Plugin A
taskRegistry.register(createPackageManagerUpdateTask(cwd, env));
taskRegistry.register(createPackageInstallTask("curl", cwd, env));

// Plugin B
taskRegistry.register(createPackageManagerUpdateTask(cwd, env));  // Deduplicated!
taskRegistry.register(createPackageInstallTask("python3.11", cwd, env));
```

### 2. Deduplication

```typescript
// Before deduplication: 4 tasks
// After deduplication: 3 tasks (apt-update appears once)
```

### 3. Priority Sorting

```typescript
// Tasks sorted by priority:
// 1. apt-update (priority: 100)
// 2. install:curl (priority: 50)
// 3. install:python3.11 (priority: 50)
```

### 4. Dependency Resolution

```typescript
// If install:curl depends on apt-update,
// apt-update executes first
```

### 5. Execution

```typescript
// Execute sequentially:
await task1.execute();
await task2.execute();
await task3.execute();
```

## Custom Tasks

### Creating Custom Tasks

```typescript
taskRegistry.register({
  id: "custom:setup:my-config",
  priority: 0,
  execute: async () => {
    // Your custom logic
    await writeFile("~/.myconfig", "content");
  },
});
```

### With Dependencies

```typescript
taskRegistry.register({
  id: "custom:install:my-tool",
  priority: 50,
  dependsOn: ["linux:package-manager:apt-update"],
  execute: async () => {
    // Runs after apt-update
    await installMyTool();
  },
});
```

## Best Practices

### 1. Use Standard Task IDs

```typescript
// ✅ Good: Standard format
"linux:package-manager:apt-update"
"macos:package:install:curl"

// ❌ Bad: Non-standard format
"update-apt"
"install_curl"
```

### 2. Set Appropriate Priorities

```typescript
// ✅ Good: Use standard priorities
createPackageManagerUpdateTask()  // priority: 100
createPackageInstallTask()        // priority: 50

// ❌ Bad: Random priorities
{ priority: 73 }
{ priority: 42 }
```

### 3. Declare Dependencies

```typescript
// ✅ Good: Explicit dependencies
{
  id: "linux:package:install:curl",
  dependsOn: ["linux:package-manager:apt-update"],
}

// ❌ Bad: Implicit dependencies
// Assuming apt-update runs first
```

### 4. Handle Errors

```typescript
// ✅ Good: Handle errors
execute: async () => {
  try {
    await runCommand("apt-get", ["update"], { cwd, env });
  } catch (error) {
    logger.error(`Failed to update: ${error.message}`);
    throw error;
  }
}
```

## Examples

### Basic Usage

```typescript
import { TaskRegistry, createPackageManagerUpdateTask } from "@ossl/genesis-core";

const taskRegistry = new TaskRegistry();

// Register tasks
taskRegistry.register(
  createPackageManagerUpdateTask(cwd, env)
);

// Execute all
const results = await taskRegistry.executeAll();
```

### Multiple Plugins

```typescript
// Plugin A
async registerTasks(runtime) {
  runtime.context.taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)
  );
  runtime.context.taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}

// Plugin B
async registerTasks(runtime) {
  runtime.context.taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)  // Deduplicated!
  );
  runtime.context.taskRegistry.register(
    createPackageInstallTask("git", cwd, env)
  );
}

// Result: apt-update runs once, curl and git install
```

## What's Next?

- [Core API](/api/core) - Core API reference
- [Plugin API](/api/plugin) - Plugin development API
- [Task Registry Guide](/guide/task-registry) - Detailed guide

