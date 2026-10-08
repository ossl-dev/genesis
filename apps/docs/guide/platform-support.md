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

The installer smoke workflow downloads and applies Node, Go, Java, Bun, Deno, pnpm, Yarn Classic, modern Yarn, and Git in temporary directories on macOS/Ubuntu/Windows CI. It validates executables and verifies that a new-context second apply is unchanged. Successful archive/npm installs and a Git source build have been observed locally on macOS ARM64. Linux container jobs verify Git, compilers, and mapped build libraries on Debian/Fedora/Arch/Alpine. Debian/Fedora also install pinned Docker packages, check optional Compose, and verify failure without a daemon. These Docker package tests stub systemctl; daemon startup still needs disposable systemd hosts. Local ARM64 Debian/Fedora/Alpine checks and an emulated x64 Arch check passed; the latter disabled pacman syscall sandboxing for emulation only. Native CI outcomes and Windows MinGit execution still need observation. All installer jobs require network access.
