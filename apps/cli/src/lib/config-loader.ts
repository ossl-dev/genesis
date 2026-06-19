import { loadConfig } from "@ossl/genesis-core";

export async function loadGenesisConfig(cwd: string) {
  return loadConfig(cwd);
}

