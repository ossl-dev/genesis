# Platform Support

CI runs unit and local integration tests on macOS, Linux, and Windows. Installer commands are mocked in plugin tests, so a passing CI run does not demonstrate every real installer works on a fresh machine.

| Platform | Current support |
| --- | --- |
| macOS | brew prerequisites; NVM/archive Node; Python/Git packages; Colima; verified staged Go/Java/Bun/Deno archives; pnpm/Yarn prefixes. |
| Debian/Ubuntu | APT prerequisites; NVM/archive Node; Python/Git packages; Docker installer; Go archive. Required package versions must exist. |
| Fedora/RHEL family | DNF prerequisites and common build dependency mappings. Individual runtime/package availability varies. |
| Arch / Alpine | pacman / APK prerequisites. Arch requires an already maintained package database; archive runtime libc restrictions still apply. |
| Arch | Shared package tasks do not select pacman. Incomplete. |
| Windows | Node/Go/Java/Bun/Deno archives and pnpm/Yarn prefixes; Git MinGit archives; Python/Docker setup still manual. Scripts use cmd.exe. |

Default Go and Java Unix destinations require permissions; `install_dir` can select a writable location. The code does not automatically elevate archive extraction. Node through NVM is a user installation.

On macOS, include the Homebrew plugin to bootstrap brew before shared system tasks, or provide an existing brew installation.

Node, Go, Java, Bun, and Deno use checksummed staged archives on macOS/Linux/Windows, reject unsupported CPU architectures, and restore the previous directory if promotion fails. Linux Docker resolves published x64/ARM64 Engine/CLI packages; Docker Desktop setup is manual. Installer coverage still needs disposable hosts across platforms. See each [plugin reference](/plugins/overview).

Configured environment values only affect Genesis and child commands. Shell profile changes are generally printed as instructions rather than persisted. System-wide rollback and project isolation are not implemented; staged runtime replacement has recovery.

The Go installer smoke workflow installs Node, Go, Java, Bun, Deno, pnpm,
Yarn Classic, modern Yarn, and Git in temporary prefixes on macOS, Ubuntu, and
Windows CI. All nine fresh installs, executable validation, and fresh-context
unchanged reapply passed locally on macOS ARM64 after the Go port.

Local race tests and six-platform cross builds passed. Cross-compilation does
not prove native Windows execution. Native Windows MinGit/shim execution and
remote CI results still need observation. Homebrew bootstrap and real Linux
systemd daemon startup still need disposable native hosts.

Go Linux container fixtures cover Debian, Fedora, Arch, and Alpine shared
packages; Debian/Fedora also check real Docker release pins and Compose-only
installation. Docker package tests intentionally stub systemctl, so they do not
prove daemon startup. These Go container checks could not run locally during
migration because the existing Docker daemon did not answer bounded probes.
Earlier pre-migration distro checks are historical evidence, not Go validation.
All installer jobs require network access.
