# Quick Start

Get up and running with Genesis in under 5 minutes.

## TL;DR

```bash
# Install
bun add -D @ossl/genesis-core @ossl/genesis-plugins @ossl/genesis-cli

# Create config
cat > genesis.config.ts << 'EOF'
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [node({ version: "20", use_nvm: true })],
  languages: [python({ version: "3.11" })],
});
EOF

# Run
bunx genesis apply
```

## Step-by-Step

### 1. Install Genesis (30 seconds)

```bash
bun add -D @ossl/genesis-core @ossl/genesis-plugins @ossl/genesis-cli
```

### 2. Create Configuration (1 minute)

Create `genesis.config.ts`:

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

### 3. Run Genesis (2-5 minutes)

```bash
bunx genesis apply
```

That's it! Node.js 20 with NVM is now installed.

## Common Configurations

### Full-Stack JavaScript

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [
    node({ version: "20", use_nvm: true }),
  ],
});
```

### Python Development

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { python } from "@ossl/genesis-plugins";

export default defineConfig({
  languages: [
    python({ version: "3.11" }),
  ],
});
```

### Multi-Language Project

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

## YAML Configuration

Prefer YAML? Create `genesis.yaml`:

```yaml
tools:
  - type: node
    version: "20"
    use_nvm: true

languages:
  - type: python
    version: "3.11"
```

Then run:

```bash
bunx genesis apply -c genesis.yaml
```

## Verify Installation

Check what's installed:

```bash
# Detect current environment
bunx genesis detect

# Verify specific tools
node --version
python3 --version
```

## What's Next?

- [Full Getting Started Guide](/guide/getting-started) - Detailed installation and setup
- [Configuration Guide](/guide/configuration) - Learn all configuration options
- [Available Plugins](/plugins/overview) - Explore what you can install
- [Task Deduplication](/guide/task-deduplication) - Understand the optimization magic

## Need Help?

- Check the [Troubleshooting Guide](/guide/troubleshooting)
- Open an [issue on GitHub](https://github.com/Open-Vanguard/genesis/issues)
- Read the [full documentation](/guide/introduction)

