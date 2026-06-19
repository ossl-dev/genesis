# Task Deduplication

::: tip
This is an alias for the [Task Registry](/guide/task-registry) guide. Both pages contain the same information.
:::

Task deduplication is one of Genesis's most powerful features. It eliminates redundant system operations when installing multiple plugins.

[Read the full Task Registry guide →](/guide/task-registry)

## Quick Overview

### The Problem

```bash
# Installing Node.js
sudo apt-get update  # ← Run 1
sudo apt-get install curl

# Installing Python
sudo apt-get update  # ← Run 2 (redundant!)
sudo apt-get install python3.11
```

### The Solution

```bash
# With Genesis task registry
sudo apt-get update  # ← Runs ONCE!
sudo apt-get install curl
sudo apt-get install python3.11
```

## How It Works

Genesis uses a three-phase execution model:

1. **Phase 1**: Plugins register system tasks
2. **Phase 2**: Tasks execute (deduplicated by ID)
3. **Phase 3**: Plugins complete installation

[Learn more about the Task Registry →](/guide/task-registry)

## Benefits

- ⚡ **50% faster** - Package manager updates run once
- 🎯 **Guaranteed dependencies** - System packages available when needed
- 🧹 **Clean execution** - No redundant operations

## Example

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

**Result:** Package manager update runs once, not twice!

## What's Next?

- [Task Registry](/guide/task-registry) - Full documentation
- [Plugin Development](/guide/plugin-development) - Use task registry in plugins
- [Architecture](/guide/architecture) - Deep dive into Genesis architecture

