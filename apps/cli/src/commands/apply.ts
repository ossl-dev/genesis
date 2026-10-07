import type { Command } from "commander";
import { runApply } from "../lib/runner.js";

export function registerApplyCommand(program: Command): void {
  program
    .command("apply")
    .description("Apply the desired environment described by the config")
    .argument("[environment-id]", "Cloud environment ID (not yet supported)")
    .option("--config <path>", "Path to config file")
    .option("--cloud", "Apply a cloud environment (not yet supported)")
    .option("--dry-run", "Show planned actions without provisioning")
    .option("--json", "Print the dry-run plan as JSON")
    .action(async (envId, options) => {
      if (envId || options.cloud) throw new Error("Cloud environment apply is not available. Use a local config.");
      if (options.json && !options.dryRun) throw new Error("--json requires --dry-run");
      await runApply({ cwd: process.cwd(), configPath: options.config, dryRun: options.dryRun, json: options.json });
      if (!options.dryRun) console.log("Environment applied successfully");
    });
}
