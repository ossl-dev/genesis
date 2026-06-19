# Utilities API Reference

Utility functions provided by `@ossl/genesis-core`.

## Platform Utilities

### `getPlatform()`

Get the current platform.

```typescript
function getPlatform(): "macos" | "linux" | "windows"
```

**Returns:** Platform identifier

**Example:**

```typescript
import { getPlatform } from "@ossl/genesis-core";

const platform = getPlatform();

if (platform === "macos") {
  // macOS-specific logic
} else if (platform === "linux") {
  // Linux-specific logic
} else {
  // Windows-specific logic
}
```

### `getPackageManager()`

Get the package manager for the current platform.

```typescript
function getPackageManager(): "brew" | "apt" | "yum" | "dnf" | null
```

**Returns:** Package manager identifier or null

**Platform Behavior:**
- **macOS**: `"brew"`
- **Debian/Ubuntu**: `"apt"`
- **RedHat/Fedora**: `"yum"` or `"dnf"`
- **Windows**: `null`

**Example:**

```typescript
import { getPackageManager } from "@ossl/genesis-core";

const pm = getPackageManager();

if (pm === "brew") {
  await runCommand("brew", ["install", "curl"], { cwd, env });
} else if (pm === "apt") {
  await runCommand("sudo", ["apt-get", "install", "curl"], { cwd, env });
}
```

### `getDistro()`

Get Linux distribution (Linux only).

```typescript
function getDistro(): "debian" | "ubuntu" | "fedora" | "redhat" | "centos" | "arch" | null
```

**Returns:** Distribution identifier or null

**Example:**

```typescript
import { getDistro } from "@ossl/genesis-core";

const distro = getDistro();

if (distro === "ubuntu") {
  // Ubuntu-specific logic
}
```

## Command Execution

### `runCommand()`

Execute a shell command.

```typescript
function runCommand(
  command: string,
  args: string[],
  options: CommandOptions
): Promise<CommandResult>
```

**Parameters:**
- `command`: Command to execute
- `args`: Array of arguments
- `options`: Execution options

**Returns:** Command result

**Example:**

```typescript
import { runCommand } from "@ossl/genesis-core";

const result = await runCommand("node", ["--version"], {
  cwd: "/path/to/dir",
  env: process.env,
});

if (result.code === 0) {
  console.log(`Node version: ${result.stdout.trim()}`);
} else {
  console.error(`Error: ${result.stderr}`);
}
```

### `CommandOptions`

```typescript
interface CommandOptions {
  cwd: string;
  env: NodeJS.ProcessEnv;
  stdin?: string;
}
```

**Properties:**
- `cwd`: Working directory
- `env`: Environment variables
- `stdin`: Optional input to pass to command

### `CommandResult`

```typescript
interface CommandResult {
  code: number;
  stdout: string;
  stderr: string;
}
```

**Properties:**
- `code`: Exit code (0 = success)
- `stdout`: Standard output
- `stderr`: Standard error

## File System

### `fileExists()`

Check if a file or directory exists.

```typescript
function fileExists(path: string): Promise<boolean>
```

**Example:**

```typescript
import { fileExists } from "@ossl/genesis-core";

if (await fileExists("~/.nvmrc")) {
  console.log(".nvmrc file exists");
}
```

### `readFile()`

Read file contents.

```typescript
function readFile(path: string): Promise<string>
```

**Example:**

```typescript
import { readFile } from "@ossl/genesis-core";

const content = await readFile("~/.bashrc");
console.log(content);
```

### `writeFile()`

Write file contents.

```typescript
function writeFile(path: string, content: string): Promise<void>
```

**Example:**

```typescript
import { writeFile } from "@ossl/genesis-core";

await writeFile("~/.myconfig", "key=value\n");
```

### `appendFile()`

Append to file.

```typescript
function appendFile(path: string, content: string): Promise<void>
```

**Example:**

```typescript
import { appendFile } from "@ossl/genesis-core";

await appendFile("~/.bashrc", "\nexport PATH=$PATH:/new/path\n");
```

### `createDirectory()`

Create a directory (recursive).

```typescript
function createDirectory(path: string): Promise<void>
```

**Example:**

```typescript
import { createDirectory } from "@ossl/genesis-core";

await createDirectory("~/.config/my-tool");
```

## Environment

### `getEnv()`

Get environment variable.

```typescript
function getEnv(key: string): string | undefined
```

**Example:**

```typescript
import { getEnv } from "@ossl/genesis-core";

const home = getEnv("HOME");
const path = getEnv("PATH");
```

### `setEnv()`

Set environment variable.

```typescript
function setEnv(key: string, value: string): void
```

**Example:**

```typescript
import { setEnv } from "@ossl/genesis-core";

setEnv("MY_VAR", "my-value");
```

### `expandPath()`

Expand path with environment variables.

```typescript
function expandPath(path: string): string
```

**Example:**

```typescript
import { expandPath } from "@ossl/genesis-core";

const expanded = expandPath("~/my-dir");
// Returns: "/Users/username/my-dir" (on macOS)
```

## Shell Integration

### `getShell()`

Get current shell.

```typescript
function getShell(): "bash" | "zsh" | "fish" | "powershell" | "cmd" | null
```

**Example:**

```typescript
import { getShell } from "@ossl/genesis-core";

const shell = getShell();

if (shell === "zsh") {
  await appendFile("~/.zshrc", "export MY_VAR=value\n");
} else if (shell === "bash") {
  await appendFile("~/.bashrc", "export MY_VAR=value\n");
}
```

### `getShellConfigFile()`

Get shell configuration file path.

```typescript
function getShellConfigFile(): string | null
```

**Returns:** Path to shell config file or null

**Example:**

```typescript
import { getShellConfigFile } from "@ossl/genesis-core";

const configFile = getShellConfigFile();
// Returns: "~/.zshrc" or "~/.bashrc" etc.

if (configFile) {
  await appendFile(configFile, "export PATH=$PATH:/new/path\n");
}
```

## Logging

### `createLogger()`

Create a logger instance.

```typescript
function createLogger(name: string): Logger
```

**Example:**

```typescript
import { createLogger } from "@ossl/genesis-core";

const logger = createLogger("my-plugin");

logger.debug("Debug message");
logger.info("Info message");
logger.warn("Warning message");
logger.error("Error message");
```

### `Logger` Interface

```typescript
interface Logger {
  debug(message: string): void;
  info(message: string): void;
  warn(message: string): void;
  error(message: string): void;
}
```

## Download Utilities

### `downloadFile()`

Download a file from URL.

```typescript
function downloadFile(url: string, destination: string): Promise<void>
```

**Example:**

```typescript
import { downloadFile } from "@ossl/genesis-core";

await downloadFile(
  "https://example.com/installer.sh",
  "/tmp/installer.sh"
);
```

### `downloadText()`

Download text content from URL.

```typescript
function downloadText(url: string): Promise<string>
```

**Example:**

```typescript
import { downloadText } from "@ossl/genesis-core";

const script = await downloadText("https://example.com/install.sh");
console.log(script);
```

## String Utilities

### `trimLines()`

Trim whitespace from each line.

```typescript
function trimLines(text: string): string
```

**Example:**

```typescript
import { trimLines } from "@ossl/genesis-core";

const text = `
  line 1
  line 2
`;

const trimmed = trimLines(text);
// Returns: "line 1\nline 2"
```

### `indent()`

Indent text by specified spaces.

```typescript
function indent(text: string, spaces: number): string
```

**Example:**

```typescript
import { indent } from "@ossl/genesis-core";

const text = "line 1\nline 2";
const indented = indent(text, 2);
// Returns: "  line 1\n  line 2"
```

## Validation

### `isValidVersion()`

Check if string is a valid version.

```typescript
function isValidVersion(version: string): boolean
```

**Example:**

```typescript
import { isValidVersion } from "@ossl/genesis-core";

isValidVersion("1.0.0");     // true
isValidVersion("20");        // true
isValidVersion("invalid");   // false
```

### `isValidUrl()`

Check if string is a valid URL.

```typescript
function isValidUrl(url: string): boolean
```

**Example:**

```typescript
import { isValidUrl } from "@ossl/genesis-core";

isValidUrl("https://example.com");  // true
isValidUrl("not a url");            // false
```

## What's Next?

- [Core API](/api/core) - Core API reference
- [Plugin API](/api/plugin) - Plugin development API
- [Task Registry API](/api/task-registry) - Task registry details

