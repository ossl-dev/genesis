# Utilities

These are the current exported utilities from `@ossl/genesis-core`.

## Shell and platform

```typescript
getPlatform(): "macos" | "windows" | "linux"
runCommand(command: string, args: string[], options?: { cwd?: string; env?: NodeJS.ProcessEnv }): Promise<{ code: number; stdout: string; stderr: string }>
```

`runCommand` invokes an executable with an argument array through execa. It does not interpret shell syntax or split a command string. Nonzero exits return results rather than rejecting; check `code`. A missing exit code is treated as failure. Exceptions can still reject.

```typescript
const result = await runCommand("git", ["--version"], { cwd, env });
if (result.code !== 0) throw new Error(result.stderr || "Git unavailable");
```

For intentional shell scripts, invoke the shell explicitly. Avoid interpolating user values into shell source; use executable arguments or shell positional parameters.

There is no public `getDistro` or package-manager discovery API. Shared package tasks select brew/APT/Chocolatey by platform.

## Filesystem paths

```typescript
ensureDir(dir: string): Promise<void>
getGenesisHome(): string
ensureGenesisHome(): Promise<string>
joinPath(...segments: string[]): string
```

Genesis home is `~/.genesis`. Use Node filesystem APIs for reads/writes. Previously documented download, environment-variable, and shell-profile helpers are not exported.

## Logger

```typescript
const logger = new Logger({ level: "info", useColors: false, prefix: "genesis" });
logger.debug("details");
logger.info("progress");
logger.warn("warning");
logger.error("failure");
```

Levels are `debug`, `info`, `warn`, and `error`. Color defaults to TTY support and respects a nonempty `NO_COLOR`. Logging does not control process exit status.

`runCommand` also accepts an optional `timeout` in milliseconds. A timed-out child returns a nonzero code and an explanatory error when stderr is empty. Docker daemon checks use a 15-second timeout.
