export const GENESIS_PLUGIN_CATEGORIES = [
  "tool",
  "sdk",
  "language",
  "library",
  "framework",
] as const;

export type GenesisPluginCategory = (typeof GENESIS_PLUGIN_CATEGORIES)[number];

export interface RepoSpec {
  url: string;
  folder: string;
  branch?: string;
}

export interface ScriptSpec {
  name: string;
  command: string;
  description?: string;
  when?: "before" | "after";
}

export interface GenesisPluginInstance<TOptions = unknown> {
  id: string;
  category: GenesisPluginCategory;
  module: string;
  options?: TOptions;
}

export interface GenesisConfig {
  tools?: GenesisPluginInstance[];
  sdks?: GenesisPluginInstance[];
  languages?: GenesisPluginInstance[];
  repositories?: RepoSpec[];
  scripts?: ScriptSpec[];
  env?: Record<string, string>;
}
