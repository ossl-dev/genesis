import { describe, it, expect, vi, beforeEach } from 'vitest';

// ── Mock variables ─────────────────────────────────────────────────
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
  default: { existsSync: mockFsExistsSync, promises: { rm: mockFsPromisesRm, unlink: mockFsPromisesUnlink }, constants: { F_OK: 0 } },
  existsSync: mockFsExistsSync,
  promises: { rm: mockFsPromisesRm, unlink: mockFsPromisesUnlink },
  constants: { F_OK: 0 },
}));

import { go, createPlugin } from './index.js';

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
    instance: { id: 'test-go', category: 'language', module: 'go-plugin', options: {} },
    options: { version: '1.22', ...overrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('go plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
    mockOsArch.mockReturnValue('arm64');
    mockFsExistsSync.mockReturnValue(false);
  });

  describe('detect', () => {
    it('returns ok:true when go version matches', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'go version go1.22.0 darwin/arm64\n'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Go 1.22.0');
    });

    it('returns ok:false when go is not installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'go: command not found'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Go is not available on PATH');
    });

    it('returns ok:false on version mismatch', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'go version go1.21.0 darwin/arm64\n'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toContain('Detected Go 1.21.0 but 1.22 is requested');
    });

    it('returns ok:false when version is unparseable', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'some random output\n'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Go version could not be determined');
    });

    it('parses version from stderr when stdout is empty', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'go version go1.22.0 darwin/arm64\n'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
    });
  });

  describe('apply', () => {
    it('returns didChange:false when Go is already installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'go version go1.22.0 darwin/arm64\n'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(false);
    });

    it('downloads and extracts on macOS (arm64)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // curl
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // tar

      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
      expect(result.details).toContain('installed to /usr/local/go');

      const curlCall = mockRunCommand.mock.calls.find((c: string[]) => c[0] === 'curl')!;
      expect(curlCall).toBeDefined();
      expect(curlCall[1].some((a: string) => a.includes('darwin-arm64'))).toBe(true);
    });

    it('downloads and extracts on Linux (x64)', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockOsArch.mockReturnValue('x64');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);

      const curlCall = mockRunCommand.mock.calls.find((c: string[]) => c[0] === 'curl')!;
      expect(curlCall).toBeDefined();
      expect(curlCall[1].some((a: string) => a.includes('linux-amd64'))).toBe(true);
    });

    it('removes previous Go installation before extracting', async () => {
      mockFsExistsSync.mockReturnValueOnce(true);
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(mockFsExistsSync).toHaveBeenCalledWith('/usr/local/go');
      expect(mockFsPromisesRm).toHaveBeenCalledWith('/usr/local/go', { recursive: true, force: true });
    });

    it('returns failure when download fails', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'download error'));

      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
    });

    it('on Windows prints manual guide', async () => {
      mockGetPlatform.mockReturnValue('windows');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('not supported on Windows');
    });
  });

  describe('validate', () => {
    it('returns ok:true when Go is present', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'go version go1.22.0 darwin/arm64\n'));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when Go is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  describe('registerTasks', () => {
    beforeEach(() => { mockRunCommand.mockResolvedValue(ok(1, '', 'missing')); });
    it('registers update, curl, and tar tasks on macOS', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('curl', '/test', {});
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('tar', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(3);
    });

    it('skips registration on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');
      const plugin = createPlugin(go({ version: '1.22' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });
  });

  describe('go factory', () => {
    it('sets id, category, and module correctly', () => {
      const inst = go({ version: '1.22' });
      expect(inst.id).toBe('go');
      expect(inst.category).toBe('language');
      expect(inst.module).toBe('@ossl/genesis-plugins/go');
    });
  });
});


it('skips prerequisite registration when the desired runtime is already present', async () => {
  vi.clearAllMocks();
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValue(ok(0, 'go version go1.22.0 darwin/arm64'));
  const instance = go({ version: "1.22.0" });
  const runtime = makeRuntime({ ...instance.options });
  await createPlugin(instance).registerTasks!(runtime);
  expect(mockTaskRegistry.register).not.toHaveBeenCalled();
});
