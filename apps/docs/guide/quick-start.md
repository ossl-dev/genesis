# Quick Start

Follow [Getting Started](/guide/getting-started) for the supported source-build workflow.

The essential sequence is: create a local config, preview with `apply --dry-run`, apply it, then validate the requested tools. Explicit config selection is available through `apply --config <file>`; JSON plan export uses `apply --dry-run --json`.

There is no working cloud apply, cache restore, or binary install endpoint yet.
