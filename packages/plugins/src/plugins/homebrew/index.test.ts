import { describe, it, expect, vi, beforeEach } from 'vitest';

// ── Mock variables ─────────────────────────────────────────────────
const { mockRunCommand, mockGetPlatform, mockCreatePkgUpdateTask, mockCreatePkgInstallTask, mockCreateCmdCheckTask, mockCreateCustomTask, mockOsArch } = vi.hoisted(() => ({
  mockRunCommand: vi.fn(),
  mockGetPlatform: vi.fn(() => 'macos'),
  mockCreatePkgUpdateTask: vi.fn(),
  mockCreatePkgInstallTask: vi.fn(),
  mockCreateCmdCheckTask: vi.fn(),
  mockCreateCustomTask: vi.fn(),
  mockOsArch: vi.fn(() => 'arm64'),
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
  default: { arch: mockOsArch, tmpdir: () => '/tmp', homedir: () => '/home/testuser' },
  arch: mockOsArch,
  tmpdir: () => '/tmp',
  homedir: () => '/home/testuser',
}));

vi.mock('node:fs', () => ({
  default: { promises: {}, constants: { F_OK: 0 } },
  promises: {},
  constants: { F_OK: 0 },
}));

import { homebrew, createPlugin } from './index.js';

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
    instance: { id: 'test-homebrew', category: 'tool', module: 'homebrew-plugin', options: {} },
    options: { update_packages: true, install_cask: true, add_to_path: true, ...overrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('homebrew plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
    mockOsArch.mockReturnValue('arm64');
  });

  describe('detect', () => {
    it('returns ok:true when brew --version succeeds', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n'));
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Homebrew 4.3.0');
    });

    it('returns ok:false when brew is not found', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'brew: command not found'));
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Homebrew is not available on PATH');
    });

    it('returns ok:false when version is unparseable', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Something strange\n'));
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Homebrew version could not be determined');
    });
  });

  describe('apply', () => {
    it('returns didChange:true when installed and update succeeds', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew update
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew upgrade
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew upgrade --cask

      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
    });

    it('installs Homebrew on macOS (Apple Silicon, arm64)', async () => {
      mockOsArch.mockReturnValue('arm64');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // curl | bash
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n')); // test
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew update
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew upgrade
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew upgrade --cask

      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
      expect(result.details).toContain('installed to /opt/homebrew');
    });

    it('installs Homebrew on macOS (Intel, x64)', async () => {
      mockOsArch.mockReturnValue('x64');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));

      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('installed to /usr/local');
    });

    it('prints unsupported platform on Linux', async () => {
      mockGetPlatform.mockReturnValue('linux');
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('not supported on linux');
    });

    it('prints unsupported platform on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toContain('not supported on windows');
    });

    it('runs brew upgrade when update_packages is true', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // upgrade

      const plugin = createPlugin(homebrew({ update_packages: true, install_cask: false }) as any);
      const result = await plugin.apply!(makeRuntime({ update_packages: true, install_cask: false }));
      expect(result.ok).toBe(true);
      expect(mockRunCommand).toHaveBeenCalledWith('brew', ['upgrade'], expect.any(Object));
    });

    it('runs brew upgrade --cask when install_cask is true', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // --cask

      const plugin = createPlugin(homebrew({ update_packages: true, install_cask: true }) as any);
      const result = await plugin.apply!(makeRuntime({ update_packages: true, install_cask: true }));
      expect(result.ok).toBe(true);
      expect(mockRunCommand).toHaveBeenCalledWith('brew', ['upgrade', '--cask'], expect.any(Object));
    });

    it('handles install failure gracefully', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'install error'));

      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
    });
  });

  describe('validate', () => {
    it('returns ok:true when brew is installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0\n'));
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when brew is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(homebrew() as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  describe('global packages', () => {
    it('preserves packages in the factory and installs them after apply', async () => {
      const instance = homebrew({ global_packages: ['wget'] });
      expect(instance.options?.global_packages).toEqual(['wget']);
      const plugin = createPlugin(instance);
      mockRunCommand.mockResolvedValue({ code: 0, stdout: '', stderr: '' });
      await plugin.postApply!(makeRuntime({ global_packages: ['wget'] }));
      expect(mockRunCommand).toHaveBeenCalledWith('brew', ['install', 'wget'], expect.anything());
      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });
  });

  describe('homebrew factory', () => {
    it('sets default options correctly', () => {
      const inst = homebrew();
      expect(inst.id).toBe('homebrew');
      expect(inst.category).toBe('tool');
      expect(inst.module).toBe('@ossl/genesis-plugins/homebrew');
      expect(inst.options.update_packages).toBe(true);
      expect(inst.options.install_cask).toBe(true);
      expect(inst.options.add_to_path).toBe(true);
    });

    it('respects explicit options', () => {
      const inst = homebrew({ update_packages: false, install_cask: false, add_to_path: false });
      expect(inst.options.update_packages).toBe(false);
      expect(inst.options.install_cask).toBe(false);
      expect(inst.options.add_to_path).toBe(false);
    });
  });
});


it.each([['update', 1], ['upgrade', 2], ['cask', 3]])('reports a failed %s instead of success', async (_, failingCall) => {
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0'));
  for (let i = 1; i < failingCall; i++) mockRunCommand.mockResolvedValueOnce(ok(0, ''));
  mockRunCommand.mockResolvedValueOnce(ok(1, '', 'network failed'));
  const result = await createPlugin(homebrew()).apply!(makeRuntime());
  expect(result).toMatchObject({ ok: false, didChange: true });
  expect(result.details).toContain('network failed');
  expect(mockRunCommand).toHaveBeenCalledTimes(failingCall + 1);
});

it('bootstraps and verifies brew before shared tasks register', async () => {
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockOsArch.mockReturnValue('arm64');
  mockContext.env = {};
  mockRunCommand.mockResolvedValueOnce(ok(1, '')).mockResolvedValueOnce(ok(0, '')).mockResolvedValueOnce(ok(0, 'Homebrew 4.3.0'));
  await createPlugin(homebrew()).prepare!(makeRuntime());
  expect(mockContext.env).toHaveProperty('PATH', expect.stringContaining('/opt/homebrew/bin'));
  expect(mockRunCommand.mock.calls.map(call => call[0])).toEqual(['brew', 'bash', 'brew']);
});

it('stops preparation when an installed brew cannot be verified', async () => {
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('macos');
  mockRunCommand.mockResolvedValueOnce(ok(1, '')).mockResolvedValueOnce(ok(0, '')).mockResolvedValueOnce(ok(1, ''));
  await expect(createPlugin(homebrew()).prepare!(makeRuntime())).rejects.toThrow('failed verification');
});
