import { describe, it, expect, vi, beforeEach } from 'vitest';

// ---- MOCKS (hoisted by vitest) ----
// All variables referenced inside vi.mock factories must be hoisted

const { mockExistsSync, mockReadFile, mockYamlParse } = vi.hoisted(() => ({
  mockExistsSync: vi.fn(),
  mockReadFile: vi.fn(),
  mockYamlParse: vi.fn(),
}));

vi.mock('node:fs', () => ({
  default: {
    existsSync: mockExistsSync,
    promises: { readFile: mockReadFile },
  },
}));

vi.mock('yaml', () => ({
  default: { parse: mockYamlParse },
}));

vi.mock('node:url', () => ({
  pathToFileURL: (filePath: string) => ({ href: filePath }),
}));

// Virtual fixture paths use POSIX syntax independent of the host OS.
vi.mock('node:path', async importOriginal => {
  const actual = await importOriginal<typeof import('node:path')>();
  return { ...actual, default: actual.posix };
});

// Static mocks for each TS config test scenario (separate paths to avoid ESM caching issues)
vi.mock('/fake/ts-default/genesis.config.ts', () => ({
  default: { tools: [], env: { NODE_ENV: 'test' } },
}));

vi.mock('/fake/ts-named/genesis.config.ts', () => ({
  default: undefined,
  config: { tools: [], env: { HELLO: 'world' } },
}));

vi.mock('/fake/ts-fallback/genesis.config.ts', () => ({
  default: undefined,
  config: undefined,
  tools: [],
  sdks: undefined,
  languages: undefined,
  repositories: undefined,
  scripts: undefined,
  env: { FALLBACK: 'yes' },
}));

vi.mock('/fake/ts-env-only/genesis.config.ts', () => ({
  default: { env: { FOO: 'bar' } },
}));

// ---- Imports ----

import { loadConfig } from './parser.js';

describe('loadConfig', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ---------------------------------------------------------------------------
  // TS config
  // ---------------------------------------------------------------------------

  it('loads genesis.config.ts that exports a default config', async () => {
    mockExistsSync.mockImplementation((p: string) =>
      p === '/fake/ts-default/genesis.config.ts'
    );

    const config = await loadConfig('/fake/ts-default');

    expect(config).toEqual({ tools: [], env: { NODE_ENV: 'test' } });
    expect(mockExistsSync).toHaveBeenCalledWith('/fake/ts-default/genesis.config.ts');
  });

  it('loads genesis.config.ts that exports a named config', async () => {
    mockExistsSync.mockImplementation((p: string) =>
      p === '/fake/ts-named/genesis.config.ts'
    );

    const config = await loadConfig('/fake/ts-named');

    expect(config).toEqual({ tools: [], env: { HELLO: 'world' } });
  });

  it('loads genesis.config.ts with only env section (no tools)', async () => {
    mockExistsSync.mockImplementation((p: string) =>
      p === '/fake/ts-env-only/genesis.config.ts'
    );

    const config = await loadConfig('/fake/ts-env-only');

    expect(config).toEqual({ env: { FOO: 'bar' } });
  });

  it('falls back to mod itself when default and config are absent', async () => {
    mockExistsSync.mockImplementation((p: string) =>
      p === '/fake/ts-fallback/genesis.config.ts'
    );

    const config = await loadConfig('/fake/ts-fallback');

    // `mod.default ?? mod.config ?? mod` — since neither default nor config exist,
    // the whole object is used
    expect(config).toEqual({ tools: [], env: { FALLBACK: 'yes' } });
  });

  // ---------------------------------------------------------------------------
  // YAML config — normalised path (type-based entries, no id/module/category)
  // ---------------------------------------------------------------------------

  it('loads genesis.config.yaml with tools section (node, python)', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('tools:\n  - type: node\n  - type: python\n');
    mockYamlParse.mockReturnValue({
      tools: [{ type: 'node' }, { type: 'python' }],
    });

    const config = await loadConfig('/fake/project');

    expect(config.tools).toHaveLength(2);
    expect(config.tools![0]).toEqual({
      id: 'node',
      category: 'tool',
      module: '@ossl/genesis-plugins/node',
      options: {},
    });
    expect(config.tools![1]).toEqual({
      id: 'python',
      category: 'language',
      module: '@ossl/genesis-plugins/python',
      options: {},
    });
  });

  it('loads genesis.config.yaml with tools + sdks + languages', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('tools: [...]; sdks: [...]; languages: [...]');
    mockYamlParse.mockReturnValue({
      tools: [{ type: 'node' }],
      sdks: [{ type: 'java' }],
      languages: [{ type: 'python' }],
    });

    const config = await loadConfig('/fake/project');

    expect(config.tools).toHaveLength(1);
    expect(config.sdks).toHaveLength(1);
    expect(config.languages).toHaveLength(1);
    expect(config.tools![0].category).toBe('tool');
    expect(config.sdks![0].id).toBe('java');
    expect(config.languages![0].category).toBe('language');
  });

  it('loads genesis.config.yaml with repositories, scripts, and env sections', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('...');
    mockYamlParse.mockReturnValue({
      tools: [{ type: 'node' }],
      repositories: [{ url: 'https://github.com/foo/bar', folder: 'bar' }],
      scripts: [{ name: 'build', command: 'npm run build', description: 'Build it' }],
      env: { DATABASE_URL: 'postgres://localhost' },
    });

    const config = await loadConfig('/fake/project');

    expect(config.tools).toHaveLength(1);
    expect(config.repositories).toEqual([
      { url: 'https://github.com/foo/bar', folder: 'bar' },
    ]);
    expect(config.scripts).toEqual([
      { name: 'build', command: 'npm run build', description: 'Build it' },
    ]);
    expect(config.env).toEqual({ DATABASE_URL: 'postgres://localhost' });
  });

  // ---------------------------------------------------------------------------
  // YAML config — already GenesisConfig shape (skips normalisation)
  // ---------------------------------------------------------------------------

  it('skips normalisation when YAML already has GenesisConfig shape (id/module/category)', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('tools:\n  - id: node\n    module: my-module\n    category: tool\n');
    mockYamlParse.mockReturnValue({
      tools: [
        { id: 'node', module: 'my-module', category: 'tool' },
      ],
    });

    const config = await loadConfig('/fake/project');

    // Normalisation would set module to @ossl/genesis-plugins/node — it was skipped
    expect(config.tools![0].module).toBe('my-module');
    expect(config.tools![0].id).toBe('node');
  });

  // ---------------------------------------------------------------------------
  // Edge cases — YAML
  // ---------------------------------------------------------------------------

  it('handles YAML with empty tools array', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('tools: []\n');
    mockYamlParse.mockReturnValue({ tools: [] });

    const config = await loadConfig('/fake/project');

    expect(config.tools).toEqual([]);
  });

  it('handles YAML with only env section (no tools)', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('env:\n  FOO: bar\n');
    mockYamlParse.mockReturnValue({ env: { FOO: 'bar' } });

    const config = await loadConfig('/fake/project');

    expect(config).toEqual({ env: { FOO: 'bar' } });
  });

  it('handles YAML with only scripts section', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('scripts:\n  - name: start\n    command: node index.js\n');
    mockYamlParse.mockReturnValue({
      scripts: [{ name: 'start', command: 'node index.js' }],
    });

    const config = await loadConfig('/fake/project');

    expect(config.scripts).toEqual([{ name: 'start', command: 'node index.js' }]);
  });

  // ---------------------------------------------------------------------------
  // Error paths
  // ---------------------------------------------------------------------------

  it('throws when no config file is found', async () => {
    mockExistsSync.mockReturnValue(false);

    await expect(loadConfig('/fake/project')).rejects.toThrow(
      'No genesis.config.ts or genesis.config.yaml found'
    );
  });

  it('throws when YAML contains an unknown plugin type', async () => {
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('tools:\n  - type: unknown-plugin\n');
    mockYamlParse.mockReturnValue({
      tools: [{ type: 'unknown-plugin' }],
    });

    await expect(loadConfig('/fake/project')).rejects.toThrow(
      "Unknown plugin type 'unknown-plugin' in genesis.config.yaml"
    );
  });
});


describe('config input boundaries', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockExistsSync.mockImplementation((p: string) => p.endsWith('.yaml'));
    mockReadFile.mockResolvedValue('');
  });

  it.each([null, 7, 'string', [], { tools: {} }, { tools: [null] }])('rejects malformed YAML %j with file context', async raw => {
    mockYamlParse.mockReturnValue(raw);
    await expect(loadConfig('/fake/project')).rejects.toThrow('/fake/project/genesis.config.yaml:');
  });

  it('normalizes each plugin entry independently', async () => {
    mockYamlParse.mockReturnValue({ tools: [
      { id: 'custom', module: 'custom-plugin', category: 'tool' },
      { type: 'node', version: '22' },
    ] });
    const config = await loadConfig('/fake/project');
    expect(config.tools?.map(plugin => plugin.id)).toEqual(['custom', 'node']);
  });

  it('honors an explicit file even when a default config exists', async () => {
    mockYamlParse.mockReturnValue({ env: { MODE: 'override' } });
    expect(await loadConfig('/fake/project', 'configs/dev.yml')).toEqual({ env: { MODE: 'override' } });
    expect(mockReadFile).toHaveBeenCalledWith('/fake/project/configs/dev.yml', 'utf8');
  });

  it('interpolates nested environment references without executing shell text', async () => {
    mockYamlParse.mockReturnValue({ tools: [{ type: 'node', version: '${VERSION}' }], env: { LOCATION: '${HOME}/work', LITERAL: '$(echo unsafe)' } });
    const config = await loadConfig('/fake/project', undefined, { HOME: '/tmp/home', VERSION: '22' });
    expect(config.tools?.[0].options).toEqual({ version: '22' });
    expect(config.env).toEqual({ LOCATION: '/tmp/home/work', LITERAL: '$(echo unsafe)' });
  });

  it('reports the path of an unset environment reference', async () => {
    mockYamlParse.mockReturnValue({ env: { TOKEN: '${MISSING}' } });
    await expect(loadConfig('/fake/project', undefined, {})).rejects.toThrow("config.env.TOKEN: environment variable 'MISSING' is not set");
  });

  it('reports nested validation paths', async () => {
    mockYamlParse.mockReturnValue({ tools: [{ id: 'node', category: 'tool', module: 42 }] });
    await expect(loadConfig('/fake/project')).rejects.toThrow('tools.0.module:');
  });
});


it.each(['bun', 'deno', 'pnpm', 'yarn'])('normalizes the %s YAML shorthand', async type => {
  mockExistsSync.mockReturnValue(true);
  mockYamlParse.mockReturnValue({ tools: [{ type, version: '1.2.3' }] });
  mockReadFile.mockResolvedValue('');
  const config = await loadConfig('/fake/plugins', 'genesis.config.yaml');
  expect(config.tools?.[0]).toEqual({ id: type, category: 'tool', module: `@ossl/genesis-plugins/${type}`, options: { version: '1.2.3' } });
});
