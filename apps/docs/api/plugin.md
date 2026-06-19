# Plugin API Reference

Complete API reference for plugin development.

## Plugin Factory Function

### Signature

```typescript
function myPlugin<TOptions>(
  options: TOptions
): GenesisPluginInstance<TOptions>
```

### Example

```typescript
export interface MyPluginOptions {
  version: string;
}

export function myPlugin(
  options: MyPluginOptions
): GenesisPluginInstance<MyPluginOptions> {
  return {
    id: "my-plugin",
    category: "tool",
    module: "@ossl/genesis-plugins/my-plugin",
    options,
  };
}
```

## Plugin Implementation

### Signature

```typescript
function createPlugin<TOptions>(
  instance: GenesisPluginInstance<TOptions>
): GenesisPlugin<TOptions>
```

### Example

```typescript
export function createPlugin(
  instance: GenesisPluginInstance<MyPluginOptions>
): GenesisPlugin<MyPluginOptions> {
  return {
    id: instance.id,
    category: instance.category,
    
    async detect(runtime) { /* ... */ },
    async registerTasks(runtime) { /* ... */ },
    async apply(runtime) { /* ... */ },
    async validate(runtime) { /* ... */ },
  };
}
```

## Plugin Interface

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

### Properties

#### `id: string`

Unique plugin identifier.

**Example:** `"node"`, `"python"`, `"my-tool"`

#### `category: "tool" | "sdk" | "language"`

Plugin category.

**Categories:**
- `"tool"` - Development tools (Node.js, Git, etc.)
- `"language"` - Programming languages (Python, Ruby, etc.)
- `"sdk"` - Software development kits

#### `dependsOn?: string[]`

Optional array of plugin IDs this plugin depends on.

**Example:**

```typescript
{
  id: "my-framework",
  dependsOn: ["node"],  // Requires Node.js
}
```

### Methods

#### `detect?(runtime): Promise<DetectResult>`

Check if the tool is already installed.

**Parameters:**
- `runtime`: Plugin runtime context

**Returns:** Detection result

**Example:**

```typescript
async detect(runtime) {
  try {
    const result = await runCommand("my-tool", ["--version"], {
      cwd: runtime.context.cwd,
      env: runtime.context.env,
    });
    
    if (result.code === 0) {
      return {
        ok: true,
        details: `my-tool ${result.stdout.trim()} is installed`,
      };
    }
  } catch (error) {
    return {
      ok: false,
      details: "my-tool is not installed",
    };
  }
}
```

#### `registerTasks?(runtime): Promise<void>`

Register system-level prerequisites.

**Parameters:**
- `runtime`: Plugin runtime context

**Returns:** Nothing (void)

**Example:**

```typescript
async registerTasks(runtime) {
  const { taskRegistry } = runtime.context;
  
  taskRegistry.register(
    createPackageManagerUpdateTask(
      runtime.context.cwd,
      runtime.context.env
    )
  );
  
  taskRegistry.register(
    createPackageInstallTask(
      "curl",
      runtime.context.cwd,
      runtime.context.env
    )
  );
}
```

#### `apply?(runtime): Promise<ApplyResult>`

Perform plugin-specific installation.

**Parameters:**
- `runtime`: Plugin runtime context

**Returns:** Installation result

**Example:**

```typescript
async apply(runtime) {
  const { options, context } = runtime;
  const { logger } = context;
  
  logger.info(`Installing my-tool ${options.version}...`);
  
  // Check if already installed
  const detectResult = await this.detect!(runtime);
  if (detectResult.ok) {
    return { ok: true, details: detectResult.details };
  }
  
  // Install
  await installMyTool(options.version);
  
  return {
    ok: true,
    details: `my-tool ${options.version} installed successfully`,
  };
}
```

#### `validate?(runtime): Promise<ValidateResult>`

Verify the installation was successful.

**Parameters:**
- `runtime`: Plugin runtime context

**Returns:** Validation result

**Example:**

```typescript
async validate(runtime) {
  // Usually just reuse detect logic
  return this.detect!(runtime);
}
```

## Runtime Context

### `PluginRuntime<TOptions>`

```typescript
interface PluginRuntime<TOptions> {
  instance: GenesisPluginInstance<TOptions>;
  options: TOptions;
  context: GenesisPluginContext;
}
```

### `GenesisPluginContext`

```typescript
interface GenesisPluginContext {
  cwd: string;
  env: NodeJS.ProcessEnv;
  logger: Logger;
  taskRegistry: TaskRegistry;
}
```

#### Properties

##### `cwd: string`

Current working directory.

**Example:**

```typescript
await runCommand("npm", ["install"], {
  cwd: runtime.context.cwd,
  env: runtime.context.env,
});
```

##### `env: NodeJS.ProcessEnv`

Environment variables.

**Example:**

```typescript
const home = runtime.context.env.HOME;
```

##### `logger: Logger`

Logger instance.

**Example:**

```typescript
runtime.context.logger.info("Installing...");
runtime.context.logger.error("Failed!");
```

##### `taskRegistry: TaskRegistry`

Task registry instance.

**Example:**

```typescript
runtime.context.taskRegistry.register(task);
```

## Result Types

### `DetectResult`

```typescript
interface DetectResult {
  ok: boolean;
  details: string;
}
```

**Example:**

```typescript
return {
  ok: true,
  details: "Node.js 20.10.0 is installed",
};
```

### `ApplyResult`

```typescript
interface ApplyResult {
  ok: boolean;
  details: string;
}
```

**Example:**

```typescript
return {
  ok: true,
  details: "Node.js 20.10.0 installed successfully",
};
```

### `ValidateResult`

```typescript
interface ValidateResult {
  ok: boolean;
  details: string;
}
```

**Example:**

```typescript
return {
  ok: false,
  details: "Node.js installation failed",
};
```

## Complete Example

```typescript
import type {
  GenesisPlugin,
  GenesisPluginInstance,
  PluginRuntime,
  DetectResult,
  ApplyResult,
  ValidateResult,
} from "@ossl/genesis-core";
import {
  getPlatform,
  runCommand,
  createPackageManagerUpdateTask,
  createPackageInstallTask,
} from "@ossl/genesis-core";

export interface MyToolOptions {
  version: string;
}

export function myTool(
  options: MyToolOptions
): GenesisPluginInstance<MyToolOptions> {
  return {
    id: "my-tool",
    category: "tool",
    module: "@ossl/genesis-plugins/my-tool",
    options,
  };
}

export function createPlugin(
  instance: GenesisPluginInstance<MyToolOptions>
): GenesisPlugin<MyToolOptions> {
  return {
    id: instance.id,
    category: instance.category,
    
    async detect(runtime: PluginRuntime<MyToolOptions>): Promise<DetectResult> {
      try {
        const result = await runCommand("my-tool", ["--version"], {
          cwd: runtime.context.cwd,
          env: runtime.context.env,
        });
        
        if (result.code === 0) {
          return {
            ok: true,
            details: `my-tool ${result.stdout.trim()} is installed`,
          };
        }
      } catch (error) {
        return {
          ok: false,
          details: "my-tool is not installed",
        };
      }
    },
    
    async registerTasks(runtime: PluginRuntime<MyToolOptions>): Promise<void> {
      const { taskRegistry } = runtime.context;
      
      taskRegistry.register(
        createPackageManagerUpdateTask(
          runtime.context.cwd,
          runtime.context.env
        )
      );
      
      taskRegistry.register(
        createPackageInstallTask(
          "build-essential",
          runtime.context.cwd,
          runtime.context.env
        )
      );
    },
    
    async apply(runtime: PluginRuntime<MyToolOptions>): Promise<ApplyResult> {
      const { options, context } = runtime;
      const { logger } = context;
      
      logger.info(`Installing my-tool ${options.version}...`);
      
      const detectResult = await this.detect!(runtime);
      if (detectResult.ok) {
        return { ok: true, details: detectResult.details };
      }
      
      // Installation logic here
      
      return {
        ok: true,
        details: `my-tool ${options.version} installed successfully`,
      };
    },
    
    async validate(runtime: PluginRuntime<MyToolOptions>): Promise<ValidateResult> {
      return this.detect!(runtime);
    },
  };
}

export const myToolPlugin = createPlugin;
```

## What's Next?

- [Core API](/api/core) - Core API reference
- [Task Registry API](/api/task-registry) - Task registry details
- [Creating a Plugin](/plugins/creating-plugin) - Step-by-step guide

