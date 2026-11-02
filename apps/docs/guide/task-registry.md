# Task Registry

`core.NewTaskRegistry()` deduplicates system operations by exact ID. Register
`core.Task` values with explicit dependencies and priorities; execute with
`ExecuteAll(context.Context)`. Missing dependencies and cycles fail before work;
failed dependencies block downstream tasks. Results retain failure details.

Plugins use `core.RegisterPackageTasks(runtime, "curl", "git")` for shared
package-manager metadata and installs. Runtime cwd/env propagate to commands.
APT/DNF/pacman/APK selection uses distro ID then ID_LIKE. Package names are
validated before command construction; build dependencies map across managers.

Pacman never runs an isolated metadata refresh: maintain the host with a full
`pacman -Syu` before provisioning. Chocolatey never upgrades every host package.
See [API](/api/task-registry) and [deduplication](/guide/task-deduplication).
