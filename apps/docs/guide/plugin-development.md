# Plugin Development

A plugin package exports a helper that creates a `GenesisPluginInstance` and a `createPlugin(instance)` implementation. The configured module must be resolvable by the runtime; absolute file URLs can be used for local JavaScript plugins.

```typescript
import {
  type GenesisPlugin,
  type GenesisPluginInstance,
  runCommand,
} from "@ossl/genesis-core";

interface Options { version: string }

export function tool(options: Options): GenesisPluginInstance<Options> {
  return { id: "my-tool", category: "tool", module: "my-genesis-plugin", options };
}

export function createPlugin(instance: GenesisPluginInstance<Options>): GenesisPlugin<Options> {
  return {
    id: instance.id,
    category: instance.category,
    parseOptions(value) {
      const options = value as Partial<Options> | undefined;
      if (!options || typeof options.version !== "string" || !options.version.trim()) {
        throw new Error("version is required");
      }
      return { version: options.version };
    },
    async detect({ options, context }) {
      const result = await runCommand("my-tool", ["--version"], context);
      return { ok: result.code === 0 && result.stdout.trim() === options.version };
    },
    async validate(runtime) {
      const result = await this.detect!(runtime);
      return { ok: result.ok, message: result.details };
    },
  };
}
```

This example detects an existing tool; it does not install it. Add an apply method appropriate to the tool's supported platforms, and verify the executable/version before returning success.

Register shared prerequisites with exact task IDs in `registerTasks`. Register both package-manager update and package install tasks when using those helpers. Do not register global runtime packages before the runtime exists.

Use `dependsOn` for configured plugin dependencies. `preApply` and `postApply` run around successful apply after prerequisites; hooks should be safe to repeat. Graph errors and option validation fail before provisioning.

Mock installer commands in unit tests. Verify argument arrays, architecture/distro selection, returned failure results, and hook order. Use temporary directories for actual filesystem/script operations. Avoid changing the development host during tests.

See [Plugin API](/api/plugin), [Lifecycle](/plugins/lifecycle), and [Task Registry](/api/task-registry).
