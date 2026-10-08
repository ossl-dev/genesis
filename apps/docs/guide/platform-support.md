# Platform Support

CI runs unit and local integration tests on macOS, Linux, and Windows. Installer commands are mocked in plugin tests, so a passing CI run does not demonstrate every real installer works on a fresh machine.

| Platform | Current support |
| --- | --- |
| macOS | brew prerequisites; NVM Node; Python/Git packages; Colima; Go archive. Java archive path experimental. |
| Debian/Ubuntu | APT prerequisites; NVM Node; Python/Git packages; Docker installer; Go archive. Required package versions must exist. |
| Fedora/CentOS | Docker installer branch exists, but shared prerequisites still invoke APT. Incomplete. |
| Arch | Shared package tasks do not select pacman. Incomplete. |
| Windows | Detect/validate preinstalled tools; missing tools generally require manual installation. Scripts use cmd.exe. |

Go and Java extraction requires permissions for system directories. The code does not automatically elevate archive extraction. Node through NVM is a user installation.

On macOS, include the Homebrew plugin to bootstrap brew before shared system tasks, or provide an existing brew installation.

Go and Java use checksummed staged archives on macOS/Linux/Windows, reject unsupported CPU architectures, and restore the previous directory if promotion fails. Docker Desktop downloads select ARM64/x64. Installer coverage still needs disposable hosts across platforms. See each [plugin reference](/plugins/overview).

Configured environment values only affect Genesis and child commands. Shell profile changes are generally printed as instructions rather than persisted. Rollback and project isolation are not implemented.
