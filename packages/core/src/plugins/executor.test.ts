import { describe, it, expect, vi, beforeEach } from 'vitest';
import type {
  PluginExecutionNode,
} from './loader.js';
import type { GenesisPlugin, GenesisPluginContext } from './types.js';
import type { TaskRegistry } from '../execution/task-registry.js';
import {
  buildPluginGraph,
  runDetect,
  runApply,
  runValidate,
  runDiff,
} from './executor.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockNode(
  id: string,
  overrides: Partial<GenesisPlugin> = {}
): PluginExecutionNode {
  const category = overrides.category ?? 'tool';
  return {
    instance: {
      id,
      category,
      module: `test/${id}`,
      options: { test: true },
    },
    plugin: {
      id,
      category,
      ...overrides,
    },
  };
}

function mockContext(overrides: Partial<GenesisPluginContext> = {}): GenesisPluginContext {
  return {
    cwd: '/fake/project',
    env: { NODE_ENV: 'test' },
    logger: mockLogger(),
    taskRegistry: mockTaskRegistry(),
    ...overrides,
  };
}

function mockLogger() {
  return {
    info: vi.fn(),
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  } as any;
}

function mockTaskRegistry(): TaskRegistry {
  return {
    register: vi.fn(),
    has: vi.fn().mockReturnValue(false),
    getStatus: vi.fn().mockReturnValue('not-found'),
    getResult: vi.fn(),
    executeAll: vi.fn().mockResolvedValue(new Map()),
    clear: vi.fn(),
    getTaskIds: vi.fn().mockReturnValue([]),
  } as any;
}

// ---------------------------------------------------------------------------
// buildPluginGraph
// ---------------------------------------------------------------------------

describe('buildPluginGraph', () => {
  it('returns nodes in dependency order for linear deps', () => {
    const a = mockNode('a');
    const b = mockNode('b', { dependsOn: ['a'] });
    const c = mockNode('c', { dependsOn: ['b'] });

    const sorted = buildPluginGraph([c, a, b]); // input order is random

    const ids = sorted.map((n) => n.instance.id);
    // a must come before b, b before c
    expect(ids.indexOf('a')).toBeLessThan(ids.indexOf('b'));
    expect(ids.indexOf('b')).toBeLessThan(ids.indexOf('c'));
    expect(ids).toHaveLength(3);
  });

  it('returns all nodes when there are no dependencies', () => {
    const a = mockNode('a');
    const b = mockNode('b');
    const c = mockNode('c');

    const sorted = buildPluginGraph([a, b, c]);

    expect(sorted).toHaveLength(3);
  });

  it('handles a single node', () => {
    const a = mockNode('a');
    const sorted = buildPluginGraph([a]);
    expect(sorted).toEqual([a]);
  });

  it('handles empty array', () => {
    const sorted = buildPluginGraph([]);
    expect(sorted).toEqual([]);
  });

  it('returns nodes in dependency order for diamond deps', () => {
    //    a
    //   / \
    //  b   c
    //   \ /
    //    d
    const a = mockNode('a');
    const b = mockNode('b', { dependsOn: ['a'] });
    const c = mockNode('c', { dependsOn: ['a'] });
    const d = mockNode('d', { dependsOn: ['b', 'c'] });

    const sorted = buildPluginGraph([d, b, c, a]);

    const ids = sorted.map((n) => n.instance.id);
    expect(ids.indexOf('a')).toBeLessThan(ids.indexOf('b'));
    expect(ids.indexOf('a')).toBeLessThan(ids.indexOf('c'));
    expect(ids.indexOf('b')).toBeLessThan(ids.indexOf('d'));
    expect(ids.indexOf('c')).toBeLessThan(ids.indexOf('d'));
    expect(ids).toHaveLength(4);
  });

  it('detects cyclic dependencies and throws', () => {
    // a → b → c → a
    const a = mockNode('a', { dependsOn: ['c'] });
    const b = mockNode('b', { dependsOn: ['a'] });
    const c = mockNode('c', { dependsOn: ['b'] });

    expect(() => buildPluginGraph([a, b, c])).toThrow(
      "Cyclic plugin dependency involving 'a'"
    );
  });

  it('throws when a dependency references a missing plugin', () => {
    const a = mockNode('a', { dependsOn: ['nonexistent'] });

    expect(() => buildPluginGraph([a])).toThrow(
      "Missing plugin 'nonexistent' referenced in dependency"
    );
  });
});

// ---------------------------------------------------------------------------
// runDetect
// ---------------------------------------------------------------------------

describe('runDetect', () => {
  let context: GenesisPluginContext;

  beforeEach(() => {
    context = mockContext();
  });

  it('returns present for plugin whose detect returns ok: true', async () => {
    const node = mockNode('node', {
      detect: vi.fn().mockResolvedValue({ ok: true, details: 'found v20' }),
    });

    const results = await runDetect([node], context);

    expect(results).toEqual([
      { id: 'node', category: 'tool', status: 'present', details: 'found v20' },
    ]);
  });

  it('returns missing for plugin whose detect returns ok: false', async () => {
    const node = mockNode('golang', {
      category: 'language',
      detect: vi.fn().mockResolvedValue({ ok: false, details: 'not on PATH' }),
    });

    const results = await runDetect([node], context);

    expect(results).toEqual([
      { id: 'golang', category: 'language', status: 'missing', details: 'not on PATH' },
    ]);
  });

  it('returns unknown for plugin without detect method', async () => {
    const node = mockNode('simple-tool', {}); // no detect

    const results = await runDetect([node], context);

    expect(results).toEqual([
      { id: 'simple-tool', category: 'tool', status: 'unknown' },
    ]);
  });

  it('handles multiple plugins with mixed detect presence', async () => {
    const a = mockNode('a', { detect: vi.fn().mockResolvedValue({ ok: true }) });
    const b = mockNode('b', {}); // no detect
    const c = mockNode('c', { detect: vi.fn().mockResolvedValue({ ok: false }) });

    const results = await runDetect([a, b, c], context);

    expect(results).toEqual([
      { id: 'a', category: 'tool', status: 'present' },
      { id: 'b', category: 'tool', status: 'unknown' },
      { id: 'c', category: 'tool', status: 'missing' },
    ]);
  });

  it('handles empty nodes array', async () => {
    const results = await runDetect([], context);
    expect(results).toEqual([]);
  });

  it('calls detect with the correct runtime arguments', async () => {
    const detectFn = vi.fn().mockResolvedValue({ ok: true });
    const node = mockNode('node', { detect: detectFn });

    await runDetect([node], context);

    expect(detectFn).toHaveBeenCalledWith({
      instance: node.instance,
      options: node.instance.options,
      context,
    });
  });
});

// ---------------------------------------------------------------------------
// runApply
// ---------------------------------------------------------------------------

describe('runApply', () => {
  let context: GenesisPluginContext;

  beforeEach(() => {
    context = mockContext();
  });

  it('executes in 3 phases: registerTasks → executeAll → apply', async () => {
    const callOrder: string[] = [];
    const registerTasksFn = vi.fn().mockImplementation(async () => {
      callOrder.push('register');
    });
    const applyFn = vi.fn().mockImplementation(async () => {
      callOrder.push('apply');
      return { ok: true, didChange: true };
    });

    const node = mockNode('node', { registerTasks: registerTasksFn, apply: applyFn });

    // executeAll is called between phases
    const execAll = vi.fn().mockResolvedValue(new Map());
    context.taskRegistry.executeAll = execAll;

    await runApply([node], context);

    // Verify call order: register first, then executeAll, then apply
    expect(callOrder[0]).toBe('register');
    expect(execAll).toHaveBeenCalledOnce();
    expect(callOrder[1]).toBe('apply');
  });

  it('plugin without apply returns ok/didChange:false', async () => {
    const node = mockNode('bare', {}); // no apply

    const results = await runApply([node], context);

    expect(results).toEqual([
      { id: 'bare', category: 'tool', ok: true, didChange: false },
    ]);
  });

  it('plugin with apply returns its result', async () => {
    const node = mockNode('installer', {
      apply: vi.fn().mockResolvedValue({ ok: true, didChange: true, details: 'installed' }),
    });

    const results = await runApply([node], context);

    expect(results).toEqual([
      { id: 'installer', category: 'tool', ok: true, didChange: true, details: 'installed' },
    ]);
  });

  it('blocks plugin apply when a system task fails', async () => {
    context.taskRegistry.executeAll = vi.fn().mockResolvedValue(new Map([
      ['system-task', { ok: false, error: 'broken' }],
    ]));
    const apply = vi.fn();
    await expect(runApply([mockNode('installer', { apply })], context)).rejects.toThrow('system-task: broken');
    expect(apply).not.toHaveBeenCalled();
  });

  it('runs hooks in dependency order and skips dependents on failure', async () => {
    const order: string[] = [];
    const a = mockNode('a', {
      preApply: async () => { order.push('pre'); },
      apply: async () => { order.push('apply'); return { ok: false, didChange: true }; },
      postApply: async () => { order.push('post'); },
    });
    const b = mockNode('b', { dependsOn: ['a'], apply: vi.fn() });
    const results = await runApply([b, a], context);
    expect(order).toEqual(['pre', 'apply']);
    expect(b.plugin.apply).not.toHaveBeenCalled();
    expect(results[1]).toMatchObject({ ok: false, details: 'Dependencies failed: a' });
  });

  it('runs postApply only after successful apply', async () => {
    const order: string[] = [];
    const node = mockNode('a', {
      preApply: async () => { order.push('pre'); },
      apply: async () => { order.push('apply'); return { ok: true, didChange: true }; },
      postApply: async () => { order.push('post'); },
    });
    await runApply([node], context);
    expect(order).toEqual(['pre', 'apply', 'post']);
  });

  it('rejects duplicate IDs before any task registration', async () => {
    const registerTasks = vi.fn();
    await expect(runApply([mockNode('a', { registerTasks }), mockNode('a')], context)).rejects.toThrow('Duplicate plugin id');
    expect(registerTasks).not.toHaveBeenCalled();
  });

  it('handles multiple plugins with mixed apply presence', async () => {
    const a = mockNode('a', {
      registerTasks: vi.fn(),
      apply: vi.fn().mockResolvedValue({ ok: true, didChange: true }),
    });
    const b = mockNode('b', {}); // no apply

    const results = await runApply([a, b], context);

    expect(results).toEqual([
      { id: 'a', category: 'tool', ok: true, didChange: true },
      { id: 'b', category: 'tool', ok: true, didChange: false },
    ]);
  });

  it('handles empty nodes array', async () => {
    const results = await runApply([], context);
    expect(results).toEqual([]);
    expect(context.taskRegistry.executeAll).toHaveBeenCalledOnce();
  });

  it('calls registerTasks and apply with correct runtime args', async () => {
    const registerTasksFn = vi.fn();
    const applyFn = vi.fn().mockResolvedValue({ ok: true, didChange: false });
    const node = mockNode('pkg', { registerTasks: registerTasksFn, apply: applyFn });

    await runApply([node], context);

    expect(registerTasksFn).toHaveBeenCalledWith({
      instance: node.instance,
      options: node.instance.options,
      context,
    });
    expect(applyFn).toHaveBeenCalledWith({
      instance: node.instance,
      options: node.instance.options,
      context,
    });
  });
});

// ---------------------------------------------------------------------------
// runValidate
// ---------------------------------------------------------------------------

describe('runValidate', () => {
  let context: GenesisPluginContext;

  beforeEach(() => {
    context = mockContext();
  });

  it('plugin with validate returns its result', async () => {
    const node = mockNode('linter', {
      validate: vi.fn().mockResolvedValue({ ok: true, message: 'all good' }),
    });

    const results = await runValidate([node], context);

    expect(results).toEqual([
      { id: 'linter', category: 'tool', ok: true, message: 'all good' },
    ]);
  });

  it('plugin without validate defaults to ok', async () => {
    const node = mockNode('bare', {});

    const results = await runValidate([node], context);

    expect(results).toEqual([
      { id: 'bare', category: 'tool', ok: true },
    ]);
  });

  it('plugin with failing validate returns ok: false with message', async () => {
    const node = mockNode('checker', {
      validate: vi.fn().mockResolvedValue({ ok: false, message: 'config invalid' }),
    });

    const results = await runValidate([node], context);

    expect(results).toEqual([
      { id: 'checker', category: 'tool', ok: false, message: 'config invalid' },
    ]);
  });

  it('handles multiple plugins with mixed validate presence', async () => {
    const a = mockNode('a', {
      validate: vi.fn().mockResolvedValue({ ok: true, message: 'ok a' }),
    });
    const b = mockNode('b', {}); // no validate
    const c = mockNode('c', {
      validate: vi.fn().mockResolvedValue({ ok: false, message: 'bad c' }),
    });

    const results = await runValidate([a, b, c], context);

    expect(results).toEqual([
      { id: 'a', category: 'tool', ok: true, message: 'ok a' },
      { id: 'b', category: 'tool', ok: true },
      { id: 'c', category: 'tool', ok: false, message: 'bad c' },
    ]);
  });

  it('handles empty nodes array', async () => {
    const results = await runValidate([], context);
    expect(results).toEqual([]);
  });

  it('calls validate with correct runtime args', async () => {
    const validateFn = vi.fn().mockResolvedValue({ ok: true });
    const node = mockNode('x', { validate: validateFn });

    await runValidate([node], context);

    expect(validateFn).toHaveBeenCalledWith({
      instance: node.instance,
      options: node.instance.options,
      context,
    });
  });
});

// ---------------------------------------------------------------------------
// runDiff
// ---------------------------------------------------------------------------

describe('runDiff', () => {
  it('returns same result as runDetect', async () => {
    const context = mockContext();
    const node = mockNode('node', {
      detect: vi.fn().mockResolvedValue({ ok: true, details: 'found' }),
    });

    const results = await runDiff([node], context);

    expect(results).toEqual([
      { id: 'node', category: 'tool', status: 'present', details: 'found' },
    ]);
  });

  it('returns unknown for plugin without detect', async () => {
    const context = mockContext();
    const node = mockNode('bare', {});

    const results = await runDiff([node], context);

    expect(results).toEqual([
      { id: 'bare', category: 'tool', status: 'unknown' },
    ]);
  });
});
