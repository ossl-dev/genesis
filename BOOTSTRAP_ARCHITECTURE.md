# Standalone Bootstrap Design

Status: proposal. The CLI currently runs from built JavaScript with Bun or Node. It does not ship a published standalone binary or bootstrap its own runtime.

The intended distribution is a per-platform executable that can run on a new machine and provision development tools from a config. Before implementing it, resolve:

- Bundling built-in plugins despite dynamic module imports.
- Supporting trusted custom modules and TypeScript configs without assuming a global runtime.
- Building and testing one target per OS/architecture, including ARM64.
- Packaging installers without embedding credentials or environment-specific paths.
- Verifying downloads and publishing release assets through a tested CI pipeline.
- Making partial installation failures recoverable before claiming zero-setup reliability.

The existing multi-target standalone build scripts are experimental; they are not a release pipeline. Homebrew/winget/apt distribution and self-bootstrap remain roadmap items.

See [ROADMAP.md](ROADMAP.md) and the [installation guide](apps/docs/guide/installation.md) for the supported workflow.
