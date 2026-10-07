import { describe, it, expect, vi, beforeEach } from 'vitest';

// ── Mock variables ─────────────────────────────────────────────────
const { mockRunCommand, mockGetPlatform, mockCreatePkgUpdateTask, mockCreatePkgInstallTask } = vi.hoisted(() => ({
  mockRunCommand: vi.fn(),
  mockGetPlatform: vi.fn(() => 'macos'),
  mockCreatePkgUpdateTask: vi.fn(),
  mockCreatePkgInstallTask: vi.fn(),
}));

// ── Module mocks ───────────────────────────────────────────────────
vi.mock('@ossl/genesis-core', () => ({
  runCommand: mockRunCommand,
  getPlatform: mockGetPlatform,
  createPackageManagerUpdateTask: mockCreatePkgUpdateTask,
  createPackageInstallTask: mockCreatePkgInstallTask,
}));

import { python, createPlugin } from './index.js';

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
    instance: { id: 'test-python', category: 'language', module: 'python-plugin', options: {} },
    options: { version: '3.12', ...overrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('python plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
  });

  describe('detect', () => {
    it('returns ok:true when python3 --version matches requested minor version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Python 3.12.3\n'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Python 3.12.3');
    });

    it('returns ok:false when python3 is not found', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'command not found: python3'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Python is not available on PATH');
    });

    it('returns ok:false when version output is unparseable', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Some weird output\n'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Python version could not be determined');
    });

    it('returns ok:false on minor version mismatch', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Python 3.11.8\n'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toContain('Detected Python 3.11.8 but 3.12 is requested');
    });

    it('parses version from stderr when stdout is empty', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '', 'Python 3.12.3\n'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('rejects a mismatched explicitly requested patch version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Python 3.12.0\n'));
      const plugin = createPlugin(python({ version: '3.12.5' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '3.12.5' }));
      expect(result.ok).toBe(false);
    });
  });

  describe('apply', () => {
    it('returns didChange:false when already installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Python 3.12.3\n'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(false);
    });

    it('on macOS reports installation via system package manager', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('still unavailable');
    });

    it('on Linux reports installation via system package manager', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
    });

    it('on Windows prints manual install guide', async () => {
      mockGetPlatform.mockReturnValue('windows');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('Manual installation required');
    });
  });

  describe('validate', () => {
    it('returns ok:true when Python is present at correct version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Python 3.12.3\n'));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when Python is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  describe('registerTasks', () => {
    beforeEach(() => { mockRunCommand.mockResolvedValue(ok(1, '', 'missing')); });
    it('registers update and python@X tasks on macOS', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'python@3.12' });
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('python@3.12', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(2);
    });

    it('registers update and pythonX.Y tasks on Linux', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'python3.12' });
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('python3.12', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(2);
    });

    it('skips registration on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');
      const plugin = createPlugin(python({ version: '3.12' }) as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });
  });

  describe('python factory', () => {
    it('sets id, category, and module correctly', () => {
      const inst = python({ version: '3.12' });
      expect(inst.id).toBe('python');
      expect(inst.category).toBe('language');
      expect(inst.module).toBe('@ossl/genesis-plugins/python');
      expect(inst.options.version).toBe('3.12');
    });
  });
});


it('skips prerequisite registration when the desired runtime is already present', async () => {
  vi.clearAllMocks();
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValue(ok(0, 'Python 3.12.3'));
  const instance = python({ version: "3.12" });
  const runtime = makeRuntime({ ...instance.options });
  await createPlugin(instance).registerTasks!(runtime);
  expect(mockTaskRegistry.register).not.toHaveBeenCalled();
});
