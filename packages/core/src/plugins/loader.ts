import { GenesisConfig, GenesisPluginInstance } from "../config/schema.js";
import { GenesisPlugin } from "./types.js";

export interface PluginExecutionNode<TOptions = unknown> {
  instance: GenesisPluginInstance<TOptions>;
  plugin: GenesisPlugin<TOptions>;
}

export function collectPluginInstances(
  config: GenesisConfig
): GenesisPluginInstance[] {
  const result: GenesisPluginInstance[] = [];
  if (config.tools) {
    result.push(...config.tools);
  }
  if (config.sdks) {
    result.push(...config.sdks);
  }
  if (config.languages) {
    result.push(...config.languages);
  }
  return result;
}

async function importPluginModule(moduleId: string): Promise<unknown> {
  const mod = await import(moduleId);
  return mod;
}

export async function loadPlugin(
  instance: GenesisPluginInstance
): Promise<PluginExecutionNode> {
  let mod: unknown;
  try {
    mod = await importPluginModule(instance.module);
  } catch (error) {
    throw new Error(`Failed to load plugin '${instance.id}' from '${instance.module}': ${error instanceof Error ? error.message : String(error)}`);
  }
  const create = (
    mod as { createPlugin?: (instance: GenesisPluginInstance) => GenesisPlugin }
  ).createPlugin;
  if (typeof create !== "function") {
    throw new Error(
      `Plugin module '${instance.module}' does not export createPlugin`
    );
  }
  const plugin = create(instance);
  if (!plugin || plugin.id !== instance.id || plugin.category !== instance.category) {
    throw new Error(`Plugin '${instance.id}' returned invalid identity from '${instance.module}'`);
  }
  if (!plugin.parseOptions) return { instance, plugin };
  try {
    return { instance: { ...instance, options: plugin.parseOptions(instance.options) }, plugin };
  } catch (error) {
    throw new Error(`Invalid options for plugin '${instance.id}': ${error instanceof Error ? error.message : String(error)}`);
  }
}

export async function loadPlugins(
  instances: GenesisPluginInstance[]
): Promise<PluginExecutionNode[]> {
  const result: PluginExecutionNode[] = [];
  for (const instance of instances) {
    const node = await loadPlugin(instance);
    result.push(node);
  }
  return result;
}
