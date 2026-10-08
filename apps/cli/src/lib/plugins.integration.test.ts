import fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { afterEach, expect, it, vi } from "vitest";
import { runApply } from "./runner.js";
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllEnvs(); });

it('loads every packaged built-in from a config outside the CLI workspace', async () => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'genesis-plugin-resolution-'));
  const log = vi.spyOn(console, 'log').mockImplementation(() => {});
  vi.stubEnv('GENESIS_DEBUG', '1');
  try {
    await fs.writeFile(path.join(cwd, 'genesis.config.yaml'), `tools:
  - { type: node, version: "22", use_nvm: false }
  - { type: bun, version: "1.3.2" }
  - { type: deno, version: "2.5.4" }
  - { type: pnpm, version: "10.19.0" }
  - { type: yarn, version: "4.9.4" }
  - { type: git }
  - { type: docker }
  - { type: homebrew }
languages:
  - { type: go, version: "1.22.5" }
  - { type: java, version: "17" }
  - { type: python, version: "3.11" }
`);
    await runApply({ cwd, dryRun: true, json: true });
    expect(log).toHaveBeenCalledTimes(1);
    const plan = JSON.parse(log.mock.calls[0][0]);
    const ids = plan.actions.filter((action: { type: string }) => action.type === 'plugin').map((action: { id: string }) => action.id);
    expect(new Set(ids).size).toBe(11);
    expect(ids.indexOf('node')).toBeLessThan(ids.indexOf('pnpm'));
    expect(ids.indexOf('node')).toBeLessThan(ids.indexOf('yarn'));
  } finally { await fs.rm(cwd, { recursive: true, force: true }); }
});
