import type { Command } from "commander";

export function registerListPluginsCommand(program: Command): void {
  program
    .command("list-plugins")
    .description("List available Genesis plugins")
    .action(async () => {
      const plugins = [
        {
          id: "node",
          module: "@ossl/genesis-plugins/node",
          description: "Node.js runtime",
        },
      ];
      for (const plugin of plugins) {
        console.log(`${plugin.module}  ${plugin.description}`);
      }
    });
}
