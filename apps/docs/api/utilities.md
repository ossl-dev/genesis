# Utilities

Import `github.com/ossl-dev/genesis/packages/core`.

## Commands and platforms

```go
func GetPlatform() string
func RunCommand(ctx context.Context, command string, args []string, options CommandOptions) (CommandResult, error)
func GetLinuxDistribution() (LinuxDistribution, error)
func ParseLinuxDistribution(data []byte) (LinuxDistribution, error)
func PackageManager(platform string, distro LinuxDistribution) (string, error)
```

Supported platform names are `macos`, `linux`, and `windows`; other hosts retain their Go OS name and package-task selection rejects them. Runtime architecture uses Go names such as `amd64` and `arm64`.

`LinuxDistribution` contains `ID`, `Like`, and `Codename`. Parsing does not execute `os-release` values. Detection reads `/etc/os-release`, falling back to `/usr/lib/os-release` only when the first file is missing.

`CommandOptions` has `Cwd`, `Env map[string]string`, `Timeout time.Duration`, optional `Stdin io.Reader`, and `OutputLimit int` (zero selects 4 MiB per output stream). Commands use argument arrays and runtime PATH resolution. Nonzero exits and launch failures return nonzero `CommandResult.Code` plus `Stderr`; context cancellation/deadlines return an error as well. Always inspect `Code` after checking `error`.

Stdout/stderr are drained with a retained prefix per stream to bound memory. `StdoutTruncated` and `StderrTruncated` report when output exceeded the configured limit; protocol consumers must reject truncated responses. Leading/trailing whitespace is trimmed. Windows batch shims run through `cmd.exe`; embedded quotes, newlines, or NUL in their arguments are rejected. Explicit shell scripts remain available through shell invocation.

```go
package example

import (
    "context"
    "fmt"
    "time"
    "github.com/ossl-dev/genesis/packages/core"
)

func CheckGit(ctx context.Context) error {
    result, err := core.RunCommand(ctx, "git", []string{"--version"}, core.CommandOptions{
        Timeout: 15 * time.Second,
    })
    if err != nil { return err }
    if result.Code != 0 { return fmt.Errorf("git unavailable: %s", result.Stderr) }
    return nil
}
```

## Filesystem and environment

```go
func EnsureDir(path string) error
func ResolveGenesisHome() (string, error)
func GetGenesisHome() string
func EnsureGenesisHome() (string, error)
func JoinPath(segments ...string) string
func WritePrivateFile(path string, data []byte) error
func NewEnvironment(values map[string]string) *Environment
```

Genesis home is `~/.genesis`. `ResolveGenesisHome` rejects missing or relative native home directories; use it before writes. `EnsureGenesisHome` propagates resolution errors. The convenience `GetGenesisHome` returns an empty string on failure. `WritePrivateFile` writes through a temporary file in the destination directory with mode 0600 and renames it into place. `NewEnvironment(nil)` copies the host environment; passing an empty map creates an empty environment. `Environment.Get`, `Lookup`, `Set`, `Snapshot`, and `PrependPath` synchronize concurrent access; returned maps are copies. `Lookup` also reports whether a key exists, distinguishing absent keys from explicitly empty values. Windows environment keys are case-insensitive and normalized to avoid conflicting PATH entries.

`Runtime.Logger` is a standard `*log.Logger`; absent loggers discard output. CLI logging goes to stderr, preserving JSON stdout. Library calls do not control process exit status.
