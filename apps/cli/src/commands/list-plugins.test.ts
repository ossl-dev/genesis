import { afterEach, expect, it, vi } from "vitest";
import { Command } from "commander";
import { registerListPluginsCommand } from "./list-plugins.js";
afterEach(() => vi.restoreAllMocks());

it('lists all new plugin modules as parseable JSON', async () => {
  const log = vi.spyOn(console, 'log').mockImplementation(() => {});
  const program = new Command(); registerListPluginsCommand(program);
  await program.parseAsync(['list-plugins', '--json'], { from: 'user' });
  expect(log).toHaveBeenCalledTimes(1);
  const plugins = JSON.parse(log.mock.calls[0][0]);
  expect(plugins.map((plugin: { id: string }) => plugin.id)).toEqual(expect.arrayContaining(['bun', 'deno', 'pnpm', 'yarn']));
  expect(plugins).toHaveLength(11);
  expect(plugins.find((plugin: { id: string }) => plugin.id === 'homebrew').description).toContain('(macOS)');
});
