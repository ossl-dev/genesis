# Creating a Plugin

Step-by-step guide to creating a Genesis plugin.

## Overview

This guide will walk you through creating a complete Genesis plugin from scratch.

## Prerequisites

- Genesis development environment set up
- Understanding of the [plugin lifecycle](/plugins/lifecycle)
- TypeScript knowledge

## Step 1: Create Plugin Directory

```bash
mkdir -p packages/plugins/src/plugins/my-tool
cd packages/plugins/src/plugins/my-tool
```

## Step 2: Define Options Interface

Create `index.ts`:

```typescript
export interface MyToolOptions {
  version: string;
  // Add more options as needed
}
```

## Step 3: Create Plugin Factory

```typescript
import type { GenesisPluginInstance } from "@ossl/genesis-core";

export function myTool(options: MyToolOptions): GenesisPluginInstance<MyToolOptions> {
  return {
    id: "my-tool",
    category: "tool",  // or "language" or "sdk"
    module: "@ossl/genesis-plugins/my-tool",
    options,
  };
}
```

## Step 4: Implement Plugin

```typescript
import type {
  GenesisPlugin,
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

export function createPlugin(
  instance: GenesisPluginInstance<MyToolOptions>
): GenesisPlugin<MyToolOptions> {
  return {
    id: instance.id,
    category: instance.category,
    
    async detect(runtime: PluginRuntime<MyToolOptions>): Promise<DetectResult> {
      const { context } = runtime;
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
    
    async registerTasks(runtime: PluginRuntime<MyToolOptions>): Promise<void> {
      const { taskRegistry, logger } = runtime.context;
      const platform = getPlatform();
      
      if (platform === "windows") {
        return; // Skip Windows for now
      }
      
      logger.debug("Registering system tasks for my-tool");
      
      // Register package manager update (deduplicated!)
      taskRegistry.register(
        createPackageManagerUpdateTask(
          runtime.context.cwd,
          runtime.context.env
        )
      );
      
      // Register any system dependencies
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
      const platform = getPlatform();
      
      logger.info(`Installing my-tool ${options.version}...`);
      
      // Check if already installed
      const detectResult = await this.detect!(runtime);
      if (detectResult.ok) {
        logger.info("my-tool already installed, skipping");
        return { ok: true, details: detectResult.details };
      }
      
      // Platform-specific installation
      if (platform === "windows") {
        logger.info("Windows installation guide:");
        logger.info("1. Visit https://my-tool.org/download");
        logger.info("2. Download installer");
        logger.info("3. Run installer");
        return {
          ok: false,
          details: "Manual installation required on Windows",
        };
      }
      
      // Install on macOS/Linux
      // System dependencies are now available from Phase 2
      
      // Your installation logic here
      // ...
      
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

## Step 5: Export Plugin

Update `packages/plugins/src/index.ts`:

```typescript
export { myTool, createPlugin as createMyToolPlugin } from "./plugins/my-tool/index.js";
export type { MyToolOptions } from "./plugins/my-tool/index.js";
```

## Step 6: Update package.json

Add to `packages/plugins/package.json`:

```json
{
  "exports": {
    ".": {
      "import": "./dist/index.js",
      "types": "./dist/index.d.ts"
    },
    "./my-tool": {
      "import": "./dist/plugins/my-tool/index.js",
      "types": "./dist/plugins/my-tool/index.d.ts"
    }
  }
}
```

## Step 7: Update Build Script

Update `packages/plugins/package.json`:

```json
{
  "scripts": {
    "build": "tsup src/index.ts src/plugins/node/index.ts src/plugins/python/index.ts src/plugins/my-tool/index.ts --dts --format esm --minify"
  }
}
```

## Step 8: Build and Test

```bash
# Build
bun run build

# Test
cd examples
cat > test-my-tool.ts << 'EOF'
import { defineConfig } from "@ossl/genesis-core";
import { myTool } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    myTool({
      version: "1.0.0",
    }),
  ],
});
EOF

bunx genesis apply -c test-my-tool.ts
```

## Step 9: Add Documentation

Create `packages/plugins/src/plugins/my-tool/README.md`:

```markdown
# My Tool Plugin

Install and manage My Tool.

## Usage

\`\`\`typescript
import { defineConfig } from "@ossl/genesis-core";
import { myTool } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    myTool({
      version: "1.0.0",
    }),
  ],
});
\`\`\`

## Options

- `version` (required): Version to install

## Platform Support

- macOS: ✅ Supported
- Linux: ✅ Supported
- Windows: ⚠️ Manual installation
```

## Step 10: Add to VitePress Docs

Create `apps/docs/plugins/my-tool.md` with full documentation.

## Complete Example

See the [Node.js plugin](https://github.com/Open-Vanguard/genesis/tree/main/packages/plugins/src/plugins/node) for a complete example.

## Best Practices

### 1. Handle All Platforms

```typescript
const platform = getPlatform();

if (platform === "windows") {
  // Windows logic
} else if (platform === "macos") {
  // macOS logic
} else {
  // Linux logic
}
```

### 2. Use Task Registry

```typescript
// ✅ Good
async registerTasks(runtime) {
  taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}

// ❌ Bad
async apply(runtime) {
  await runCommand("sudo", ["apt-get", "install", "curl"]);
}
```

### 3. Make It Idempotent

```typescript
async apply(runtime) {
  // Check if already installed
  const detectResult = await this.detect!(runtime);
  if (detectResult.ok) {
    return { ok: true, details: "Already installed" };
  }
  
  // Install...
}
```

### 4. Provide Detailed Logging

```typescript
logger.debug("Checking if my-tool is installed...");
logger.info("Installing my-tool...");
logger.warn("my-tool version mismatch");
logger.error("Failed to install my-tool");
```

### 5. Handle Errors Gracefully

```typescript
try {
  await installMyTool();
  return { ok: true, details: "Installed successfully" };
} catch (error) {
  logger.error(`Installation failed: ${error.message}`);
  return { ok: false, details: error.message };
}
```

## Testing Your Plugin

### Manual Testing

```bash
cd examples
bunx genesis apply -c test-config.ts
```

### Automated Testing

```typescript
// tests/my-tool.test.ts
import { describe, it, expect } from "bun:test";
import { myTool } from "../src/plugins/my-tool";

describe("myTool", () => {
  it("creates plugin instance", () => {
    const instance = myTool({ version: "1.0.0" });
    expect(instance.id).toBe("my-tool");
    expect(instance.options.version).toBe("1.0.0");
  });
});
```

## Publishing

### Internal Plugin

Add to `@ossl/genesis-plugins` package (requires PR).

### External Plugin

Publish as separate npm package:

```json
{
  "name": "@my-org/genesis-plugin-my-tool",
  "version": "1.0.0",
  "peerDependencies": {
    "@ossl/genesis-core": "^0.1.0"
  }
}
```

## What's Next?

- [Plugin Lifecycle](/plugins/lifecycle) - Understand the lifecycle
- [Best Practices](/plugins/best-practices) - Plugin development tips
- [Contributing](/guide/contributing) - Submit your plugin

