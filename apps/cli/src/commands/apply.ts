import type { Command } from "commander";
import fs from "node:fs";
import path from "node:path";
import os from "node:os";
import { runApply } from "../lib/runner.js";

function findCachedEnvironment(envId: string): { file: string; data: any } | null {
  const cacheDir = path.join(os.homedir(), ".genesis", "cache");

  try {
    if (!fs.existsSync(cacheDir)) return null;

    // Try direct file match
    const directFile = path.join(cacheDir, `${envId}.json`);
    if (fs.existsSync(directFile)) {
      const data = JSON.parse(fs.readFileSync(directFile, "utf8"));
      return { file: directFile, data };
    }

    // Search through all cache files for matching id or name
    const files = fs.readdirSync(cacheDir).filter((f) => f.endsWith(".json"));
    for (const file of files) {
      try {
        const filePath = path.join(cacheDir, file);
        const data = JSON.parse(fs.readFileSync(filePath, "utf8"));
        if (data.id === envId || data.name === envId) {
          return { file: filePath, data };
        }
      } catch { /* skip corrupted files */ }
    }
  } catch { /* cache dir might not exist */ }

  return null;
}

export function registerApplyCommand(program: Command): void {
  const apply = program
    .command("apply")
    .description("Apply the desired environment described by the config")
    .argument(
      "[environment-id]",
      "Environment ID from Genesis Cloud (optional)",
    )
    .option("--config <path>", "Path to config file")
    .option("--cloud", "Apply environment from Genesis Cloud")
    .action(async (envId, options) => {
      if (envId || options.cloud) {
        // Apply from cloud
        const environmentId = envId || "default";
        console.log(
          `🌐 Looking up environment "${environmentId}" from Genesis Cloud...`,
        );

        // Check local cache first
        const cached = findCachedEnvironment(environmentId);
        if (cached) {
          const env = cached.data;
          console.log(`📦 Found cached environment: "${env.name}"`);
          console.log(`   ID: ${env.id}`);
          console.log(`   Description: ${env.description || "(none)"}`);
          if (env.createdAt) {
            console.log(`   Created: ${new Date(env.createdAt).toLocaleString()}`);
          }
          console.log("");
          console.log("To restore this cached environment, use:");
          console.log("  genesis restore --from-cache");
          console.log("  OR reinstall with: genesis apply");
          return;
        }

        // Check if user is authenticated
        const tokenFile = path.join(os.homedir(), ".genesis", "auth.json");
        let isAuthenticated = false;
        try {
          if (fs.existsSync(tokenFile)) {
            isAuthenticated = true;
          }
        } catch { /* ignore */ }

        if (isAuthenticated) {
          console.log("🌐 Cloud API is not yet available.");
          console.log("   Environment ID resolution from the cloud will be available");
          console.log("   in a future release. For now, apply local configs with:");
          console.log("");
          console.log("     genesis apply");
          console.log("");
          console.log("   Or if you have a cached environment, you can restore it.");
        } else {
          console.log("⚠️  Not authenticated with Genesis Cloud.");
          console.log("   Run `genesis login` to connect.");
          console.log("");
          console.log("   For local development, apply your config with:");
          console.log("     genesis apply");
        }
      } else {
        // Apply local config
        console.log(`🏠 Applying local environment from ${options.config ?? "project config"}...`);
        await runApply({ cwd: process.cwd(), configPath: options.config });
        console.log("✅ Local environment applied successfully!");
      }
    });
}
