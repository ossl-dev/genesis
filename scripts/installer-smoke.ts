import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { TaskRegistry, type GenesisPlugin, type GenesisPluginInstance } from "../packages/core/dist/index.js";
import { node, createPlugin as nodePlugin } from "../packages/plugins/src/plugins/node/index.js";
import { go, createPlugin as goPlugin } from "../packages/plugins/src/plugins/go/index.js";
import { java, createPlugin as javaPlugin } from "../packages/plugins/src/plugins/java/index.js";
import { bun, createPlugin as bunPlugin } from "../packages/plugins/src/plugins/bun/index.js";
import { deno, createPlugin as denoPlugin } from "../packages/plugins/src/plugins/deno/index.js";
import { pnpm, createPlugin as pnpmPlugin } from "../packages/plugins/src/plugins/pnpm/index.js";
import { git, createPlugin as gitPlugin } from "../packages/plugins/src/plugins/git/index.js";
import { yarn, createPlugin as yarnPlugin } from "../packages/plugins/src/plugins/yarn/index.js";

const root = await fs.mkdtemp(path.join(os.tmpdir(), "genesis installer smoke-"));
const logger = { info: console.log, warn: console.warn, error: console.error, debug: () => {} };

let nodeEnvironment: NodeJS.ProcessEnv | undefined;

async function check<T>(instance: GenesisPluginInstance<T>, plugin: GenesisPlugin<T>) {
  if (!plugin.parseOptions || !plugin.apply || !plugin.validate) throw new Error(`Incomplete plugin: ${instance.id}`);
  const options = plugin.parseOptions(instance.options);
  const context = { cwd: root, env: { ...(nodeEnvironment ?? process.env), npm_config_cache: path.join(root, "npm-cache") }, logger, taskRegistry: new TaskRegistry(logger) };
  const runtime = { instance, options, context };
  const first = await plugin.apply(runtime);
  if (!first.ok || !first.didChange) throw new Error(`${instance.id} fresh install: ${first.details}`);
  const validation = await plugin.validate(runtime);
  if (!validation.ok) throw new Error(`${instance.id} validation: ${validation.message}`);
  // A new context must discover the install without PATH from the first apply.
  const repeated = await plugin.apply({ ...runtime, context: { ...context, env: { ...(nodeEnvironment ?? process.env) } } });
  if (instance.id === "node") nodeEnvironment = { ...context.env };
  if (!repeated.ok || repeated.didChange) throw new Error(`${instance.id} repeated apply: ${repeated.details}`);
  console.log(`${instance.id}: installed, validated, and rediscovered on ${process.platform}/${process.arch}`);
}

try {
  const nodeInstance = node({ version: "22.18.0", use_nvm: false, install_dir: path.join(root, "node") });
  await check(nodeInstance, nodePlugin(nodeInstance));
  const goInstance = go({ version: "1.22.5", install_dir: path.join(root, "go") });
  await check(goInstance, goPlugin(goInstance));
  const javaInstance = java({ version: "17.0.9", install_dir: path.join(root, "java") });
  await check(javaInstance, javaPlugin(javaInstance));
  const bunInstance = bun({ version: "1.3.2", install_dir: path.join(root, "bun") });
  await check(bunInstance, bunPlugin(bunInstance));
  const denoInstance = deno({ version: "2.5.4", install_dir: path.join(root, "deno") });
  await check(denoInstance, denoPlugin(denoInstance));
  const pnpmInstance = pnpm({ version: "10.19.0", install_dir: path.join(root, "pnpm"), node_plugin: null });
  await check(pnpmInstance, pnpmPlugin(pnpmInstance));
  const yarnInstance = yarn({ version: "4.9.4", install_dir: path.join(root, "yarn"), node_plugin: null });
  await check(yarnInstance, yarnPlugin(yarnInstance));
  const gitInstance = git({ version: "2.53.0", install_method: process.platform === "win32" ? "binary" : "source", install_dir: path.join(root, "git") });
  await check(gitInstance, gitPlugin(gitInstance));
  const classic = yarn({ version: "1.22.22", install_dir: path.join(root, "yarn-classic"), node_plugin: null });
  await check(classic, yarnPlugin(classic));
} finally {
  await fs.rm(root, { recursive: true, force: true });
}
