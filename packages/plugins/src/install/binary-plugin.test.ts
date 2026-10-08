import { beforeEach, describe, expect, it, vi } from "vitest";
import path from "node:path";
const mocks = vi.hoisted(() => ({ run: vi.fn(), platform: vi.fn(), exists: vi.fn(), install: vi.fn(), release: vi.fn(), mkdir: vi.fn(), rename: vi.fn(), chmod: vi.fn() }));
vi.mock("@ossl/genesis-core", () => ({ runCommand: mocks.run, getPlatform: mocks.platform }));
vi.mock("node:fs", () => ({ default: { existsSync: mocks.exists, promises: { mkdir: mocks.mkdir, rename: mocks.rename, chmod: mocks.chmod } } }));
vi.mock("./archive.js", () => ({ InstallationRecoveryError: class extends Error {}, installArchive: mocks.install, singleDirectory: async (root: string) => path.join(root, 'bun-archive'), prependPath: (env: NodeJS.ProcessEnv, dir: string) => { env.PATH = dir; } }));
vi.mock("./releases.js", () => ({ bunRelease: mocks.release, denoRelease: mocks.release }));
import { bun, createPlugin as bunPlugin } from "../plugins/bun/index.js";
import { deno, createPlugin as denoPlugin } from "../plugins/deno/index.js";

beforeEach(() => {
  vi.resetAllMocks();
  mocks.platform.mockReturnValue('linux');
  mocks.exists.mockReturnValue(false);
  mocks.release.mockResolvedValue({ url: 'https://example.com/runtime.zip', sha256: 'a'.repeat(64), format: 'zip' });
});
for (const [name, factory, create] of [['bun', bun, bunPlugin], ['deno', deno, denoPlugin]] as const) {
  const version = name === 'bun' ? '1.3.2' : '2.5.4';
  const output = name === 'bun' ? `${version}\n` : `deno ${version} (stable, release)\nv8 14.0.0\ntypescript 5.9.2\n`;
  const success = { code: 0, stdout: output, stderr: '' };
  function runtime() {
    const instance = factory({ version, install_dir: path.resolve(`test-${name}`) });
    return { instance, options: instance.options, context: { cwd: process.cwd(), env: {} as NodeJS.ProcessEnv, logger: { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() }, taskRegistry: {} as any } };
  }
  describe(name, () => {
    it('does not install an already matching runtime', async () => {
      mocks.run.mockResolvedValue(success);
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: false });
      expect(mocks.release).not.toHaveBeenCalled();
    });
    it.each(['linux', 'macos', 'windows'])('installs, verifies, and exposes the runtime on %s', async platform => {
      mocks.platform.mockReturnValue(platform);
      mocks.run.mockResolvedValue(success).mockResolvedValueOnce({ code: 1, stdout: '', stderr: 'missing' });
      mocks.install.mockImplementation(async config => {
        const selected = await config.select(path.resolve('unpack'));
        await config.verify(selected);
      });
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: true, didChange: true });
      expect(rt.context.env[`${name.toUpperCase()}_INSTALL`]).toBe(rt.options.install_dir);
      expect(rt.context.env.PATH).toBe(path.join(rt.options.install_dir!, 'bin'));
      expect(mocks.rename).toHaveBeenCalledWith(expect.stringContaining(platform === 'windows' ? `${name}.exe` : name), expect.stringContaining(`${path.sep}bin${path.sep}${name}`));
      expect(mocks.chmod).toHaveBeenCalledTimes(platform === 'windows' ? 0 : 1);
    });
    it('discovers its managed runtime on later invocations', async () => {
      mocks.exists.mockReturnValue(true); mocks.run.mockResolvedValue(success);
      const rt = runtime();
      expect((await create(rt.instance).detect!(rt)).ok).toBe(true);
      expect(mocks.run.mock.calls[0][0]).toBe(path.join(rt.options.install_dir!, 'bin', name));
    });
    it('rejects failed staged verification', async () => {
      mocks.run.mockResolvedValue({ code: 0, stdout: 'invalid version', stderr: '' });
      mocks.install.mockImplementation(async config => { await config.verify(path.resolve('stage')); });
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: false, details: expect.stringContaining('version could not be determined') });
    });
    it('reports unavailable releases', async () => {
      mocks.run.mockResolvedValue({ code: 1, stdout: '', stderr: '' }); mocks.release.mockRejectedValue(new Error('404'));
      const rt = runtime();
      expect(await create(rt.instance).apply!(rt)).toMatchObject({ ok: false, didChange: false, details: expect.stringContaining('404') });
    });
    it('validates options before installation', () => {
      const rt = runtime();
      expect(() => create(rt.instance).parseOptions!({ version: '2; echo hello' })).toThrow();
      expect(() => create(rt.instance).parseOptions!({ version: '2' })).toThrow('full version');
      expect(() => create(rt.instance).parseOptions!({ version, install_dir: '/' })).toThrow();
    });
  });
}
