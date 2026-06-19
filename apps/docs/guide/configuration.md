# Configuration

Learn how to configure Genesis for your development environment.

## Configuration File

Genesis supports two configuration formats:

### TypeScript Configuration (Recommended)

Create `genesis.config.ts` in your project root:

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [],
  languages: [],
  sdks: [],
});
```

**Advantages:**
- ✅ Full TypeScript type safety
- ✅ IntelliSense and autocomplete
- ✅ Programmatic configuration
- ✅ Import and reuse configs

### YAML Configuration

Create `genesis.yaml` in your project root:

```yaml
tools: []
languages: []
sdks: []
```

**Advantages:**
- ✅ Simple and readable
- ✅ No build step required
- ✅ Easy for non-developers

## Configuration Structure

### Top-Level Fields

```typescript
interface GenesisConfig {
  tools?: PluginInstance[];      // Development tools (Node.js, etc.)
  languages?: PluginInstance[];  // Programming languages (Python, etc.)
  sdks?: PluginInstance[];       // SDKs and frameworks
}
```

### Plugin Instance

Each plugin is configured with its specific options:

```typescript
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({
      version: "20",      // Required
      use_nvm: true,      // Optional
    }),
  ],
  languages: [
    python({
      version: "3.11",    // Required
    }),
  ],
});
```

## Plugin Categories

### Tools

Development tools and utilities:

```typescript
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

### Languages

Programming language runtimes:

```typescript
import { python } from "@ossl/genesis-plugins";

export default defineConfig({
  languages: [
    python({
      version: "3.11",
    }),
  ],
});
```

### SDKs

Software development kits (coming soon):

```typescript
export default defineConfig({
  sdks: [
    // Coming soon: AWS SDK, Google Cloud SDK, etc.
  ],
});
```

## Complete Example

Here's a comprehensive configuration:

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  // Development tools
  tools: [
    // Node.js with NVM
    node({
      version: "20",
      use_nvm: true,
    }),
  ],
  
  // Programming languages
  languages: [
    // Python 3.11
    python({
      version: "3.11",
    }),
  ],
});
```

## Multiple Versions

Install multiple versions of the same tool:

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
    node({ version: "18", use_nvm: true }),
  ],
  languages: [
    python({ version: "3.11" }),
    python({ version: "3.10" }),
  ],
});
```

## Environment-Specific Configuration

### Using Environment Variables

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

const nodeVersion = process.env.NODE_VERSION || "20";

export default defineConfig({
  tools: [
    node({
      version: nodeVersion,
      use_nvm: true,
    }),
  ],
});
```

### Conditional Configuration

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

const isDevelopment = process.env.NODE_ENV === "development";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
  languages: [
    // Only install Python in development
    ...(isDevelopment ? [python({ version: "3.11" })] : []),
  ],
});
```

## Shared Configurations

### Importing Configs

Create reusable configuration modules:

```typescript
// configs/base.ts
import { node } from "@ossl/genesis-plugins";

export const baseTools = [
  node({ version: "20", use_nvm: true }),
];
```

```typescript
// genesis.config.ts
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";
import { baseTools } from "./configs/base";

export default defineConfig({
  tools: baseTools,
  languages: [
    python({ version: "3.11" }),
  ],
});
```

### Extending Configs

```typescript
// configs/base.config.ts
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
});
```

```typescript
// genesis.config.ts
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";
import baseConfig from "./configs/base.config";

export default defineConfig({
  ...baseConfig,
  languages: [
    python({ version: "3.11" }),
  ],
});
```

## Configuration Validation

Genesis automatically validates your configuration:

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({
      version: "20",
      // TypeScript will error on invalid options
      invalid_option: true,  // ❌ Error!
    }),
  ],
});
```

## CLI Options

Override configuration via CLI:

```bash
# Use specific config file
genesis apply -c custom-config.ts

# Use YAML config
genesis apply -c genesis.yaml

# Dry run (show what would be installed)
genesis apply --dry-run

# Verbose output
genesis apply --verbose

# Specify working directory
genesis apply --cwd /path/to/project
```

## Best Practices

### 1. Use TypeScript Configuration

TypeScript provides type safety and better developer experience:

```typescript
// ✅ Good: Type-safe
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [node({ version: "20", use_nvm: true })],
});
```

### 2. Version Control Your Config

Commit your configuration file:

```bash
git add genesis.config.ts
git commit -m "Add Genesis configuration"
```

### 3. Document Plugin Options

Add comments to explain choices:

```typescript
export default defineConfig({
  tools: [
    // Use NVM for easy version switching between projects
    node({
      version: "20",  // LTS version for stability
      use_nvm: true,
    }),
  ],
});
```

### 4. Keep It Simple

Start with minimal configuration and add as needed:

```typescript
// ✅ Good: Start simple
export default defineConfig({
  tools: [node({ version: "20", use_nvm: true })],
});

// ❌ Avoid: Over-configuring upfront
export default defineConfig({
  tools: [/* 20 different tools */],
});
```

## What's Next?

- [Plugin Overview](/plugins/overview) - Learn about available plugins
- [Task Registry](/guide/task-registry) - Understand task deduplication
- [Plugin Development](/guide/plugin-development) - Create custom plugins

