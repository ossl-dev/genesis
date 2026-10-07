import { describe, it, expect, vi, beforeEach } from 'vitest';

// ── Mock variables ─────────────────────────────────────────────────
const { mockRunCommand, mockGetPlatform, mockCreatePkgUpdateTask, mockCreatePkgInstallTask, mockCreateCmdCheckTask, mockCreateCustomTask, mockFsPromisesAccess, mockOsHomedir, mockOsArch, mockOsTmpdir } = vi.hoisted(() => ({
  mockRunCommand: vi.fn(),
  mockGetPlatform: vi.fn(() => 'macos'),
  mockCreatePkgUpdateTask: vi.fn(),
  mockCreatePkgInstallTask: vi.fn(),
  mockCreateCmdCheckTask: vi.fn(),
  mockCreateCustomTask: vi.fn(),
  mockFsPromisesAccess: vi.fn(),
  mockOsHomedir: vi.fn(() => '/home/testuser'),
  mockOsArch: vi.fn(() => 'arm64'),
  mockOsTmpdir: vi.fn(() => '/tmp'),
}));

// ── Module mocks ───────────────────────────────────────────────────
vi.mock('@ossl/genesis-core', () => ({
  runCommand: mockRunCommand,
  getPlatform: mockGetPlatform,
  createPackageManagerUpdateTask: mockCreatePkgUpdateTask,
  createPackageInstallTask: mockCreatePkgInstallTask,
  createCommandCheckTask: mockCreateCmdCheckTask,
  createCustomTask: mockCreateCustomTask,
}));

vi.mock('node:os', () => ({
  default: { homedir: mockOsHomedir, arch: mockOsArch, tmpdir: mockOsTmpdir },
  homedir: mockOsHomedir,
  arch: mockOsArch,
  tmpdir: mockOsTmpdir,
}));

vi.mock('node:fs', () => ({
  default: { promises: { access: mockFsPromisesAccess, rm: vi.fn() }, existsSync: vi.fn(() => false), constants: { F_OK: 0 } },
  promises: { access: mockFsPromisesAccess, rm: vi.fn() },
  existsSync: vi.fn(() => false),
  constants: { F_OK: 0 },
}));

import { node, createPlugin } from './index.js';

// ── Test helpers ───────────────────────────────────────────────────
const mockLogger = { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() } as any;
const mockTaskRegistry = {
  register: vi.fn(),
  executeAll: vi.fn().mockResolvedValue(new Map()),
  has: vi.fn().mockReturnValue(false),
} as any;
const mockContext = { cwd: '/test', env: {}, logger: mockLogger, taskRegistry: mockTaskRegistry };

function makeRuntime(optionsOverrides: Record<string, unknown> = {}) {
  return {
    instance: { id: 'test-node', category: 'tool', module: 'node-plugin', options: {} },
    options: { version: '20', use_nvm: true, ...optionsOverrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('node plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
    mockContext.env = {};
  });

  // ── detect ──
  describe('detect', () => {
    it('returns ok:true when node --version succeeds and matches requested version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'v20.11.0\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Node 20.11.0');
    });

    it('returns ok:false when node is not on PATH', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'node: command not found'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Node is not available on PATH');
    });

    it('returns ok:false when version output is empty', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Node version could not be determined');
    });

    it('returns ok:false on major version mismatch', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'v18.17.0\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toContain('Detected Node 18.17.0 but 20 is requested');
    });

    it('handles version output without "v" prefix', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, '20.11.0\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Node 20.11.0');
    });

    it('returns ok:false when version does not start with requested version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'v22.0.0\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  // ── apply: use_nvm on macOS ──
  describe('apply with use_nvm on macOS', () => {
    it('returns didChange:false when Node is already installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'v20.11.0\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(false);
    });

    it('installs NVM then Node when neither is present', async () => {
      // detect fails
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      // isNvmInstalled: access throws
      mockFsPromisesAccess.mockRejectedValueOnce(new Error('ENOENT'));
      // installNvm: curl | bash succeeds
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      // installNodeViaNvm: nvm install succeeds
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      // nvm alias default succeeds
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      mockRunCommand.mockResolvedValueOnce(ok(0, '/home/testuser/.nvm/versions/node/v20.11.0/bin/node'));
      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
      expect(result.details).toContain('installed via NVM');
    });

    it('skips NVM install when already installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      // isNvmInstalled: access succeeds
      mockFsPromisesAccess.mockResolvedValueOnce(undefined);
      // installNodeViaNvm
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      mockRunCommand.mockResolvedValueOnce(ok(0, '/home/testuser/.nvm/versions/node/v20.11.0/bin/node'));
      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(true);
      expect(mockLogger.debug).toHaveBeenCalledWith('NVM is already installed');
    });

    it('returns failure when NVM install script fails', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockFsPromisesAccess.mockRejectedValueOnce(new Error('ENOENT'));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'install failed'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
    });

    it('returns didChange:true when NVM installs but Node install fails', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockFsPromisesAccess.mockRejectedValueOnce(new Error('ENOENT'));
      // installNvm ok
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      // installNodeViaNvm fails
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'nvm install error'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(true); // NVM was installed
    });
  });

  // ── apply: use_nvm on Linux ──
  describe('apply with use_nvm on Linux', () => {
    beforeEach(() => {
      mockGetPlatform.mockReturnValue('linux');
    });

    it('installs Node via NVM on Linux', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockFsPromisesAccess.mockRejectedValueOnce(new Error('ENOENT'));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      mockRunCommand.mockResolvedValueOnce(ok(0, '/home/testuser/.nvm/versions/node/v20.11.0/bin/node'));
      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
    });
  });

  // ── apply: use_nvm:false ──
  describe('apply with use_nvm:false on macOS', () => {
    it('returns standalone-not-supported message', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));

      const plugin = createPlugin(node({ version: '20', use_nvm: false }) as any);
      const result = await plugin.apply!(makeRuntime({ use_nvm: false }));

      expect(result.ok).toBe(false);
      expect(result.details).toContain('Standalone installation not supported');
    });
  });

  // ── apply on Windows ──
  describe('apply on Windows', () => {
    beforeEach(() => {
      mockGetPlatform.mockReturnValue('windows');
    });

    it('prints nvm guide when use_nvm is true', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.apply!(makeRuntime());

      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('not supported on Windows');
    });

    it('prints standalone guide when use_nvm is false', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));

      const plugin = createPlugin(node({ version: '20', use_nvm: false }) as any);
      const result = await plugin.apply!(makeRuntime({ use_nvm: false }));

      expect(result.ok).toBe(false);
      expect(result.details).toContain('Manual installation required');
    });
  });

  // ── validate ──
  describe('validate', () => {
    it('returns ok:true when Node is at correct version', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'v20.11.0\n'));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.message).toContain('Detected Node 20.11.0');
    });

    it('returns ok:false when Node is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));

      const plugin = createPlugin(node({ version: '20' }) as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  // ── registerTasks ──
  describe('registerTasks', () => {
    beforeEach(() => { mockRunCommand.mockResolvedValue(ok(1, '', 'missing')); });
    it('registers update and curl tasks on macOS with use_nvm', async () => {
      const updateTask = { id: 'macos:package-manager:brew-update', description: 'update' };
      const curlTask = { id: 'macos:package-manager:brew-install:curl', description: 'curl' };
      mockCreatePkgUpdateTask.mockReturnValue(updateTask);
      mockCreatePkgInstallTask.mockReturnValue(curlTask);

      const plugin = createPlugin(node({ version: '20' }) as any);
      await plugin.registerTasks!(makeRuntime());

      expect(mockCreatePkgUpdateTask).toHaveBeenCalled();
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('curl', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledWith(updateTask);
      expect(mockTaskRegistry.register).toHaveBeenCalledWith(curlTask);
    });

    it('skips registration on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');

      const plugin = createPlugin(node({ version: '20' }) as any);
      await plugin.registerTasks!(makeRuntime());

      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });

    it('keeps global packages out of the prerequisite phase', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'curl' });
      mockCreateCmdCheckTask.mockReturnValue({ id: 'check-npm' });
      mockCreateCustomTask.mockReturnValue({ id: 'custom-global-packages' });

      const plugin = createPlugin(node({ version: '20', global_packages: ['typescript'] }) as any);
      await plugin.registerTasks!(makeRuntime({ global_packages: ['typescript'] }));

      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(2);
      expect(mockCreateCustomTask).not.toHaveBeenCalled();
    });

    it('does not register global package tasks when list is empty', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'curl' });

      const plugin = createPlugin(node({ version: '20', global_packages: [] }) as any);
      await plugin.registerTasks!(makeRuntime({ global_packages: [] }));

      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(2);
    });

    it('does not register global package tasks when undefined', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'curl' });

      const plugin = createPlugin(node({ version: '20' }) as any);
      await plugin.registerTasks!(makeRuntime());

      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(2);
    });
  });

  // ── factory function ──
  describe('node factory', () => {
    it('sets default options correctly', () => {
      const inst = node({ version: '20' });
      expect(inst.id).toBe('node');
      expect(inst.category).toBe('tool');
      expect(inst.module).toBe('@ossl/genesis-plugins/node');
      expect(inst.options.use_nvm).toBe(true);
      expect(inst.options.version).toBe('20');
    });

    it('respects explicit use_nvm: false', () => {
      const inst = node({ version: '20', use_nvm: false });
      expect(inst.options.use_nvm).toBe(false);
    });
  });
});


it('installs global packages through NVM after Node is ready', async () => {
  mockFsPromisesAccess.mockResolvedValue(undefined);
  mockRunCommand.mockResolvedValue({ code: 0, stdout: '', stderr: '' });
  const plugin = createPlugin(node({ version: '20', global_packages: ['typescript'] }));
  await plugin.postApply!(makeRuntime({ global_packages: ['typescript'] }));
  expect(mockRunCommand).toHaveBeenCalledWith('bash', expect.arrayContaining(['genesis-npm', '/home/testuser/.nvm/nvm.sh', '20', 'typescript']), expect.anything());
});


it('skips prerequisite registration when the desired runtime is already present', async () => {
  vi.clearAllMocks();
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValue(ok(0, 'v20.11.0'));
  const instance = node({ version: "20" });
  const runtime = makeRuntime({ ...instance.options });
  await createPlugin(instance).registerTasks!(runtime);
  expect(mockTaskRegistry.register).not.toHaveBeenCalled();
});
