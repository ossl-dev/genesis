# @ossl/genesis-plugins

Eleven built-in Genesis plugins: Node, Bun, Deno, pnpm, Yarn, Python, Go, Java, Git, Docker, and Homebrew. Import helper factories from the package root or individual subpaths.

```typescript
import { defineConfig } from "@ossl/genesis-core";
import { node, git } from "@ossl/genesis-plugins";

export default defineConfig({
  tools: [node({ version: "22" }), git()],
});
```

Factories return plugin instances; `loadPlugins` loads their implementations and validates/defaults options. Global packages install after Node/Homebrew is ready. Requested runtimes already present on PATH skip system prerequisites.

See the [plugin overview](../../apps/docs/plugins/overview.md) for supported paths and the [lifecycle reference](../../apps/docs/plugins/lifecycle.md) for dependencies and hooks. Archive and npm-prefix installers support Windows targets; Python Windows installation remains manual. Shared Linux tasks select APT/DNF/pacman/APK. Git supports pinned Unix source builds and Windows MinGit archives. Staged installers recover failed replacements; system-wide rollback is not implemented.

From the repository root: `bun run build`, `bun run lint`, `bun run test`.

Pinned [Bun](../../apps/docs/plugins/bun.md) and [Deno](../../apps/docs/plugins/deno.md) archive plugins are also available as `@ossl/genesis-plugins/bun` and `@ossl/genesis-plugins/deno`.

[pnpm](../../apps/docs/plugins/pnpm.md) and [Yarn](../../apps/docs/plugins/yarn.md) provision pinned package-manager versions after the configured Node plugin, using staged npm prefixes.
