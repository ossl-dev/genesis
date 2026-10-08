import { type GenesisPluginInstance } from "@ossl/genesis-core";
import { optionSchemas } from "../../options.js";
import { createBinaryPlugin, type BinaryOptions } from "../../install/binary-plugin.js";
import { bunRelease } from "../../install/releases.js";

export interface BunOptions extends BinaryOptions {
  libc?: "auto" | "glibc" | "musl";
}

export function bun(options: BunOptions): GenesisPluginInstance<BunOptions> {
  return { id: "bun", category: "tool", module: "@ossl/genesis-plugins/bun", options };
}

export function createPlugin(instance: GenesisPluginInstance<BunOptions>) {
  return createBinaryPlugin(instance, {
    name: "bun",
    parseOptions: options => optionSchemas.bun.parse(options),
    release: (options, platform) => bunRelease(options.version, platform, options.libc),
  });
}
