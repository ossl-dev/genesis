// Node.js plugin
export {
  node,
  createPlugin as createNodePlugin,
} from "./plugins/node/index.js";
export type { NodeOptions } from "./plugins/node/index.js";

// Python plugin
export {
  python,
  createPlugin as createPythonPlugin,
} from "./plugins/python/index.js";
export type { PythonOptions } from "./plugins/python/index.js";

// Java plugin
export {
  java,
  createPlugin as createJavaPlugin,
} from "./plugins/java/index.js";
export type { JavaOptions } from "./plugins/java/index.js";

// Go plugin
export { go, createPlugin as createGoPlugin } from "./plugins/go/index.js";
export type { GoOptions } from "./plugins/go/index.js";

// Git plugin
export { git, createPlugin as createGitPlugin } from "./plugins/git/index.js";
export type { GitOptions } from "./plugins/git/index.js";

// Docker plugin
export {
  docker,
  createPlugin as createDockerPlugin,
} from "./plugins/docker/index.js";
export type { DockerOptions } from "./plugins/docker/index.js";

// Homebrew plugin
export {
  homebrew,
  createPlugin as createHomebrewPlugin,
} from "./plugins/homebrew/index.js";
export type { HomebrewOptions } from "./plugins/homebrew/index.js";

export { bun, createPlugin as createBunPlugin } from "./plugins/bun/index.js";
export type { BunOptions } from "./plugins/bun/index.js";
export { deno, createPlugin as createDenoPlugin } from "./plugins/deno/index.js";
export type { DenoOptions } from "./plugins/deno/index.js";
export { pnpm, createPlugin as createPnpmPlugin } from "./plugins/pnpm/index.js";
export type { PnpmOptions } from "./plugins/pnpm/index.js";
export { yarn, createPlugin as createYarnPlugin } from "./plugins/yarn/index.js";
export type { YarnOptions } from "./plugins/yarn/index.js";
