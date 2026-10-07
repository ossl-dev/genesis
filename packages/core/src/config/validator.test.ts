import { describe, it, expect } from 'vitest';
import { ZodError } from 'zod';
import { validateConfig } from './validator.js';

describe('validateConfig', () => {
  // ---------------------------------------------------------------------------
  // Happy paths
  // ---------------------------------------------------------------------------

  it('validates a minimal config with only env', () => {
    const config = validateConfig({ env: { NODE_ENV: 'test' } });
    expect(config).toEqual({ env: { NODE_ENV: 'test' } });
  });

  it('validates a config with tools array', () => {
    const config = validateConfig({
      tools: [
        { id: 'node', category: 'tool', module: '@ossl/genesis-plugins/node' },
      ],
    });
    expect(config.tools).toHaveLength(1);
  });

  it('validates a fully populated config with all sections', () => {
    const config = validateConfig({
      tools: [
        { id: 'node', category: 'tool', module: '@ossl/genesis-plugins/node' },
      ],
      sdks: [
        { id: 'aws', category: 'sdk', module: '@ossl/genesis-plugins/aws-sdk' },
      ],
      languages: [
        { id: 'python', category: 'language', module: '@ossl/genesis-plugins/python' },
      ],
      repositories: [
        { url: 'https://github.com/foo/bar', folder: 'bar' },
      ],
      scripts: [
        { name: 'build', command: 'npm run build', description: 'Builds the project' },
      ],
      env: { NODE_ENV: 'production', DEBUG: 'true' },
    });

    expect(config.tools).toHaveLength(1);
    expect(config.sdks).toHaveLength(1);
    expect(config.languages).toHaveLength(1);
    expect(config.repositories).toHaveLength(1);
    expect(config.scripts).toHaveLength(1);
    expect(config.env).toEqual({ NODE_ENV: 'production', DEBUG: 'true' });
  });

  it('validates an empty object (all sections optional)', () => {
    const config = validateConfig({});
    expect(config).toEqual({});
  });

  it('validates a config with empty sections', () => {
    const config = validateConfig({
      tools: [],
      sdks: [],
      languages: [],
      repositories: [],
      scripts: [],
      env: {},
    });
    expect(config.tools).toEqual([]);
    expect(config.sdks).toEqual([]);
    expect(config.env).toEqual({});
  });

  it('validates plugins with all valid categories', () => {
    const config = validateConfig({
      tools: [
        { id: 'git', category: 'tool', module: 'pkg-a' },
      ],
      sdks: [
        { id: 'aws', category: 'sdk', module: 'pkg-b' },
      ],
      languages: [
        { id: 'ts', category: 'language', module: 'pkg-c' },
      ],
    });

    // Also test 'library' and 'framework'
    const config2 = validateConfig({
      tools: [
        { id: 'lib', category: 'library', module: 'pkg-d' },
        { id: 'fw', category: 'framework', module: 'pkg-e' },
      ],
    });

    expect(config.tools![0].category).toBe('tool');
    expect(config2.tools![0].category).toBe('library');
    expect(config2.tools![1].category).toBe('framework');
  });

  it('validates plugin instances with options', () => {
    const config = validateConfig({
      tools: [
        {
          id: 'node',
          category: 'tool',
          module: '@ossl/genesis-plugins/node',
          options: { version: '20', installGlobal: true },
        },
      ],
    });
    expect(config.tools![0].options).toEqual({ version: '20', installGlobal: true });
  });

  it('validates script spec without optional description', () => {
    const config = validateConfig({
      scripts: [{ name: 'lint', command: 'eslint .' }],
    });
    expect(config.scripts![0].description).toBeUndefined();
  });

  it('validates multiple repos and scripts', () => {
    const config = validateConfig({
      repositories: [
        { url: 'a', folder: 'a' },
        { url: 'b', folder: 'b' },
      ],
      scripts: [
        { name: 'one', command: 'echo 1' },
        { name: 'two', command: 'echo 2' },
      ],
    });
    expect(config.repositories).toHaveLength(2);
    expect(config.scripts).toHaveLength(2);
  });

  // ---------------------------------------------------------------------------
  // Error paths
  // ---------------------------------------------------------------------------

  it('throws ZodError for invalid category', () => {
    expect(() =>
      validateConfig({
        tools: [
          { id: 'x', category: 'invalid-category', module: 'pkg' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for missing required fields in plugin instance', () => {
    expect(() =>
      validateConfig({
        tools: [

          { category: 'tool' }, // missing id and module
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for plugin instance missing id', () => {
    expect(() =>
      validateConfig({
        tools: [

          { category: 'tool', module: 'pkg' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for plugin instance missing module', () => {
    expect(() =>
      validateConfig({
        tools: [

          { id: 'x', category: 'tool' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for invalid repo spec (missing folder)', () => {
    expect(() =>
      validateConfig({
        repositories: [

          { url: 'https://example.com' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for invalid repo spec (missing url)', () => {
    expect(() =>
      validateConfig({
        repositories: [

          { folder: 'x' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for invalid script spec (missing command)', () => {
    expect(() =>
      validateConfig({
        scripts: [

          { name: 'test' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for invalid script spec (missing name)', () => {
    expect(() =>
      validateConfig({
        scripts: [

          { command: 'npm test' },
        ],
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for non-object config', () => {
    expect(() => validateConfig(null)).toThrow(ZodError);
    expect(() => validateConfig(undefined)).toThrow(ZodError);
    expect(() => validateConfig('string')).toThrow(ZodError);
    expect(() => validateConfig(42)).toThrow(ZodError);
  });

  it('throws ZodError for tools being a non-array', () => {
    expect(() =>
      validateConfig({
        tools: 'not-an-array',
      })
    ).toThrow(ZodError);
  });

  it('throws ZodError for env being a non-record', () => {
    expect(() =>
      validateConfig({
        env: [{ key: 'val' }],
      })
    ).toThrow(ZodError);
  });

  it('includes ZodError with descriptive issues', () => {
    try {
      validateConfig({
        tools: [

          { id: 123, category: 'invalid', module: true },
        ],
      });
      expect.fail('Should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ZodError);
      const zodErr = err as ZodError;
      expect(zodErr.issues.length).toBeGreaterThan(0);
    }
  });
});


describe('config safety', () => {
  it('rejects duplicate IDs across sections', () => {
    const plugin = { id: 'node', module: 'plugin', category: 'tool' };
    expect(() => validateConfig({ tools: [plugin], languages: [plugin] })).toThrow("Duplicate plugin id 'node'");
  });

  it('rejects empty plugin IDs and modules', () => {
    expect(() => validateConfig({ tools: [{ id: ' ', module: '', category: 'tool' }] })).toThrow(ZodError);
  });

  it('accepts the documented repository path and lifecycle timing', () => {
    expect(validateConfig({ repositories: [{ url: 'repo', path: './repo', branch: 'main' }], scripts: [{ name: 'start', command: 'echo ready', when: 'before' }] })).toEqual({ repositories: [{ url: 'repo', folder: './repo', branch: 'main' }], scripts: [{ name: 'start', command: 'echo ready', when: 'before' }] });
  });
});
