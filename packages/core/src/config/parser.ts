import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import yaml from "yaml";
import { z } from "zod";
import { type GenesisConfig } from "./schema.js";
import { BUILTIN_PLUGINS } from "../plugins/catalog.js";
import { validateConfig } from "./validator.js";

const yamlPluginDefaults = new Map(BUILTIN_PLUGINS.map(plugin => [plugin.id, plugin]));

function normalizeYamlConfig(raw: unknown): unknown {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return raw;
  const config = { ...raw } as Record<string, unknown>;
  for (const section of ["tools", "sdks", "languages"]) {
    const entries = config[section];
    if (!Array.isArray(entries)) continue;
    config[section] = entries.map((entry, index) => {
      if (!entry || typeof entry !== "object" || !("type" in entry)) return entry;
      const { type, ...options } = entry;
      const meta = typeof type === "string" ? yamlPluginDefaults.get(type) : undefined;
      if (!meta) {
        throw new Error(`Unknown plugin type '${type}' in genesis.config.yaml (${section}[${index}])`);
      }
      return { id: type, category: meta.category, module: meta.module, options };
    });
  }
  return config;
}

function interpolate(value: unknown, env: NodeJS.ProcessEnv, location = "config"): unknown {
  if (typeof value === "string") {
    return value.replace(/\$\{([A-Za-z_][A-Za-z0-9_]*)\}/g, (_, name: string) => {
      if (env[name] === undefined) throw new Error(`${location}: environment variable '${name}' is not set`);
      return env[name];
    });
  }
  if (Array.isArray(value)) return value.map((entry, index) => interpolate(entry, env, `${location}[${index}]`));
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.entries(value).map(([key, entry]) => [key, interpolate(entry, env, `${location}.${key}`)]));
  }
  return value;
}

export async function loadConfig(
  cwd: string,
  configPath?: string,
  env: NodeJS.ProcessEnv = process.env,
): Promise<GenesisConfig> {
  const file = configPath
    ? path.resolve(cwd, configPath)
    : ["genesis.config.ts", "genesis.config.yaml"].map(name => path.join(cwd, name)).find(candidate => fs.existsSync(candidate));
  if (!file) throw new Error("No genesis.config.ts or genesis.config.yaml found");
  try {
    let raw: unknown;
    if (path.extname(file) === ".ts") {
      const mod = await import(pathToFileURL(file).href);
      raw = mod.default ?? mod.config ?? mod;
    } else if ([".yaml", ".yml"].includes(path.extname(file))) {
      raw = normalizeYamlConfig(yaml.parse(await fs.promises.readFile(file, "utf8")));
    } else {
      throw new Error("Config file must use .ts, .yaml, or .yml");
    }
    return validateConfig(interpolate(raw, env));
  } catch (error) {
    const message = error instanceof z.ZodError
      ? error.issues.map(issue => `${issue.path.join(".") || "config"}: ${issue.message}`).join("\n")
      : error instanceof Error ? error.message : String(error);
    throw new Error(`${file}: ${message}`);
  }
}
