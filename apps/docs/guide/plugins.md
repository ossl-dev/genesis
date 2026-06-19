# Plugins

Learn about Genesis's plugin system and how to use plugins in your configuration.

## What are Plugins?

Plugins are modular components that handle the installation and configuration of specific tools, languages, or SDKs. Each plugin encapsulates all the platform-specific logic needed to install and manage a particular piece of software.

## Plugin Categories

Genesis organizes plugins into three categories:

### Tools

Development tools and utilities:
- Node.js (with NVM support)
- More coming soon...

### Languages

Programming language runtimes:
- Python
- More coming soon...

### SDKs

Software development kits:
- Coming soon...

## Using Plugins

### Import Plugins

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, python } from "@ossl/genesis-plugins";
```

### Configure Plugins

```typescript
export default defineConfig({
  tools: [
    node({
      version: "20",
      use_nvm: true,
    }),
  ],
  languages: [
    python({
      version: "3.11",
    }),
  ],
});
```

## Plugin Lifecycle

Each plugin goes through four phases:

### 1. Detection
Checks if the tool is already installed and at the correct version.

### 2. Task Registration
Registers system-level prerequisites (package manager updates, system packages).

### 3. Installation
Performs the actual installation of the tool.

### 4. Validation
Verifies the installation was successful.

## Available Plugins

See the [Plugin Overview](/plugins/overview) for a complete list of available plugins.

## Creating Custom Plugins

Want to create your own plugin? Check out the [Plugin Development Guide](/guide/plugin-development).

## What's Next?

- [Plugin Overview](/plugins/overview) - See all available plugins
- [Task Registry](/guide/task-registry) - Learn about task deduplication
- [Plugin Development](/guide/plugin-development) - Create custom plugins

