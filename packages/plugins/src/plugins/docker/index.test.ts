import { describe, it, expect, vi, beforeEach } from 'vitest';

// ── Mock variables ─────────────────────────────────────────────────
const { mockRunCommand, mockGetPlatform, mockCreatePkgUpdateTask, mockCreatePkgInstallTask, mockFsReadFile, mockFsPromisesUnlink } = vi.hoisted(() => ({
  mockRunCommand: vi.fn(),
  mockGetPlatform: vi.fn(() => 'macos'),
  mockCreatePkgUpdateTask: vi.fn(),
  mockCreatePkgInstallTask: vi.fn(),
  mockFsReadFile: vi.fn(),
  mockFsPromisesUnlink: vi.fn(),
}));

// ── Module mocks ───────────────────────────────────────────────────
vi.mock('@ossl/genesis-core', () => ({
  runCommand: mockRunCommand,
  getPlatform: mockGetPlatform,
  createPackageManagerUpdateTask: mockCreatePkgUpdateTask,
  createPackageInstallTask: mockCreatePkgInstallTask,
}));

vi.mock('node:os', () => ({
  default: { arch: () => 'arm64', tmpdir: () => '/tmp', homedir: () => '/home/testuser' },
  arch: () => 'arm64',
  tmpdir: () => '/tmp',
  homedir: () => '/home/testuser',
}));

vi.mock('node:fs', () => ({
  default: { promises: { readFile: mockFsReadFile, unlink: mockFsPromisesUnlink }, constants: { F_OK: 0 } },
  promises: { readFile: mockFsReadFile, unlink: mockFsPromisesUnlink },
  constants: { F_OK: 0 },
}));

import { docker, createPlugin } from './index.js';

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
    instance: { id: 'test-docker', category: 'tool', module: 'docker-plugin', options: {} },
    options: { version: 'latest', include_compose: true, install_desktop: false, ...overrides },
    context: mockContext,
  };
}

function ok(code: number, stdout: string, stderr = '') {
  return { code, stdout, stderr };
}

// ── Tests ──────────────────────────────────────────────────────────
describe('docker plugin', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetPlatform.mockReturnValue('macos');
  });

  describe('detect', () => {
    it('returns ok:true when docker --version succeeds (with compose)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0, build abc123\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker Compose version v2.27.0\n'));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Detected Docker 26.1.0');
      expect(result.details).toContain('Docker Compose 2.27.0');
    });

    it('returns ok:false when docker not installed', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', 'docker: command not found'));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Docker is not available on PATH');
    });

    it('returns ok:false when version is unparseable', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Something else\n'));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.detect!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.details).toBe('Docker version could not be determined');
    });

    it('checks specific version match (with compose)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0, build abc123\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker Compose version v2.27.0\n'));
      const plugin = createPlugin(docker({ version: '26.1.0' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '26.1.0' }));
      expect(result.ok).toBe(true);
    });

    it('returns ok:false on specific version mismatch', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 25.0.0\n'));
      const plugin = createPlugin(docker({ version: '26.1.0' }) as any);
      const result = await plugin.detect!(makeRuntime({ version: '26.1.0' }));
      expect(result.ok).toBe(false);
    });

    it('with include_compose checks both docker and docker-compose (both present)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker Compose version v2.27.0\n'));
      const plugin = createPlugin(docker({ include_compose: true }) as any);
      const result = await plugin.detect!(makeRuntime({ include_compose: true }));
      expect(result.ok).toBe(true);
      expect(result.details).toContain('Docker 26.1.0');
      expect(result.details).toContain('Docker Compose 2.27.0');
    });

    it('with include_compose fails when compose is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(docker({ include_compose: true }) as any);
      const result = await plugin.detect!(makeRuntime({ include_compose: true }));
      expect(result.ok).toBe(false);
    });

    it('without include_compose only checks docker', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0\n'));
      const plugin = createPlugin(docker({ include_compose: false }) as any);
      const result = await plugin.detect!(makeRuntime({ include_compose: false }));
      expect(result.ok).toBe(true);
      expect(mockRunCommand).toHaveBeenCalledTimes(1);
    });
  });

  describe('apply', () => {
    it('returns didChange:false when Docker is already installed (with compose)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker Compose version v2.27.0\n'));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(false);
    });

    it('on macOS without install_desktop installs via Colima', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // brew install colima
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // colima start
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // docker run hello-world test

      const plugin = createPlugin(docker() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(true);
      expect(result.didChange).toBe(true);
      expect(result.details).toContain('Colima');
    });

    it('on macOS with install_desktop downloads Docker Desktop DMG', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // curl download
      mockRunCommand.mockResolvedValueOnce(ok(0, '', '')); // docker run hello-world test

      const plugin = createPlugin(docker({ install_desktop: true }) as any);
      const result = await plugin.apply!(makeRuntime({ install_desktop: true }));
      expect(result.ok).toBe(false);
      expect(result.details).toContain('Complete installation');
      expect(mockFsPromisesUnlink).not.toHaveBeenCalled();
    });

    it('on Linux (Ubuntu) installs Docker Engine', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockFsReadFile.mockResolvedValueOnce('ID=ubuntu\nVERSION_CODENAME=jammy\n');
      // All install commands succeed
      mockRunCommand.mockResolvedValue(ok(0, '', ''));

      const plugin = createPlugin(docker() as any);
      await plugin.apply!(makeRuntime());
      // It should run multiple commands (detect + install + test)
      expect(mockRunCommand).toHaveBeenCalled();
    });

    it('handles unknown Linux distro gracefully', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      mockFsReadFile.mockResolvedValueOnce('ID=arch\n');

      const plugin = createPlugin(docker() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
    });

    it('on Windows prints manual guide', async () => {
      mockGetPlatform.mockReturnValue('windows');
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.apply!(makeRuntime());
      expect(result.ok).toBe(false);
      expect(result.didChange).toBe(false);
      expect(result.details).toContain('not supported on Windows');
    });
  });

  describe('validate', () => {
    it('returns ok:true when Docker is installed (with compose)', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0\n'));
      mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker Compose version v2.27.0\n'));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(true);
    });

    it('returns ok:false when Docker is missing', async () => {
      mockRunCommand.mockResolvedValueOnce(ok(1, '', ''));
      const plugin = createPlugin(docker() as any);
      const result = await plugin.validate!(makeRuntime());
      expect(result.ok).toBe(false);
    });
  });

  describe('registerTasks', () => {
    beforeEach(() => { mockRunCommand.mockResolvedValue(ok(1, '', 'missing')); });
    it('registers update and colima on macOS without install_desktop', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'colima' });
      const plugin = createPlugin(docker() as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockCreatePkgInstallTask).toHaveBeenCalledWith('colima', '/test', {});
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(4);
    });

    it('does not require Homebrew for Docker Desktop', async () => {
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      const plugin = createPlugin(docker({ install_desktop: true }) as any);
      await plugin.registerTasks!(makeRuntime({ install_desktop: true }));
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(0);
    });

    it('registers update + curl + ca-certificates + gnupg on Linux', async () => {
      mockGetPlatform.mockReturnValue('linux');
      mockCreatePkgUpdateTask.mockReturnValue({ id: 'update' });
      mockCreatePkgInstallTask.mockReturnValue({ id: 'pkg' });
      const plugin = createPlugin(docker() as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockTaskRegistry.register).toHaveBeenCalledTimes(4);
    });

    it('skips registration on Windows', async () => {
      mockGetPlatform.mockReturnValue('windows');
      const plugin = createPlugin(docker() as any);
      await plugin.registerTasks!(makeRuntime());
      expect(mockTaskRegistry.register).not.toHaveBeenCalled();
    });
  });

  describe('docker factory', () => {
    it('sets default options correctly', () => {
      const inst = docker();
      expect(inst.id).toBe('docker');
      expect(inst.category).toBe('tool');
      expect(inst.options.version).toBe('latest');
      expect(inst.options.include_compose).toBe(true);
      expect(inst.options.install_desktop).toBe(false);
    });

    it('respects explicit options', () => {
      const inst = docker({ version: '26.1.0', include_compose: false, install_desktop: true });
      expect(inst.options.version).toBe('26.1.0');
      expect(inst.options.include_compose).toBe(false);
      expect(inst.options.install_desktop).toBe(true);
    });
  });
});


it('falls back to legacy Compose when the Docker subcommand is unavailable', async () => {
  mockRunCommand.mockReset();
  mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker version 26.1.0'));
  mockRunCommand.mockResolvedValueOnce(ok(1, '', 'no compose subcommand'));
  mockRunCommand.mockResolvedValueOnce(ok(0, 'Docker Compose version v2.27.0'));
  const plugin = createPlugin(docker());
  expect((await plugin.detect!(makeRuntime())).ok).toBe(true);
  expect(mockRunCommand).toHaveBeenNthCalledWith(2, 'docker', ['compose', 'version'], expect.anything());
  expect(mockRunCommand).toHaveBeenNthCalledWith(3, 'docker-compose', ['--version'], expect.anything());
});

it('uses Debian repositories and passes the execution environment to installer commands', async () => {
  mockRunCommand.mockReset();
  mockGetPlatform.mockReturnValue('linux');
  mockRunCommand.mockResolvedValueOnce(ok(1, '', 'missing'));
  mockRunCommand.mockResolvedValue(ok(0, '', ''));
  mockFsReadFile.mockResolvedValueOnce('ID=debian\nVERSION_CODENAME=bookworm\n');
  await createPlugin(docker()).apply!(makeRuntime());
  const commands = mockRunCommand.mock.calls.filter(call => call[0] === 'bash');
  expect(commands.some(call => call[1][1].includes('linux/debian/gpg'))).toBe(true);
  expect(commands.some(call => call[1][1].includes('linux/ubuntu'))).toBe(false);
  expect(commands.every(call => call[2].cwd === '/test')).toBe(true);
});
