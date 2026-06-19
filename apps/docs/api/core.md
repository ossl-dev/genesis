# Core API Reference

Complete API reference for `@ossl/genesis-core`.

## Configuration

### `defineConfig()`

Define a Genesis configuration.

```typescript
function defineConfig(config: GenesisConfig): GenesisConfig
```

**Parameters:**
- `config`: Configuration object

**Returns:** The same configuration object (for type safety)

**Example:**

```typescript
import { defineConfig } from "@ossl/genesis-core";

export default defineConfig({
  tools: [],
  languages: [],
  sdks: [],
});
```

### `GenesisConfig`

Configuration interface.

```typescript
interface GenesisConfig {
  tools?: GenesisPluginInstance[];
  languages?: GenesisPluginInstance[];
  sdks?: GenesisPluginInstance[];
}
```

## Plugin System

### `GenesisPlugin<TOptions>`

Plugin interface.

```typescript
interface GenesisPlugin<TOptions = any> {
  id: string;
  category: "tool" | "sdk" | "language";
  dependsOn?: string[];
  
  detect?(runtime: PluginRuntime<TOptions>): Promise<DetectResult>;
  registerTasks?(runtime: PluginRuntime<TOptions>): Promise<void>;
  apply?(runtime: PluginRuntime<TOptions>): Promise<ApplyResult>;
  validate?(runtime: PluginRuntime<TOptions>): Promise<ValidateResult>;
}
```

### `GenesisPluginInstance<TOptions>`

Plugin instance (returned by plugin factory functions).

```typescript
interface GenesisPluginInstance<TOptions = any> {
  id: string;
  category: "tool" | "sdk" | "language";
  module: string;
  options: TOptions;
}
```

### `PluginRuntime<TOptions>`

Runtime context passed to plugin methods.

```typescript
interface PluginRuntime<TOptions> {
  instance: GenesisPluginInstance<TOptions>;
  options: TOptions;
  context: GenesisPluginContext;
}
```

### `GenesisPluginContext`

Plugin execution context.

```typescript
interface GenesisPluginContext {
  cwd: string;
  env: NodeJS.ProcessEnv;
  logger: Logger;
  taskRegistry: TaskRegistry;
}
```

### Result Types

```typescript
interface DetectResult {
  ok: boolean;
  details: string;
}

interface ApplyResult {
  ok: boolean;
  details: string;
}

interface ValidateResult {
  ok: boolean;
  details: string;
}
```

## Task Registry

See [Task Registry API](/api/task-registry) for complete documentation.

### `TaskRegistry`

Task registry class.

```typescript
class TaskRegistry {
  register(task: Task): void;
  executeAll(): Promise<Map<TaskId, TaskResult>>;
}
```

### `Task`

Task interface.

```typescript
interface Task {
  id: TaskId;
  priority: number;
  dependsOn?: TaskId[];
  execute: () => Promise<void>;
}
```

### Helper Functions

```typescript
function createPackageManagerUpdateTask(
  cwd: string,
  env: NodeJS.ProcessEnv
): Task;

function createPackageInstallTask(
  packageName: string,
  cwd: string,
  env: NodeJS.ProcessEnv
): Task;
```

## Platform Utilities

### `getPlatform()`

Get current platform.

```typescript
function getPlatform(): "macos" | "linux" | "windows"
```

**Example:**

```typescript
import { getPlatform } from "@ossl/genesis-core";

const platform = getPlatform();
if (platform === "macos") {
  // macOS-specific logic
}
```

### `getPackageManager()`

Get package manager for current platform.

```typescript
function getPackageManager(): "brew" | "apt" | "yum" | "dnf" | null
```

## Command Execution

### `runCommand()`

Execute a shell command.

```typescript
function runCommand(
  command: string,
  args: string[],
  options: CommandOptions
): Promise<CommandResult>
```

**Parameters:**
- `command`: Command to execute
- `args`: Command arguments
- `options`: Execution options

**Returns:** Command result

**Example:**

```typescript
import { runCommand } from "@ossl/genesis-core";

const result = await runCommand("node", ["--version"], {
  cwd: "/path/to/dir",
  env: process.env,
});

if (result.code === 0) {
  console.log(result.stdout);
}
```

### `CommandOptions`

```typescript
interface CommandOptions {
  cwd: string;
  env: NodeJS.ProcessEnv;
  stdin?: string;
}
```

### `CommandResult`

```typescript
interface CommandResult {
  code: number;
  stdout: string;
  stderr: string;
}
```

## Logging

### `Logger`

Logger interface.

```typescript
interface Logger {
  debug(message: string): void;
  info(message: string): void;
  warn(message: string): void;
  error(message: string): void;
}
```

**Example:**

```typescript
logger.debug("Checking installation...");
logger.info("Installing my-tool...");
logger.warn("Version mismatch detected");
logger.error("Installation failed");
```

## File System

### `fileExists()`

Check if file exists.

```typescript
function fileExists(path: string): Promise<boolean>
```

### `readFile()`

Read file contents.

```typescript
function readFile(path: string): Promise<string>
```

### `writeFile()`

Write file contents.

```typescript
function writeFile(path: string, content: string): Promise<void>
```

## Environment

### `getEnv()`

Get environment variable.

```typescript
function getEnv(key: string): string | undefined
```

### `setEnv()`

Set environment variable.

```typescript
function setEnv(key: string, value: string): void
```

## Type Exports

All types are exported from `@ossl/genesis-core`:

```typescript
export type {
  GenesisConfig,
  GenesisPlugin,
  GenesisPluginInstance,
  GenesisPluginContext,
  PluginRuntime,
  DetectResult,
  ApplyResult,
  ValidateResult,
  Task,
  TaskId,
  TaskResult,
  Logger,
  CommandOptions,
  CommandResult,
};
```

## What's Next?

- [Task Registry API](/api/task-registry) - Task registry details
- [Plugin API](/api/plugin) - Plugin development API
- [Utilities API](/api/utilities) - Utility functions

