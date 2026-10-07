import { Command } from "commander";
import os from "node:os";
import path from "node:path";
import fs from "node:fs";

const TOKEN_FILE = path.join(os.homedir(), ".genesis", "auth.json");

function saveToken(token: string, url: string): void {
  const dir = path.dirname(TOKEN_FILE);
  fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
  fs.writeFileSync(TOKEN_FILE, JSON.stringify({ token, url, savedAt: new Date().toISOString() }), { mode: 0o600 });
  fs.chmodSync(TOKEN_FILE, 0o600);
}

function getStoredToken(): { token: string; url: string; savedAt?: string } | null {
  try {
    if (fs.existsSync(TOKEN_FILE)) {
      const stored = JSON.parse(fs.readFileSync(TOKEN_FILE, "utf8"));
      if (typeof stored?.token === "string" && stored.token.length > 0 && typeof stored?.url === "string") return stored;
    }
  } catch { /* ignore */ }
  return null;
}

export function registerLoginCommand(program: Command): void {
  const login = program.command("login").description("Store cloud token metadata (backend unavailable)");

  login
    .option("-t, --token <token>", "Authentication token")
    .option(
      "-u, --url <url>",
      "Genesis Cloud URL",
      "https://cloud.genesis-docs.vercel.app",
    )
    .action(async (options) => {
      if (options.token) {
        saveToken(options.token, options.url);
        console.log(`Token saved for ${options.url}`);
        console.log(`   Token saved to ${TOKEN_FILE}`);
        return;
      }

      // Check for stored token
      const stored = getStoredToken();
      if (stored) {
        console.log(`A token is saved for ${stored.url}`);
        console.log(`   Saved: ${stored.savedAt}`);
        console.log("");
        console.log("To replace the saved token, run:");
        console.log("  genesis login --token <new-token>");
        return;
      }

      console.log("Cloud OAuth is not implemented. Token storage is available locally.");
      console.log("Use genesis login --token <token> to save metadata without verifying cloud access.");
    });
}
