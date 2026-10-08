import { type GenesisPluginInstance } from "@ossl/genesis-core";
import { optionSchemas } from "../../options.js";
import { createNpmPlugin, type PackageManagerOptions } from "../../install/npm-plugin.js";

export type PnpmOptions = PackageManagerOptions;

export function pnpm(options: PnpmOptions): GenesisPluginInstance<PnpmOptions> {
  return { id: "pnpm", category: "tool", module: "@ossl/genesis-plugins/pnpm", options };
}

export function createPlugin(instance: GenesisPluginInstance<PnpmOptions>) {
  return createNpmPlugin(instance, "pnpm", options => optionSchemas.pnpm.parse(options));
}
