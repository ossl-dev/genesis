# Quick Start

[Build Genesis](/guide/installation), run `genesis init --format yaml`, then
`genesis apply --dry-run`, `genesis apply`, and `genesis doctor`.

Explicit config: `genesis apply --config configs/dev.yml`.
JSON plan: `genesis apply --dry-run --json`.
Bounded concurrency: `genesis apply --parallel 4`.

Cloud apply and environment restore are not implemented.
