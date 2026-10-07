import { describe, it, expect, vi, beforeEach } from 'vitest';

const { mockRunCommand, mockGetPlatform, mockCreatePkgUpdateTask, mockCreatePkgInstallTask, mockFsExistsSync, mockFsPromisesRm, mockFsPromisesUnlink, mockOsArch } = vi.hoisted(() => ({
  mockRunCommand: vi.fn(),
  mockGetPlatform: vi.fn(() => 'macos'),
  mockCreatePkgUpdateTask: vi.fn(),
  mockCreatePkgInstallTask: vi.fn(),
  mockFsExistsSync: vi.fn(() => false),
  mockFsPromisesRm: vi.fn(),
  mockFsPromisesUnlink: vi.fn(),
  mockOsArch: vi.fn(() => 'arm64'),
}));

vi.mock('@ossl/genesis-core', () => ({
  runCommand: mockRunCommand,
  getPlatform: mockGetPlatform,
  createPackageManagerUpdateTask: mockCreatePkgUpdateTask,
  createPackageInstallTask: mockCreatePkgInstallTask,
}));

vi.mock('node:os', () => ({ default: { arch: mockOsArch, tmpdir: () => '/tmp', homedir: () => '/home/testuser' }, arch: mockOsArch, tmpdir: () => '/tmp', homedir: () => '/home/testuser' }));

vi.mock('node:fs', () => ({ default: { existsSync: mockFsExistsSync, promises: { rm: mockFsPromisesRm, unlink: mockFsPromisesUnlink }, constants: { F_OK: 0 } }, existsSync: mockFsExistsSync, promises: { rm: mockFsPromisesRm, unlink: mockFsPromisesUnlink }, constants: { F_OK: 0 } }));

import { git, createPlugin } from './index.js';

// ── Test helpers ───────────────────────────────────────────────────
const mockLogger = { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() } as any;
const mockTaskRegistry = {
  register: vi.fn(),
  executeAll: vi.fn().mockResolvedValue(new Map()),
  has: vi.fn().mockReturnValue(false),
} as any;
const mockContext = { cwd: '/test', env: {}, logger: mockLogger, taskRegistry: mockTaskRegistry };

function makeRuntime(overrides: Record<string, unknown> = {}) {
  return {
    instance: { id: 'test-git', category: 'tool', module: 'git-plugin', options: {} },
    options: { version: 'latest', install_method: 'package', ...overrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('git plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
    mockOsArch.mockReturnValue('arm64');
    mockFsExistsSync.mockReturnValue(false);
  });

  // ── detect ──
  describe('detect', () => {
    it('returns ok:true when git --version succeeds', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'git version 2.43.0\n'));
      const plugin = createPlugin(git() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Git 2.43.0');
    });

    it('returns ok:false when git not found', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'git: command not found'));
      const plugin = createPlugin(git() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Git is not available on PATH');
    });

    it('returns ok:false when version is unparseable', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Some weird output\n'));
      const plugin = createPlugin(git() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Git version could not be determined');
    });

    it('returns ok:true for "latest" regardless of version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'git version 2.30.0\n'));
      const plugin = createPlugin(git() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('checks version match when specific version is requested', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'git version 2.43.0\n'));
      const plugin = createPlugin(git({ version: '2.43.0' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '2.43.0' }));
      expect(result.ok).toBe(true);
    });

    it('returns ok:false on specific version mismatch', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'git version 2.39.0\n'));
      const plugin = createPlugin(git({ version: '2.43.0' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '2.43.0' }));
      expect(result.ok).toBe(false);
      expect(result.details).toContain('Detected Git 2.39.0 but 2.43.0 is requested');
    });

    it('parses version from stderr when stdout empty', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'git version 2.43.0\n'));
      const plugin = createPlugin(git() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
    });
  });

  // ── apply: package ──
  describe('apply with install_method: package', () => {
    it('fails if prerequisites did not make Git available', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'missing'));
      const plugin = createPlugin(git());
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toContain('still unavailable');
      expect(mockRunCommand).toHaveBeenCalledTimes(1);
    });
  });

  // ── apply: source ──
  describe('apply with install_method: source', () => {
    it('clones and builds from source', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // git clone
      // macOS deps: gettext, openssl, zlib
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      // make all + sudo make install
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      // git config x2
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(git({ install_method: 'source' }) as any);
      const result = await plugin.apply!(makeRuntime({ install_method: 'source' }));
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
      expect(result.details).toContain('source');
    });

    it('handles clone failure', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'clone failed'));

      const plugin = createPlugin(git({ install_method: 'source' }) as any);
      const result = await plugin.apply!(makeRuntime({ install_method: 'source' }));
      expect(result.ok).toBe(false);
    });

    it('cleans up temp source directory before cloning', async () => {
      mockFsExistsSync.mockReturnValueOnce(true);
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(git({ install_method: 'source' }) as any);
      const result = await plugin.apply!(makeRuntime({ install_method: 'source' }));
      expect(result.ok).toBe(true);
      expect(mockFsExistsSync).toHaveBeenCalled();
      expect(mockFsPromisesRm).toHaveBeenCalledWith(
        expect.stringContaining('git-source'),
        { recursive: true, force: true },
      );
    });
  });

  // ── apply: binary ──
  describe('apply with install_method: binary', () => {
    it('downloads and extracts binary', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // curl
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // tar
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // config
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // config

      const plugin = createPlugin(git({ install_method: 'binary' }) as any);
      const result = await plugin.apply!(makeRuntime({ install_method: 'binary' }));
      expect(result.ok).toBe(true);
      expect(result.details).toContain('binary');
    });

    it('handles download failure', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'download failed'));

      const plugin = createPlugin(git({ install_method: 'binary' }) as any);
      const result = await plugin.apply!(makeRuntime({ install_method: 'binary' }));
      expect(result.ok).toBe(false);
    });

    it('uses correct binary URL for macOS arm64', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(git({ install_method: 'binary' }) as any);
      await plugin.apply!(makeRuntime({ install_method: 'binary' }));

      const curlCall = mockRunCommand.mock.calls.find((c: string[]) => c[0] === 'curl')!;
      expect(curlCall).toBeDefined();
      expect(curlCall[1].some((a: string) => a.includes('darwin-arm64'))).toBe(true);
    });

    it('uses correct binary URL for Linux x64', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockOsArch.mockReturnValue('x64');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(git({ install_method: 'binary' }) as any);
      await plugin.apply!(makeRuntime({ install_method: 'binary' }));

      const curlCall = mockRunCommand.mock.calls.find((c: string[]) => c[0] === 'curl')!;
      expect(curlCall).toBeDefined();
      expect(curlCall[1].some((a: string) => a.includes('linux-x86_64'))).toBe(true);
    });
  });

  // ── apply on Windows ──
  describe('apply on Windows', () => {
    it('prints manual guide', async () => {
      mockGetPlatform.mockReturnValue('windows');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));

      const plugin = createPlugin(git() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('not supported on Windows');
    });
  });

  // ── validate ──
  describe('validate', () => {
    it('returns ok:true when git is installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'git version 2.43.0\n'));
      const plugin = createPlugin(git() as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when git is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(git() as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  // ── registerTasks ──
  describe('registerTasks', () => {
    beforeEach(() => { mockRunCommand.mockResolvedValue(ok(1, '', 'missing')); });
    it('registers update and git for package method on macOS', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'git' });
      const plugin = createPlugin(git({ install_method: 'package' }) as any);
      await plugin.registerTasks!(makeRuntime({ install_method: 'package' }));
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('git', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(2);
    });

    it('registers git-all on Linux for package method', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(git({ install_method: 'package' }) as any);
      await plugin.registerTasks!(makeRuntime({ install_method: 'package' }));
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('git-all', '/test', {});
    });

    it('registers build tools for source method on macOS', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(git({ install_method: 'source' }) as any);
      await plugin.registerTasks!(makeRuntime({ install_method: 'source' }));
      // update + make + gcc + autoconf = 4
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(4);
    });

    it('registers build tools for source method on Linux', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(git({ install_method: 'source' }) as any);
      await plugin.registerTasks!(makeRuntime({ install_method: 'source' }));
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(4);
    });

    it('registers curl and tar for binary method', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(git({ install_method: 'binary' }) as any);
      await plugin.registerTasks!(makeRuntime({ install_method: 'binary' }));
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(3);
    });

    it('skips registration on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');
      const plugin = createPlugin(git() as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });
  });

  // ── factory ──
  describe('git factory', () => {
    it('sets default options correctly', () => {
      const inst = git();
      expect(inst.id).toBe('git');
      expect(inst.category).toBe('tool');
      expect(inst.module).toBe('@ossl/genesis-plugins/git');
      expect(inst.options.version).toBe('latest');
      expect(inst.options.install_method).toBe('package');
    });

    it('respects explicit options', () => {
      const inst = git({ version: '2.43.0', install_method: 'source' });
      expect(inst.options.version).toBe('2.43.0');
      expect(inst.options.install_method).toBe('source');
    });
  });
});


it('skips prerequisite registration when the desired runtime is already present', async () => {
  vi.clearAllMocks();
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValue(ok(0, 'git version 2.43.0'));
  const instance = git();
  const runtime = makeRuntime({ ...instance.options });
  await createPlugin(instance).registerTasks!(runtime);
  expect(mockTaskRegistry.register).not.toHaveBeenCalled();
});
