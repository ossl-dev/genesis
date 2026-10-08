import type { GenesisPluginCategory } from "../config/schema.js";

const definitions = [
  { id: "node", category: "tool", description: "Node.js JavaScript runtime" },
  { id: "bun", category: "tool", description: "Bun JavaScript runtime" },
  { id: "deno", category: "tool", description: "Deno JavaScript runtime" },
  { id: "pnpm", category: "tool", description: "pnpm package manager" },
  { id: "yarn", category: "tool", description: "Yarn Classic and modern package manager" },
  { id: "python", category: "language", description: "Python programming language" },
  { id: "go", category: "language", description: "Go programming language" },
  { id: "java", category: "language", description: "Java development kit" },
  { id: "docker", category: "tool", description: "Docker container runtime" },
  { id: "homebrew", category: "tool", description: "Homebrew package manager (macOS)" },
  { id: "git", category: "tool", description: "Git version control system" },
] satisfies { id: string; category: GenesisPluginCategory; description: string }[];

export const BUILTIN_PLUGINS = definitions.map(plugin => ({ ...plugin, module: `@ossl/genesis-plugins/${plugin.id}` }));
