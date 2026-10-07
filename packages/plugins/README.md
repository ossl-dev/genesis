# @ossl/genesis-plugins

Seven built-in Genesis plugins: Node, Python, Go, Java, Git, Docker, and Homebrew. Import helper factories from the package root or individual subpaths.

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, git } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [node({ version: "22" }), git()],
});
```

Factories return plugin instances; `loadPlugins` loads their implementations and validates/defaults options. Global packages install after Node/Homebrew is ready. Requested runtimes already present on PATH skip system prerequisites.

See the [plugin overview](../../apps/docs/plugins/overview.md) for supported paths and the [lifecycle reference](../../apps/docs/plugins/lifecycle.md) for dependencies and hooks. Windows installation and non-APT Linux package tasks are unfinished. Java archive resolution and Git source/binary installation are experimental. Rollback is not implemented.

From the repository root: `bun run build`, `bun run lint`, `bun run test`.
