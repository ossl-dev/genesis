import type { Command } from "commander";

import { BUILTIN_PLUGINS } from "@ossl/genesis-core";

export function registerListPluginsCommand(program: Command): void {
  program
    .command("list-plugins")
    .description("List available Genesis plugins")
    .option("--json", "Output as JSON")
    .action(async (options) => {
      const plugins = BUILTIN_PLUGINS;

      if (options.json) {
        console.log(JSON.stringify(plugins, null, 2));
        return;
      }

      console.log(`Available plugins (${plugins.length}):`);
      console.log("");
      const moduleWidth = Math.max(...plugins.map((p) => p.module.length));
      for (const plugin of plugins) {
        console.log(`  ${plugin.module.padEnd(moduleWidth + 2)} ${plugin.description}`);
      }
      console.log("");
      console.log("Usage in genesis.config.yaml:");
      console.log("  tools:");
      console.log("    - type: node");
      console.log('      version: "20"');
    });
}
