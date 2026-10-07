import { z } from "zod";
import {
  GENESIS_PLUGIN_CATEGORIES,
  type GenesisConfig,
  type GenesisPluginInstance,
  type RepoSpec,
  type ScriptSpec,
} from "./schema.js";

const pluginCategoryEnum = z.enum(GENESIS_PLUGIN_CATEGORIES);

const pluginInstanceSchema: z.ZodType<GenesisPluginInstance> = z.object({
  id: z.string().trim().min(1),
  category: pluginCategoryEnum,
  module: z.string().trim().min(1),
  options: z.unknown().optional(),
});

const repoSpecSchema: z.ZodType<RepoSpec> = z.object({
  url: z.string().min(1),
  folder: z.string().min(1),
  branch: z.string().min(1).optional(),
});

const scriptSpecSchema: z.ZodType<ScriptSpec> = z.object({
  name: z.string(),
  command: z.string().min(1),
  description: z.string().optional(),
  when: z.enum(["before", "after"]).optional(),
});

const genesisConfigSchema: z.ZodType<GenesisConfig> = z.object({
  tools: z.array(pluginInstanceSchema).optional(),
  sdks: z.array(pluginInstanceSchema).optional(),
  languages: z.array(pluginInstanceSchema).optional(),
  repositories: z.array(z.preprocess(value => {
    if (value && typeof value === "object" && "path" in value && !("folder" in value)) {
      const { path, ...repo } = value as Record<string, unknown>;
      return { ...repo, folder: path };
    }
    return value;
  }, repoSpecSchema)).optional(),
  scripts: z.array(scriptSpecSchema).optional(),
  env: z.record(z.string(), z.string()).optional(),
});

export function validateConfig(value: unknown): GenesisConfig {
  const parsed = genesisConfigSchema.parse(value);
  const seen = new Set<string>();
  for (const section of ["tools", "sdks", "languages"] as const) {
    for (const [index, plugin] of (parsed[section] ?? []).entries()) {
      if (seen.has(plugin.id)) {
        throw new z.ZodError([{ code: "custom", path: [section, index, "id"], message: `Duplicate plugin id '${plugin.id}'` }]);
      }
      seen.add(plugin.id);
    }
  }
  return parsed;
}
