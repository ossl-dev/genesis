import { beforeEach, describe, expect, it, vi } from "vitest";
import path from "node:path";
const mocks = vi.hoisted(() => ({ run: vi.fn(), platform: vi.fn(), exists: vi.fn(), install: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getPlatform: mocks.platform }));
vi.mock("node:fs", () => ({ default: { existsSync: mocks.exists } }));
vi.mock("./archive.js", () => ({ InstallationRecoveryError: class extends Error {}, installDirectory: mocks.install, prependPath: (env: NodeJS.ProcessEnv, dir: string) => { env.PATH = dir; } }));
import { pnpm, createPlugin as pnpmPlugin } from "../plugins/pnpm/index.js";
import { yarn, createPlugin as yarnPlugin } from "../plugins/yarn/index.js";
beforeEach(() => { vi.resetAllMocks(); mocks.platform.mockReturnValue('linux'); mocks.exists.mockReturnValue(false); });

for (const [name, factory, create, version] of [['pnpm', pnpm, pnpmPlugin, '10.19.0'], ['yarn', yarn, yarnPlugin, '4.9.4']] as const) {
  function runtime(options = {}) {
    const instance = factory({ version, install_dir: path.resolve(`test-${name}`), ...options });
    return { instance, options: instance.options, context: { cwd: process.cwd(), env: {} as NodeJS.ProcessEnv, logger: { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() }, taskRegistry: {} as any } };
  }
  describe(name, () => {
    it('depends on configured Node by default', () => {
      const rt = runtime();
      expect(create(rt.instance).dependsOn).toEqual(['node']);
      const custom = runtime({ node_plugin: 'custom-node' });
      expect(create(custom.instance).dependsOn).toEqual(['custom-node']);
      const host = runtime({ node_plugin: null });
      expect(create(host.instance).dependsOn).toEqual([]);
    });
    it('skips an already matching managed install', async () => {
      mocks.run.mockResolvedValue({ code: 0, stdout: version, stderr: '' });
      mocks.exists.mockReturnValue(true);
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: false });
      expect(mocks.install).not.toHaveBeenCalled();
      expect(rt.context.env.PATH).toBe(path.join(rt.options.install_dir!, 'bin'));
    });
    it.each(['linux', 'macos', 'windows'])('installs a staged npm prefix and verifies shims on %s', async platform => {
      mocks.platform.mockReturnValue(platform);
      mocks.run.mockResolvedValue({ code: 0, stdout: version, stderr: '' }).mockResolvedValueOnce({ code: 1, stdout: '', stderr: 'missing' });
      mocks.install.mockImplementation(async config => {
        const prefix = await config.prepare(path.resolve('staged'));
        await config.verify(prefix);
        await config.verify(config.destination);
      });
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: true });
      expect(mocks.run).toHaveBeenCalledWith('npm', expect.arrayContaining(['--prefix', path.resolve('staged', 'prefix'), '--engine-strict', '--ignore-scripts', `${name === 'yarn' ? '@yarnpkg/cli-dist' : 'pnpm'}@${version}`]), expect.objectContaining({ cwd: process.cwd(), env: rt.context.env }));
      expect(rt.context.env.PATH).toBe(platform === 'windows' ? rt.options.install_dir : path.join(rt.options.install_dir!, 'bin'));
      expect(mocks.run.mock.calls[3][0]).toBe(path.resolve('staged', 'prefix', ...(platform === 'windows' ? [`${name}.cmd`] : ['bin', name])));
    });
    it('fails clearly if npm is unavailable', async () => {
      mocks.run.mockResolvedValue({ code: 1, stdout: '', stderr: 'missing' });
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: false, details: expect.stringContaining('npm is required') });
      expect(mocks.install).not.toHaveBeenCalled();
    });
    it('does not promote an incompatible or failed npm install', async () => {
      mocks.run.mockResolvedValueOnce({ code: 1, stdout: '', stderr: '' }).mockResolvedValueOnce({ code: 0, stdout: '10.9.0', stderr: '' }).mockResolvedValueOnce({ code: 1, stdout: '', stderr: 'EBADENGINE' });
      mocks.install.mockImplementation(async config => { await config.prepare(path.resolve('stage')); });
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining('EBADENGINE') });
    });
    it('rejects the wrong installed version during staging', async () => {
      mocks.run.mockResolvedValue({ code: 0, stdout: '1.0.0', stderr: '' });
      mocks.install.mockImplementation(async config => { await config.verify(path.resolve('stage')); });
      const rt = runtime();
      expect((await create(rt.instance).apply!(rt)).ok).toBe(false);
    });
    it('rejects unknown options and incomplete versions at load time', () => {
      const rt = runtime();
      expect(() => create(rt.instance).parseOptions!({ version: '10' })).toThrow('full version');
      expect(() => create(rt.instance).parseOptions!({ version, global_packages: ['typescript'] })).toThrow();
    });
  });
}

it('uses the classic package for Yarn 1', async () => {
  mocks.run.mockResolvedValue({ code: 0, stdout: '1.22.22', stderr: '' }).mockResolvedValueOnce({ code: 1, stdout: '', stderr: '' });
  mocks.install.mockImplementation(async config => { await config.prepare(path.resolve('stage')); });
  const instance = yarn({ version: '1.22.22', node_plugin: null });
  await yarnPlugin(instance).apply!({ instance, options: instance.options, context: { cwd: process.cwd(), env: {}, logger: { info: vi.fn() } as any, taskRegistry: {} as any } });
  expect(mocks.run).toHaveBeenCalledWith('npm', expect.arrayContaining(['yarn@1.22.22']), expect.anything());
});
