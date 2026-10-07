# Contributing

Work on a focused item from the roadmap or a reproducible bug. Describe the actual behavior before/after the change and record how it was verified.

```bash
bun install --frozen-lockfile
bun run build
bun run test
bun run lint
bun run docs:build
```

Use existing commit conventions: `fix:`, `feat:`, `refactor:`, or `docs:` followed by a concrete description. Keep related behavior, tests, and documentation together.

Add tests for meaningful execution failures, ordering, and edge cases. Mock installer commands so tests do not change the host. Use real temporary directories and local Git repositories for integration behavior. Ensure path assertions can run on the CI OS matrix.

Keep docs concise and matched to exported code. Mark experimental installer paths and unfinished roadmap items explicitly. Do not claim speedups, cloud integration, or platform support without evidence.

See [Plugin Development](/guide/plugin-development) for extension points. Report issues at [ossl-dev/genesis](https://github.com/ossl-dev/genesis/issues).
