# What Is Genesis?

Genesis aims to make development onboarding repeatable: a team checks one config into its repository, and contributors apply that config to provision tools, clone repositories, and run setup scripts.

The vision is a portable environment definition with plugin-based installers, useful previews, and reliable verification. The current implementation focuses on local host provisioning. It does not isolate projects, provide system-wide rollback, or guarantee identical package versions across every OS. Managed archive prefixes have recovery on failed replacement.

## What works

- YAML and Go config loading, environment references, and load-time validation.
- Eleven built-in plugins, with different installation limits by platform.
- Deduplicated system prerequisites and dependency-ordered plugin lifecycle hooks.
- Repository cloning that preserves existing worktrees, plus before/after scripts.
- Dry-run action plans, JSON export, detection, and failed-check exit statuses.
- Bounded parallel core execution and CLI `apply --parallel N`; sequential by default.

## What remains

Cloud integration, working environment cache restore, remaining Windows/Linux distribution support, system-wide rollback, additional plugins, and package-manager distribution. Standalone builds and version-tag release automation are implemented. The roadmap distinguishes these from shipped behavior.

Start with [Installation](/guide/installation) and [Configuration](/guide/configuration).
