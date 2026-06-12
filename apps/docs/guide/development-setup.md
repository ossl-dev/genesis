# Development Setup

Set up your local development environment for contributing to Genesis.

## Prerequisites

- **Node.js 18+**
- **Bun** (recommended) or npm
- **Git**
- **TypeScript** knowledge

## Clone the Repository

```bash
git clone https://github.com/Open-Vanguard/genesis.git
cd genesis
```

## Install Dependencies

```bash
bun install
```

## Project Structure

```
genesis/
├── apps/
│   ├── cli/              # CLI application
│   └── docs/             # VitePress documentation
├── packages/
│   ├── core/             # Core library
│   └── plugins/          # Plugin library
├── examples/             # Example configurations
├── docs/                 # Additional documentation
├── package.json          # Root package.json
├── turbo.json            # Turborepo configuration
└── README.md
```

## Build Packages

```bash
# Build all packages
bun run build

# Build specific package
bun run build --filter=@genesis/core
```

## Development Workflow

### 1. Make Changes

Edit files in `packages/core/` or `packages/plugins/`

### 2. Build

```bash
bun run build
```

### 3. Test Locally

Create a test configuration in `examples/`:

```typescript
// examples/test.config.ts
import { defineConfig } from "@genesis/core";
import { node } from "@genesis/plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
});
```

Run it:

```bash
cd examples
bunx genesis apply -c test.config.ts
```

### 4. Run Documentation

```bash
# Start docs dev server
bun run docs:dev

# Build docs
bun run docs:build
```

## Package Development

### Core Package (`@genesis/core`)

Located in `packages/core/`:

```
packages/core/
├── src/
│   ├── config/           # Configuration system
│   ├── plugins/          # Plugin system
│   ├── execution/        # Task registry
│   ├── os/               # Platform utilities
│   ├── fs/               # File system utilities
│   ├── env/              # Environment management
│   ├── utils/            # Utilities
│   └── index.ts          # Main export
├── package.json
├── tsconfig.json
└── tsup.config.ts
```

**Build:**

```bash
cd packages/core
bun run build
```

### Plugins Package (`@genesis/plugins`)

Located in `packages/plugins/`:

```
packages/plugins/
├── src/
│   ├── plugins/
│   │   ├── node/         # Node.js plugin
│   │   └── python/       # Python plugin
│   └── index.ts          # Main export
├── package.json
├── tsconfig.json
└── tsup.config.ts
```

**Build:**

```bash
cd packages/plugins
bun run build
```

### CLI Package (`@genesis/cli`)

Located in `apps/cli/`:

```
apps/cli/
├── src/
│   ├── commands/         # CLI commands
│   ├── lib/              # CLI utilities
│   └── index.ts          # Main entry
├── package.json
├── tsconfig.json
└── tsup.config.ts
```

**Build:**

```bash
cd apps/cli
bun run build
```

## Creating a New Plugin

### 1. Create Plugin Directory

```bash
mkdir -p packages/plugins/src/plugins/my-tool
```

### 2. Create Plugin Files

```typescript
// packages/plugins/src/plugins/my-tool/index.ts
import type {
  GenesisPlugin,
  GenesisPluginInstance,
  PluginRuntime,
} from "@genesis/core";

export interface MyToolOptions {
  version: string;
}

export function myTool(options: MyToolOptions): GenesisPluginInstance<MyToolOptions> {
  return {
    id: "my-tool",
    category: "tool",
    module: "@genesis/plugins/my-tool",
    options,
  };
}

export function createPlugin(
  instance: GenesisPluginInstance<MyToolOptions>
): GenesisPlugin<MyToolOptions> {
  return {
    id: instance.id,
    category: instance.category,
    
    async detect(runtime) {
      // Implementation
    },
    
    async registerTasks(runtime) {
      // Implementation
    },
    
    async apply(runtime) {
      // Implementation
    },
    
    async validate(runtime) {
      // Implementation
    },
  };
}

export const myToolPlugin = createPlugin;
```

### 3. Export Plugin

```typescript
// packages/plugins/src/index.ts
export { myTool, createMyToolPlugin } from "./plugins/my-tool/index.js";
export type { MyToolOptions } from "./plugins/my-tool/index.js";
```

### 4. Update package.json

```json
{
  "exports": {
    "./my-tool": {
      "import": "./dist/plugins/my-tool/index.js",
      "types": "./dist/plugins/my-tool/index.d.ts"
    }
  }
}
```

### 5. Update Build Script

```json
{
  "scripts": {
    "build": "tsup src/index.ts src/plugins/node/index.ts src/plugins/python/index.ts src/plugins/my-tool/index.ts --dts --format esm --minify"
  }
}
```

### 6. Build and Test

```bash
bun run build
cd examples
# Test your plugin
```

## Running Tests

```bash
# Run tests (if configured)
bun test
```

## Linting

```bash
# Lint code
bun run lint
```

## Documentation

### Local Development

```bash
# Start docs dev server
bun run docs:dev
```

Visit `http://localhost:5173` in your browser.

### Build Documentation

```bash
bun run docs:build
```

### Preview Built Docs

```bash
bun run docs:preview
```

## Debugging

### Enable Verbose Logging

```typescript
// In your code
logger.setLevel("debug");
```

### Use Node Debugger

```bash
node --inspect-brk ./apps/cli/dist/index.js apply
```

## Common Tasks

### Add a New Dependency

```bash
# To core package
cd packages/core
bun add <package>

# To plugins package
cd packages/plugins
bun add <package>
```

### Update All Dependencies

```bash
bun update
```

### Clean Build Artifacts

```bash
# Clean all packages
bun run clean

# Or manually
rm -rf packages/*/dist apps/*/dist
```

## Troubleshooting

### Build Fails

```bash
# Clean and rebuild
bun run clean
bun install
bun run build
```

### Type Errors

```bash
# Check TypeScript
cd packages/core
bunx tsc --noEmit
```

### Import Errors

Make sure you're using `.js` extensions in imports:

```typescript
// ✅ Correct
import { foo } from "./foo.js";

// ❌ Wrong
import { foo } from "./foo";
```

## What's Next?

- [Plugin Development](/guide/plugin-development) - Create plugins
- [Contributing Guide](/guide/contributing) - Contribution guidelines
- [Architecture](/guide/architecture) - Understand the architecture

