# What Is Genesis?

Genesis aims to make development onboarding repeatable: a team checks one config into its repository, and contributors apply that config to provision tools, clone repositories, and run setup scripts.

The vision is a portable environment definition with plugin-based installers, useful previews, and reliable verification. The current implementation focuses on local host provisioning. It does not isolate projects, provide rollback, or guarantee identical package versions across every OS.

## What works

- YAML and TypeScript config loading, environment references, and load-time validation.
- Seven built-in plugins, with different installation limits by platform.
- Deduplicated system prerequisites and dependency-ordered plugin lifecycle hooks.
- Repository cloning that preserves existing worktrees, plus before/after scripts.
- Dry-run action plans, JSON export, detection, and failed-check exit statuses.
- An opt-in parallel core API; CLI provisioning is sequential.

## What remains

Cloud integration, working environment cache restore, complete Windows/Linux distribution support, safe archive staging and rollback, additional plugins, and published standalone binaries. The roadmap distinguishes these from shipped behavior.

Start with [Installation](/guide/installation) and [Configuration](/guide/configuration).
