import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { TaskRegistry, runCommand } from "../packages/core/dist/index.js";
import { docker, createPlugin } from "../packages/plugins/src/plugins/docker/index.js";
import { installDockerLinux } from "../packages/plugins/src/install/docker-linux.js";

if (process.platform !== "linux" || process.getuid?.() !== 0 || process.env.GENESIS_DISPOSABLE_HOST !== "1") throw new Error("Run only in a disposable Linux container with GENESIS_DISPOSABLE_HOST=1");
const cwd = await fs.mkdtemp(path.join(os.tmpdir(), "genesis-docker-"));
const logger = { info: console.log, warn: console.warn, error: console.error, debug: () => {} };
try {
  // Package tests need no privileged daemon or systemd inside the container.
  const bin = path.join(cwd, "bin");
  await fs.mkdir(bin);
  await fs.writeFile(path.join(bin, "systemctl"), "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  const context = { cwd, env: { ...process.env, PATH: `${bin}:${process.env.PATH}` }, logger, taskRegistry: new TaskRegistry(logger) };
  const instance = docker({ version: "29", include_compose: false });
  const plugin = createPlugin(instance);
  const runtime = { instance, options: plugin.parseOptions!(instance.options), context };
  await plugin.prepare!(runtime);
  await plugin.registerTasks!(runtime);
  const tasks = await context.taskRegistry.executeAll();
  for (const [id, result] of tasks) if (!result.ok) throw new Error(`${id}: ${result.error}`);
  await installDockerLinux(runtime, () => {});
  if (!(await plugin.detect!(runtime)).ok) throw new Error("Pinned Docker client was not installed");
  const compose = await runCommand("docker", ["compose", "version"], { cwd, env: context.env });
  if (compose.code === 0) throw new Error("Compose was installed despite include_compose: false");
  runtime.options.include_compose = true;
  await installDockerLinux(runtime, () => {}, true);
  if (!(await plugin.detect!(runtime)).ok) throw new Error("Docker Compose was not installed");
  const applied = await plugin.apply!(runtime);
  if (applied.ok || applied.didChange || !applied.details?.includes("daemon is unavailable")) throw new Error(`Missing daemon was not reported: ${applied.details}`);
  console.log("Pinned Docker packages, optional Compose, and missing-daemon failure verified");
} finally { await fs.rm(cwd, { recursive: true, force: true }); }
