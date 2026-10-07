# Development Setup

```bash
git clone https://github.com/ossl-dev/genesis.git
cd genesis
bun install --frozen-lockfile
bun run build
bun run test
bun run lint
```

The workspace uses Bun and Turborepo. Core must build before consumers can resolve its exported JavaScript/types. `bun run lint` currently runs TypeScript checks, not an ESLint ruleset. Tests use Vitest; installer unit tests mock shell commands, while environment integration tests run scripts and Git in temporary directories.

```bash
bun run docs:dev
bun run docs:build
```

Do not run real installer configs as routine tests on your development machine. Use mocks for command/architecture selection, and temporary directories for filesystem/repository/script behavior.

See [Architecture](/guide/architecture) for package responsibilities and [Contributing](/guide/contributing) for the review workflow.
