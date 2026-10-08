import { type GenesisPluginInstance } from "@ossl/genesis-core";
import { optionSchemas } from "../../options.js";
import { createBinaryPlugin, type BinaryOptions } from "../../install/binary-plugin.js";
import { denoRelease } from "../../install/releases.js";

export type DenoOptions = BinaryOptions;

export function deno(options: DenoOptions): GenesisPluginInstance<DenoOptions> {
  return { id: "deno", category: "tool", module: "@ossl/genesis-plugins/deno", options };
}

export function createPlugin(instance: GenesisPluginInstance<DenoOptions>) {
  return createBinaryPlugin(instance, {
    name: "deno",
    parseOptions: options => optionSchemas.deno.parse(options),
    release: (options, platform) => denoRelease(options.version, platform),
  });
}
