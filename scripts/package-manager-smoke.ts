import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { createPackageManagerUpdateTask, createPackageInstallTask, getLinuxDistribution, runCommand } from "../packages/core/dist/index.js";

if (process.platform !== "linux" || process.getuid?.() !== 0 || process.env.GENESIS_DISPOSABLE_HOST !== "1") {
  throw new Error("Run this smoke test only in a disposable Linux container with GENESIS_DISPOSABLE_HOST=1");
}
const cwd = await fs.mkdtemp(path.join(os.tmpdir(), "genesis-packages-"));
const env = { ...process.env };
try {
  const tasks = [createPackageManagerUpdateTask(cwd, env), ...["git", "build-essential", "libcurl4-openssl-dev", "zlib1g-dev", "libexpat1-dev"].map(name => createPackageInstallTask(name, cwd, env))];
  for (const task of tasks) {
    console.log(task.description);
    const result = await task.executor();
    if (!result.ok) throw new Error(`${task.id}: ${result.error}`);
  }
  const git = await runCommand("git", ["--version"], { cwd, env });
  if (git.code !== 0) throw new Error(git.stderr);
  const source = path.join(cwd, "probe.c");
  const binary = path.join(cwd, "probe");
  await fs.writeFile(source, '#include <curl/curl.h>\n#include <expat.h>\n#include <zlib.h>\nint main(void) { return !(curl_version() && XML_ExpatVersion() && zlibVersion()); }\n');
  const compiled = await runCommand("cc", [source, "-lcurl", "-lexpat", "-lz", "-o", binary], { cwd, env });
  if (compiled.code !== 0) throw new Error(compiled.stderr);
  const probe = await runCommand(binary, [], { cwd, env });
  if (probe.code !== 0) throw new Error("Installed build libraries failed the runtime probe");
  console.log(`${getLinuxDistribution().id}: Git, compiler, and mapped libraries verified`);
} finally { await fs.rm(cwd, { recursive: true, force: true }); }
