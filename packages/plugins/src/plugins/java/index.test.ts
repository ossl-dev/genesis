import { describe, it, expect, vi, beforeEach } from 'vitest';

// ── Mock variables ─────────────────────────────────────────────────
const { mockRunCommand, mockGetPlatform, mockCreatePkgUpdateTask, mockCreatePkgInstallTask, mockFsPromisesMkdir, mockFsPromisesReaddir, mockFsPromisesRename, mockFsPromisesUnlink, mockOsArch } = vi.hoisted(() => ({
  mockRunCommand: vi.fn(),
  mockGetPlatform: vi.fn(() => 'macos'),
  mockCreatePkgUpdateTask: vi.fn(),
  mockCreatePkgInstallTask: vi.fn(),
  mockFsPromisesMkdir: vi.fn(),
  mockFsPromisesReaddir: vi.fn(),
  mockFsPromisesRename: vi.fn(),
  mockFsPromisesUnlink: vi.fn(),
  mockOsArch: vi.fn(() => 'arm64'),
}));

// ── Module mocks ───────────────────────────────────────────────────
vi.mock('@ossl/genesis-core', () => ({
  runCommand: mockRunCommand,
  getPlatform: mockGetPlatform,
  createPackageManagerUpdateTask: mockCreatePkgUpdateTask,
  createPackageInstallTask: mockCreatePkgInstallTask,
}));

vi.mock('node:os', () => ({
  default: { arch: mockOsArch, tmpdir: () => '/tmp', homedir: () => '/home/testuser' },
  arch: mockOsArch,
  tmpdir: () => '/tmp',
  homedir: () => '/home/testuser',
}));

vi.mock('node:fs', () => ({
  default: {
    promises: { mkdir: mockFsPromisesMkdir, readdir: mockFsPromisesReaddir, rename: mockFsPromisesRename, unlink: mockFsPromisesUnlink },
    constants: { F_OK: 0 },
  },
  promises: { mkdir: mockFsPromisesMkdir, readdir: mockFsPromisesReaddir, rename: mockFsPromisesRename, unlink: mockFsPromisesUnlink },
  constants: { F_OK: 0 },
}));

import { java, createPlugin } from './index.js';

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
    instance: { id: 'test-java', category: 'language', module: 'java-plugin', options: {} },
    options: { version: '17', distribution: 'openjdk', ...overrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('java plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
    mockOsArch.mockReturnValue('arm64');
  });

  describe('detect', () => {
    it('returns ok:true when java -version matches major (Java 11+ style)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'openjdk version "17.0.9" 2023-10-17\n'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Java 17.0.9');
    });

    it('returns ok:true for Java 8 style version "1.8.0_202"', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'java version "1.8.0_202"\n'));
      const plugin = createPlugin(java({ version: '1.8' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '1.8' }));
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when Java not installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'java: command not found'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Java is not available on PATH');
    });

    it('returns ok:false on major version mismatch', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'openjdk version "11.0.21" 2023-10-17\n'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toContain('major 11');
    });

    it('returns ok:false when version is unparseable', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'Some weird output\n'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Java version could not be determined');
    });

    it('checks stdout as fallback when stderr empty', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'java version "17.0.9"\n', ''));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('matches Java 8 major when requested version is "8"', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'java version "1.8.0_202"\n'));
      const plugin = createPlugin(java({ version: '8' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '8' }));
      expect(result.ok).toBe(true);
    });

    it('fallback parsing for alt version format', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'OpenJDK Runtime Environment (build 17.0.9+9)\n'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
    });
  });

  describe('apply', () => {
    it('returns didChange:false when Java already installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'openjdk version "17.0.9"\n'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(false);
    });

    it('installs OpenJDK from Adoptium on macOS (arm64)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // curl
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // tar
      mockFsPromisesReaddir.mockResolvedValueOnce(['jdk-17.0.9+7']);

      const plugin = createPlugin(java({ version: '17.0.9' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);

      const curlCall = mockRunCommand.mock.calls.find((c: string[]) => c[0] === 'curl')!;
      expect(curlCall).toBeDefined();
      expect(curlCall[1].some((a: string) => a.includes('aarch64'))).toBe(true);
    });

    it('installs OpenJDK on Linux (x64)', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockOsArch.mockReturnValue('x64');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockFsPromisesReaddir.mockResolvedValueOnce(['jdk-17.0.9+7']);

      const plugin = createPlugin(java({ version: '17.0.9' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);

      const curlCall = mockRunCommand.mock.calls.find((c: string[]) => c[0] === 'curl')!;
      expect(curlCall).toBeDefined();
      expect(curlCall[1].some((a: string) => a.includes('linux'))).toBe(true);
    });

    it('with oracle distribution prints manual download guide', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(java({ version: '17', distribution: 'oracle' }) as any);
      const result = await plugin.apply!(makeRuntime({ distribution: 'oracle' }));
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('manual download');
    });

    it('on Windows prints guide', async () => {
      mockGetPlatform.mockReturnValue('windows');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('Manual installation required');
    });

    it('returns failure when curl download fails', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'download error'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
    });
  });

  describe('validate', () => {
    it('returns ok:true when Java is at correct version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'openjdk version "17.0.9"\n'));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when Java is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(java({ version: '17' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  describe('registerTasks', () => {
    beforeEach(() => { mockRunCommand.mockResolvedValue(ok(1, '', 'missing')); });
    it('registers update, curl, and tar on macOS/Linux', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(java({ version: '17' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('curl', '/test', {});
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('tar', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(3);
    });

    it('skips registration on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');
      const plugin = createPlugin(java({ version: '17' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });
  });

  describe('java factory', () => {
    it('sets id, category, module, and default distribution', () => {
      const inst = java({ version: '17' });
      expect(inst.id).toBe('java');
      expect(inst.category).toBe('language');
      expect(inst.module).toBe('@ossl/genesis-plugins/java');
      expect(inst.options.distribution).toBe('openjdk');
    });

    it('respects explicit distribution option', () => {
      const inst = java({ version: '17', distribution: 'oracle' });
      expect(inst.options.distribution).toBe('oracle');
    });
  });
});


it('skips prerequisite registration when the desired runtime is already present', async () => {
  vi.clearAllMocks();
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValue(ok(0, 'openjdk version "17.0.9"'));
  const instance = java({ version: "17" });
  const runtime = makeRuntime({ ...instance.options });
  await createPlugin(instance).registerTasks!(runtime);
  expect(mockTaskRegistry.register).not.toHaveBeenCalled();
});
