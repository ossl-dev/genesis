import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import yaml from "yaml";
import {
  type GenesisConfig,
  type GenesisPluginInstance,
  type GenesisPluginCategory,
} from "./schema.js";
import { validateConfig } from "./validator.js";

interface YamlPluginEntry {
  type: string;
  [key: string]: unknown;
}

interface YamlConfig {
  tools?: YamlPluginEntry[];
  sdks?: YamlPluginEntry[];
  languages?: YamlPluginEntry[];
  repositories?: GenesisConfig["repositories"];
  scripts?: GenesisConfig["scripts"];
  env?: GenesisConfig["env"];
}

const yamlPluginDefaults: Record<
  string,
  { module: string; category: GenesisPluginCategory }
> = {
  node: { module: "@ossl/genesis-plugins/node", category: "tool" },
  python: { module: "@ossl/genesis-plugins/python", category: "language" },
};

function yamlEntryToInstance(entry: YamlPluginEntry): GenesisPluginInstance {
  const meta = yamlPluginDefaults[entry.type];
  if (!meta) {
    throw new Error(
      `Unknown plugin type '${entry.type}' in genesis.config.yaml`
    );
  }
  const { type, ...options } = entry;
  return {
    id: type,
    category: meta.category,
    module: meta.module,
    options,
  };
}

function isGenesisConfigShape(value: unknown): boolean {
  if (!value || typeof value !== "object") {
    return false;
  }
  const data = value as Record<string, unknown>;
  const sections = ["tools", "sdks", "languages"];
  for (const section of sections) {
    const list = data[section];
    if (Array.isArray(list) && list.length > 0) {
      const first = list[0] as Record<string, unknown>;
      if (!first.id || !first.module || !first.category) {
        return false;
      }
    }
  }
  return true;
}

function normalizeYamlConfig(raw: unknown): GenesisConfig {
  const value = raw as YamlConfig;
  const config: GenesisConfig = {};
  if (value.tools) {
    config.tools = value.tools.map(yamlEntryToInstance);
  }
  if (value.sdks) {
    config.sdks = value.sdks.map(yamlEntryToInstance);
  }
  if (value.languages) {
    config.languages = value.languages.map(yamlEntryToInstance);
  }
  if (value.repositories) {
    config.repositories = value.repositories;
  }
  if (value.scripts) {
    config.scripts = value.scripts;
  }
  if (value.env) {
    config.env = value.env;
  }
  return validateConfig(config);
}

export async function loadConfig(cwd: string): Promise<GenesisConfig> {
  const tsPath = path.join(cwd, "genesis.config.ts");
  const yamlPath = path.join(cwd, "genesis.config.yaml");
  if (fs.existsSync(tsPath)) {
    const url = pathToFileURL(tsPath).href;
    const mod = await import(url);
    const value = mod.default ?? mod.config ?? mod;
    return validateConfig(value);
  }
  if (fs.existsSync(yamlPath)) {
    const rawText = await fs.promises.readFile(yamlPath, "utf8");
    const raw = yaml.parse(rawText);
    if (isGenesisConfigShape(raw)) {
      return validateConfig(raw);
    }
    return normalizeYamlConfig(raw);
  }
  throw new Error("No genesis.config.ts or genesis.config.yaml found");
}
