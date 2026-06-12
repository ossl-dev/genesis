# Troubleshooting

Common issues and solutions when using Genesis.

## Installation Issues

### Command not found: genesis

**Problem:** The `genesis` command is not available.

**Solution:**

```bash
# Make sure CLI is installed
bun add -D @genesis/cli

# Run via package.json scripts
bun run genesis:apply

# Or use bunx
bunx genesis apply
```

### Permission errors on Linux

**Problem:** Permission denied during installation.

**Solution:**

```bash
# Some operations require sudo
sudo bun run genesis:apply

# Or run specific commands with sudo
sudo apt-get update
sudo apt-get install <package>
```

### TypeScript errors

**Problem:** TypeScript compilation errors in config file.

**Solution:**

```bash
# Install TypeScript
bun add -D typescript

# Check your tsconfig.json
{
  "compilerOptions": {
    "module": "ESNext",
    "moduleResolution": "bundler",
    "target": "ES2020"
  }
}
```

## Plugin Issues

### Plugin not found

**Problem:** Plugin import fails.

**Solution:**

```typescript
// ✅ Correct import
import { node, python } from "@genesis/plugins";

// ❌ Wrong import
import { node } from "@genesis/core";  // Wrong package!
```

### Version mismatch

**Problem:** Installed version doesn't match requested version.

**Solution:**

```bash
# Check what's installed
genesis detect

# Uninstall and reinstall
# For Node.js with NVM
nvm uninstall <version>
genesis apply

# For Python
sudo apt-get remove python<version>
genesis apply
```

## Platform-Specific Issues

### macOS: Homebrew not found

**Problem:** Homebrew is not installed.

**Solution:**

```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Then run Genesis
genesis apply
```

### Linux: Package not found

**Problem:** Package not available in package manager.

**Solution:**

```bash
# Update package lists
sudo apt-get update

# Search for package
apt-cache search <package>

# Add additional repositories if needed
# For Python on Ubuntu
sudo add-apt-repository ppa:deadsnakes/ppa
sudo apt-get update
```

### Windows: Manual installation required

**Problem:** Automatic installation not supported on Windows.

**Solution:**

Follow the installation guides provided by Genesis. Each plugin will display detailed instructions for Windows installation.

## Task Registry Issues

### Tasks not deduplicating

**Problem:** Same task runs multiple times.

**Solution:**

This shouldn't happen! If you see this, please [open an issue](https://github.com/Open-Vanguard/genesis/issues) with:
- Your configuration file
- Genesis output logs
- Platform information

### Task execution fails

**Problem:** System task fails to execute.

**Solution:**

```bash
# Check logs for specific error
genesis apply --verbose

# Common fixes:
# 1. Update package manager manually
sudo apt-get update

# 2. Check internet connection
ping google.com

# 3. Check disk space
df -h
```

## Configuration Issues

### Invalid configuration

**Problem:** Configuration validation fails.

**Solution:**

```typescript
// Use TypeScript for type checking
import { defineConfig } from "@genesis/core";
import { node } from "@genesis/plugins";

export default defineConfig({
  tools: [
    node({
      version: "20",  // ✅ Valid
      // invalid_option: true,  // ❌ TypeScript error
    }),
  ],
});
```

### Config file not found

**Problem:** Genesis can't find configuration file.

**Solution:**

```bash
# Specify config file explicitly
genesis apply -c genesis.config.ts

# Or use default name
# genesis.config.ts or genesis.yaml in project root
```

## Network Issues

### Download fails

**Problem:** Failed to download installers or scripts.

**Solution:**

```bash
# Check internet connection
ping google.com

# Check firewall settings
# May need to allow outbound connections

# Use VPN if behind corporate firewall

# Retry (Genesis is idempotent)
genesis apply
```

### Slow downloads

**Problem:** Downloads are very slow.

**Solution:**

```bash
# Check your internet speed
# Some installers are large (Node.js, Python)

# Use a faster mirror if available
# For APT on Linux
sudo apt-get update --fix-missing
```

## Shell Integration Issues

### NVM not available after installation

**Problem:** `nvm` command not found after installation.

**Solution:**

```bash
# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc

# Or restart your terminal

# Verify NVM is installed
nvm --version
```

### Environment variables not set

**Problem:** PATH or other environment variables not updated.

**Solution:**

```bash
# Reload shell
source ~/.bashrc  # or ~/.zshrc

# Or restart terminal

# Check PATH
echo $PATH
```

## Debugging

### Enable verbose logging

```bash
genesis apply --verbose
```

### Check detection results

```bash
genesis detect
```

### Dry run

```bash
genesis apply --dry-run
```

### Check Genesis version

```bash
genesis --version
```

## Getting Help

### Check Documentation

- [Getting Started](/guide/getting-started)
- [Configuration](/guide/configuration)
- [Plugin Overview](/plugins/overview)

### Search Issues

Search existing issues: [GitHub Issues](https://github.com/Open-Vanguard/genesis/issues)

### Open an Issue

If you can't find a solution, [open a new issue](https://github.com/Open-Vanguard/genesis/issues/new) with:

1. **Genesis version**: `genesis --version`
2. **Platform**: macOS/Linux/Windows
3. **Configuration file**: Your `genesis.config.ts` or `genesis.yaml`
4. **Error message**: Full error output
5. **Steps to reproduce**: What you did before the error

### Contributing

Found a bug? Want to help? See the [Contributing Guide](/guide/contributing).

## Common Error Messages

### "Plugin not found"

```
Error: Plugin "my-plugin" not found
```

**Fix:** Check plugin name and import:

```typescript
import { node } from "@genesis/plugins";  // ✅ Correct
```

### "Version mismatch"

```
Warning: Node.js 18.0.0 found, but 20.0.0 required
```

**Fix:** Uninstall old version and run Genesis again.

### "Permission denied"

```
Error: EACCES: permission denied
```

**Fix:** Run with sudo (Linux) or check file permissions.

### "Command failed"

```
Error: Command failed: apt-get update
```

**Fix:** Check internet connection and package manager status.

## What's Next?

- [Getting Started](/guide/getting-started) - Start fresh
- [Configuration](/guide/configuration) - Review your config
- [GitHub Issues](https://github.com/Open-Vanguard/genesis/issues) - Get help

