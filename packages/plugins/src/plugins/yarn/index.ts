import { type GenesisPluginInstance } from "@ossl/genesis-core";
import { optionSchemas } from "../../options.js";
import { createNpmPlugin, type PackageManagerOptions } from "../../install/npm-plugin.js";

export type YarnOptions = PackageManagerOptions;

export function yarn(options: YarnOptions): GenesisPluginInstance<YarnOptions> {
  return { id: "yarn", category: "tool", module: "@ossl/genesis-plugins/yarn", options };
}

export function createPlugin(instance: GenesisPluginInstance<YarnOptions>) {
  return createNpmPlugin(instance, "yarn", options => optionSchemas.yarn.parse(options));
}
