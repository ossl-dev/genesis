#!/usr/bin/env node
import { Command } from "commander";
import { registerCreateCommand } from "./commands/create.js";
import { registerApplyCommand } from "./commands/apply.js";
import { registerDoctorCommand } from "./commands/doctor.js";
import { registerDiffCommand } from "./commands/diff.js";
import { registerValidateCommand } from "./commands/validate.js";
import { registerInitCommand } from "./commands/init.js";
import { registerListPluginsCommand } from "./commands/list-plugins.js";
import { registerLoginCommand } from "./commands/login.js";
import { registerListCommand } from "./commands/list.js";

export function createCli(): Command {
  const program = new Command();
  program.name("genesis");
  program.description(
    "Rebuild any developer environment. Deterministic. Portable. Declarative.",
  );
  registerCreateCommand(program);
  registerApplyCommand(program);
  registerDoctorCommand(program);
  registerDiffCommand(program);
  registerValidateCommand(program);
  registerInitCommand(program);
  registerListPluginsCommand(program);
  registerLoginCommand(program);
  registerListCommand(program);
  return program;
}

export async function run(
  argv: string[] = globalThis.process?.argv.slice() || [],
): Promise<void> {
  const program = createCli();
  await program.parseAsync(argv);
}

run().catch((error) => {
  const value = error instanceof Error ? error.message : String(error);
  console.error(value);
  if (typeof globalThis.process !== "undefined") {
    globalThis.process.exit(1);
  }
});
