# Plugin Development

Learn how to create custom Genesis plugins.

## Overview

Creating a Genesis plugin allows you to automate the installation and configuration of any tool, language, or SDK. Plugins use the three-phase execution model and task registry for optimal performance.

## Plugin Structure

A Genesis plugin consists of:

1. **Plugin factory function** - Creates plugin instances
2. **Plugin implementation** - Implements the plugin lifecycle
3. **Type definitions** - TypeScript interfaces for options

## Basic Plugin Template

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

// 1. Define options interface
export interface MyToolOptions {
  version: string;
}

// 2. Create plugin factory function
export function myTool(options: MyToolOptions): GenesisPluginInstance<MyToolOptions> {
  return {
    id: "my-tool",
    category: "tool",
    module: "@ossl/genesis-plugins/my-tool",
    options,
  };
}

// 3. Create plugin implementation
export function createPlugin(
  instance: GenesisPluginInstance<MyToolOptions>
): GenesisPlugin<MyToolOptions> {
  return {
    id: instance.id,
    category: instance.category,
    
    // Phase 0: Detection
    async detect(runtime: PluginRuntime<MyToolOptions>): Promise<DetectResult> {
      const { options, context } = runtime;
      const { logger } = context;
      
      logger.debug("Detecting my-tool...");
      
      try {
        const result = await runCommand("my-tool", ["--version"], {
          cwd: context.cwd,
          env: context.env,
        });
        
        if (result.code === 0) {
          return {
            ok: true,
            details: `my-tool ${result.stdout.trim()} is installed`,
          };
        }
      } catch (error) {
        logger.debug("my-tool not found");
      }
      
      return {
        ok: false,
        details: "my-tool is not installed",
      };
    },
    
    // Phase 1: Register system tasks
    async registerTasks(runtime: PluginRuntime<MyToolOptions>): Promise<void> {
      const { taskRegistry, logger } = runtime.context;
      const platform = getPlatform();
      
      if (platform === "windows") {
        return; // Skip Windows
      }
      
      logger.debug("Registering system tasks for my-tool");
      
      // Register package manager update (deduplicated!)
      taskRegistry.register(
        createPackageManagerUpdateTask(
          runtime.context.cwd,
          runtime.context.env
        )
      );
      
      // Register system dependencies
      taskRegistry.register(
        createPackageInstallTask(
          "build-essential",
          runtime.context.cwd,
          runtime.context.env
        )
      );
    },
    
    // Phase 3: Plugin installation
    async apply(runtime: PluginRuntime<MyToolOptions>): Promise<ApplyResult> {
      const { options, context } = runtime;
      const { logger } = context;
      const platform = getPlatform();
      
      logger.info(`Installing my-tool ${options.version}...`);
      
      // System packages are now available from Phase 2
      // Perform plugin-specific installation here
      
      return {
        ok: true,
        details: `my-tool ${options.version} installed successfully`,
      };
    },
    
    // Phase 4: Validation
    async validate(runtime: PluginRuntime<MyToolOptions>): Promise<ValidateResult> {
      return this.detect!(runtime);
    },
  };
}

export const myToolPlugin = createPlugin;
```

## Plugin Lifecycle

### 1. Detection (`detect`)

Check if the tool is already installed:

```typescript
async detect(runtime): Promise<DetectResult> {
  try {
    const result = await runCommand("my-tool", ["--version"], {
      cwd: context.cwd,
      env: context.env,
    });
    
    return {
      ok: result.code === 0,
      details: `my-tool ${result.stdout.trim()} is installed`,
    };
  } catch (error) {
    return {
      ok: false,
      details: "my-tool is not installed",
    };
  }
}
```

### 2. Task Registration (`registerTasks`)

Register system-level prerequisites:

```typescript
async registerTasks(runtime): Promise<void> {
  const { taskRegistry } = runtime.context;
  
  // Register package manager update (deduplicated!)
  taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)
  );
  
  // Register system packages
  taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}
```

### 3. Installation (`apply`)

Perform plugin-specific installation:

```typescript
async apply(runtime): Promise<ApplyResult> {
  const { options, context } = runtime;
  const { logger } = context;
  
  logger.info(`Installing my-tool ${options.version}...`);
  
  // System dependencies are now available
  // Perform installation
  
  return {
    ok: true,
    details: `my-tool ${options.version} installed`,
  };
}
```

### 4. Validation (`validate`)

Verify the installation:

```typescript
async validate(runtime): Promise<ValidateResult> {
  // Usually just reuse detect logic
  return this.detect!(runtime);
}
```

## Best Practices

### 1. Use Task Registry for System Operations

```typescript
// ✅ Good: Use task registry
async registerTasks(runtime) {
  taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}

// ❌ Bad: Install in apply()
async apply(runtime) {
  await runCommand("sudo", ["apt-get", "install", "curl"]);
}
```

### 2. Handle All Platforms

```typescript
async apply(runtime) {
  const platform = getPlatform();
  
  if (platform === "windows") {
    // Windows-specific logic
  } else if (platform === "macos") {
    // macOS-specific logic
  } else {
    // Linux-specific logic
  }
}
```

### 3. Provide Detailed Logging

```typescript
logger.debug("Checking if my-tool is installed...");
logger.info("Installing my-tool...");
logger.warn("my-tool version mismatch");
logger.error("Failed to install my-tool");
```

### 4. Make It Idempotent

```typescript
async apply(runtime) {
  // Check if already installed
  const detectResult = await this.detect!(runtime);
  if (detectResult.ok) {
    logger.info("my-tool already installed, skipping");
    return { ok: true, details: detectResult.details };
  }
  
  // Install only if needed
  // ...
}
```

## Testing Your Plugin

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { myTool } from "./my-plugin";

export default defineConfig({
  tools: [
    myTool({
      version: "1.0.0",
    }),
  ],
});
```

## Publishing Your Plugin

1. Create a package in `packages/plugins/src/plugins/my-tool/`
2. Export from `packages/plugins/src/index.ts`
3. Add to package.json exports
4. Update build script
5. Write documentation
6. Submit PR

## What's Next?

- [Task Registry](/guide/task-registry) - Learn about task deduplication
- [Plugin Overview](/plugins/overview) - See existing plugins
- [API Reference](/api/plugin) - Complete plugin API

