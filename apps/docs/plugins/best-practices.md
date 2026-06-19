# Plugin Best Practices

Best practices for developing Genesis plugins.

## Architecture

### Use Task Registry for System Operations

**✅ Good:**

```typescript
async registerTasks(runtime) {
  const { taskRegistry } = runtime.context;
  
  taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)
  );
  
  taskRegistry.register(
    createPackageInstallTask("curl", cwd, env)
  );
}
```

**❌ Bad:**

```typescript
async apply(runtime) {
  // Don't install system packages in apply()
  await runCommand("sudo", ["apt-get", "update"]);
  await runCommand("sudo", ["apt-get", "install", "curl"]);
}
```

**Why:** Task registry deduplicates operations across plugins.

### Separate Concerns

**✅ Good:**

```typescript
// Phase 1: Register system tasks
async registerTasks(runtime) {
  taskRegistry.register(createPackageInstallTask("curl", cwd, env));
}

// Phase 3: Plugin-specific installation
async apply(runtime) {
  await installNvm();  // Uses curl from Phase 2
}
```

**❌ Bad:**

```typescript
async apply(runtime) {
  // Don't mix system and plugin operations
  await installCurl();
  await installNvm();
}
```

## Platform Support

### Handle All Platforms

**✅ Good:**

```typescript
async apply(runtime) {
  const platform = getPlatform();
  
  if (platform === "windows") {
    return this.installWindows(runtime);
  } else if (platform === "macos") {
    return this.installMacOS(runtime);
  } else {
    return this.installLinux(runtime);
  }
}
```

**❌ Bad:**

```typescript
async apply(runtime) {
  // Assumes Linux only
  await runCommand("apt-get", ["install", "my-tool"]);
}
```

### Provide Windows Guidance

**✅ Good:**

```typescript
if (platform === "windows") {
  logger.info("Windows installation guide:");
  logger.info("1. Visit https://my-tool.org/download");
  logger.info("2. Download the installer");
  logger.info("3. Run the installer");
  logger.info("4. Restart your terminal");
  
  return {
    ok: false,
    details: "Manual installation required on Windows",
  };
}
```

**❌ Bad:**

```typescript
if (platform === "windows") {
  throw new Error("Windows not supported");
}
```

## Error Handling

### Graceful Degradation

**✅ Good:**

```typescript
try {
  await installMyTool();
  return { ok: true, details: "Installed successfully" };
} catch (error) {
  logger.error(`Installation failed: ${error.message}`);
  logger.info("You can install manually from https://my-tool.org");
  return { ok: false, details: error.message };
}
```

**❌ Bad:**

```typescript
// Let errors bubble up
await installMyTool();  // Crashes if fails
```

### Detailed Error Messages

**✅ Good:**

```typescript
throw new Error(
  `Failed to install Node.js ${version}: NVM installation failed. ` +
  `Please check your internet connection and try again. ` +
  `See https://github.com/nvm-sh/nvm for manual installation.`
);
```

**❌ Bad:**

```typescript
throw new Error("Installation failed");
```

## Idempotency

### Check Before Installing

**✅ Good:**

```typescript
async apply(runtime) {
  // Check if already installed
  const detectResult = await this.detect!(runtime);
  if (detectResult.ok) {
    logger.info("Already installed, skipping");
    return { ok: true, details: detectResult.details };
  }
  
  // Install only if needed
  await installMyTool();
}
```

**❌ Bad:**

```typescript
async apply(runtime) {
  // Always install, even if already present
  await installMyTool();
}
```

### Handle Partial Installations

**✅ Good:**

```typescript
async apply(runtime) {
  // Check each step
  if (!await isNvmInstalled()) {
    await installNvm();
  }
  
  if (!await isNodeInstalled(version)) {
    await installNode(version);
  }
}
```

## Logging

### Use Appropriate Log Levels

**✅ Good:**

```typescript
logger.debug("Checking if my-tool is installed...");
logger.info("Installing my-tool 1.0.0...");
logger.warn("my-tool version mismatch: found 0.9.0, expected 1.0.0");
logger.error("Failed to install my-tool: network error");
```

**❌ Bad:**

```typescript
// Everything at info level
logger.info("Checking if my-tool is installed...");
logger.info("my-tool not found");
logger.info("Installing my-tool...");
logger.info("Installation failed");
```

### Provide Progress Updates

**✅ Good:**

```typescript
logger.info("Downloading NVM installer...");
logger.info("Installing NVM...");
logger.info("Installing Node.js 20.10.0...");
logger.info("Configuring shell integration...");
logger.info("Installation complete!");
```

**❌ Bad:**

```typescript
logger.info("Installing...");
// Long silence
logger.info("Done");
```

## Type Safety

### Use TypeScript Strictly

**✅ Good:**

```typescript
export interface MyToolOptions {
  version: string;
  installPath?: string;
}

export function myTool(options: MyToolOptions): GenesisPluginInstance<MyToolOptions> {
  return {
    id: "my-tool",
    category: "tool",
    module: "@ossl/genesis-plugins/my-tool",
    options,
  };
}
```

**❌ Bad:**

```typescript
export function myTool(options: any) {
  return {
    id: "my-tool",
    options,
  };
}
```

### Validate Options

**✅ Good:**

```typescript
export function myTool(options: MyToolOptions): GenesisPluginInstance<MyToolOptions> {
  if (!options.version) {
    throw new Error("version is required");
  }
  
  if (!options.version.match(/^\d+\.\d+(\.\d+)?$/)) {
    throw new Error(`Invalid version format: ${options.version}`);
  }
  
  return {
    id: "my-tool",
    category: "tool",
    module: "@ossl/genesis-plugins/my-tool",
    options,
  };
}
```

## Performance

### Avoid Redundant Operations

**✅ Good:**

```typescript
// Use task registry - runs once
async registerTasks(runtime) {
  taskRegistry.register(
    createPackageManagerUpdateTask(cwd, env)
  );
}
```

**❌ Bad:**

```typescript
// Runs every time
async apply(runtime) {
  await runCommand("apt-get", ["update"]);
}
```

### Cache Detection Results

**✅ Good:**

```typescript
private detectCache?: DetectResult;

async detect(runtime) {
  if (this.detectCache) {
    return this.detectCache;
  }
  
  this.detectCache = await this.performDetection(runtime);
  return this.detectCache;
}
```

## Documentation

### Document All Options

**✅ Good:**

```typescript
/**
 * Options for the My Tool plugin.
 */
export interface MyToolOptions {
  /**
   * Version to install (e.g., "1.0.0")
   */
  version: string;
  
  /**
   * Custom installation path (optional)
   * @default "/usr/local/my-tool"
   */
  installPath?: string;
}
```

### Provide Examples

**✅ Good:**

```typescript
/**
 * Install My Tool.
 * 
 * @example
 * ```typescript
 * import { myTool } from "@ossl/genesis-plugins";
 * 
 * export default defineConfig({
 *   tools: [
 *     myTool({ version: "1.0.0" }),
 *   ],
 * });
 * ```
 */
export function myTool(options: MyToolOptions) {
  // ...
}
```

## Testing

### Test All Platforms

```typescript
describe("myTool", () => {
  it("works on macOS", async () => {
    // Mock getPlatform() to return "macos"
    // Test installation
  });
  
  it("works on Linux", async () => {
    // Mock getPlatform() to return "linux"
    // Test installation
  });
  
  it("provides guidance on Windows", async () => {
    // Mock getPlatform() to return "windows"
    // Test that guidance is provided
  });
});
```

### Test Idempotency

```typescript
it("is idempotent", async () => {
  // Install once
  await plugin.apply(runtime);
  
  // Install again - should skip
  const result = await plugin.apply(runtime);
  expect(result.ok).toBe(true);
  expect(result.details).toContain("already installed");
});
```

## Security

### Validate Downloads

**✅ Good:**

```typescript
async downloadInstaller(url: string) {
  const response = await fetch(url);
  const content = await response.text();
  
  // Verify checksum
  const checksum = await computeChecksum(content);
  if (checksum !== expectedChecksum) {
    throw new Error("Checksum mismatch - download may be corrupted");
  }
  
  return content;
}
```

### Avoid Arbitrary Code Execution

**✅ Good:**

```typescript
// Use runCommand with explicit arguments
await runCommand("curl", ["-fsSL", url], { cwd, env });
```

**❌ Bad:**

```typescript
// Don't use shell interpolation
await runCommand("sh", ["-c", `curl ${url}`]);  // Vulnerable to injection
```

## What's Next?

- [Creating a Plugin](/plugins/creating-plugin) - Build your plugin
- [Plugin Lifecycle](/plugins/lifecycle) - Understand the lifecycle
- [API Reference](/api/plugin) - Complete API documentation

