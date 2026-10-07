import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock platform and shell — imported by task-registry.ts
vi.mock('../os/platform.js', () => ({
  getPlatform: vi.fn(() => 'linux' as const),
}));

vi.mock('../os/shell.js', () => ({
  runCommand: vi.fn(),
}));

import { TaskRegistry, type Task, type TaskResult } from './task-registry.js';

function mockLogger() {
  return {
    info: vi.fn(),
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  } as any;
}

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 'default',
    description: 'default task',
    executor: vi.fn().mockResolvedValue({ ok: true }),
    ...overrides,
  };
}

function successResult(details?: string): TaskResult {
  return { ok: true, details };
}

function failResult(error?: string): TaskResult {
  return { ok: false, error };
}

describe('TaskRegistry', () => {
  let registry: TaskRegistry;
  let logger: ReturnType<typeof mockLogger>;

  beforeEach(() => {
    vi.clearAllMocks();
    logger = mockLogger();
    registry = new TaskRegistry(logger);
  });

  // ---------------------------------------------------------------------------
  // register
  // ---------------------------------------------------------------------------

  it('registers a single task', () => {
    const task = makeTask({ id: 'task-1' });
    registry.register(task);

    expect(registry.has('task-1')).toBe(true);
    expect(registry.getStatus('task-1')).toBe('pending');
  });

  it('skips registering a duplicate task (does not overwrite)', () => {
    const exec1 = vi.fn().mockResolvedValue(successResult());
    const exec2 = vi.fn().mockResolvedValue(successResult());

    registry.register(makeTask({ id: 'dup', executor: exec1 }));
    registry.register(makeTask({ id: 'dup', executor: exec2 }));

    // Should still use the first registered executor
    expect(registry.has('dup')).toBe(true);
    expect(logger.debug).toHaveBeenCalledWith(
      expect.stringContaining('already registered')
    );
  });

  it('logs debug when registering a task', () => {
    registry.register(makeTask({ id: 'log-test', description: 'Log me' }));
    expect(logger.debug).toHaveBeenCalledWith(
      expect.stringContaining('Registering task: log-test - Log me')
    );
  });

  // ---------------------------------------------------------------------------
  // has
  // ---------------------------------------------------------------------------

  it('has() returns true for registered task', () => {
    registry.register(makeTask({ id: 'exists' }));
    expect(registry.has('exists')).toBe(true);
  });

  it('has() returns false for unregistered task', () => {
    expect(registry.has('nope')).toBe(false);
  });

  // ---------------------------------------------------------------------------
  // getStatus
  // ---------------------------------------------------------------------------

  it('getStatus returns not-found for unregistered task', () => {
    expect(registry.getStatus('ghost')).toBe('not-found');
  });

  it('getStatus returns correct status after execution', async () => {
    registry.register(
      makeTask({ id: 'status-test', executor: vi.fn().mockResolvedValue(successResult()) })
    );
    await registry.executeAll();

    expect(registry.getStatus('status-test')).toBe('completed');
  });

  it('getStatus returns failed after task fails', async () => {
    registry.register(
      makeTask({ id: 'fail-test', executor: vi.fn().mockResolvedValue(failResult('boom')) })
    );
    await registry.executeAll();

    expect(registry.getStatus('fail-test')).toBe('failed');
  });

  // ---------------------------------------------------------------------------
  // getResult
  // ---------------------------------------------------------------------------

  it('getResult returns undefined for unregistered task', () => {
    expect(registry.getResult('ghost')).toBeUndefined();
  });

  it('getResult returns undefined for pending task', () => {
    registry.register(makeTask({ id: 'pend' }));
    expect(registry.getResult('pend')).toBeUndefined();
  });

  it('getResult returns result after successful execution', async () => {
    registry.register(
      makeTask({
        id: 'res-test',
        executor: vi.fn().mockResolvedValue(successResult('all good')),
      })
    );
    await registry.executeAll();

    expect(registry.getResult('res-test')).toEqual({ ok: true, details: 'all good' });
  });

  it('getResult returns result after failed execution', async () => {
    registry.register(
      makeTask({
        id: 'res-fail',
        executor: vi.fn().mockResolvedValue(failResult('kaput')),
      })
    );
    await registry.executeAll();

    expect(registry.getResult('res-fail')).toEqual({ ok: false, error: 'kaput' });
  });

  // ---------------------------------------------------------------------------
  // executeAll — basic
  // ---------------------------------------------------------------------------

  it('executeAll with empty registry returns empty map', async () => {
    const results = await registry.executeAll();
    expect(results.size).toBe(0);
    expect(logger.debug).toHaveBeenCalledWith('No tasks to execute');
  });

  it('executes a single task successfully', async () => {
    const executor = vi.fn().mockResolvedValue(successResult('done'));
    registry.register(makeTask({ id: 't1', executor }));

    const results = await registry.executeAll();

    expect(executor).toHaveBeenCalledOnce();
    expect(results.get('t1')).toEqual({ ok: true, details: 'done' });
    expect(registry.getStatus('t1')).toBe('completed');
  });

  it('executes a single task that fails', async () => {
    const executor = vi.fn().mockResolvedValue(failResult('error msg'));
    registry.register(makeTask({ id: 't1', executor }));

    const results = await registry.executeAll();

    expect(results.get('t1')).toEqual({ ok: false, error: 'error msg' });
    expect(registry.getStatus('t1')).toBe('failed');
    expect(logger.warn).toHaveBeenCalledWith(
      expect.stringContaining('Task t1 failed')
    );
  });

  it('handles a task that throws an error', async () => {
    const executor = vi.fn().mockRejectedValue(new Error('crash'));
    registry.register(makeTask({ id: 'thrower', executor }));

    const results = await registry.executeAll();

    expect(results.get('thrower')).toEqual({ ok: false, error: 'crash' });
    expect(registry.getStatus('thrower')).toBe('failed');
    expect(logger.error).toHaveBeenCalledWith(
      expect.stringContaining('Task thrower threw error')
    );
  });

  // ---------------------------------------------------------------------------
  // executeAll — dependencies
  // ---------------------------------------------------------------------------

  it('executes tasks in topological order based on dependencies', async () => {
    const order: string[] = [];
    const makeOrderedTask = (id: string, deps?: string[]) =>
      makeTask({
        id,
        dependsOn: deps,
        executor: vi.fn().mockImplementation(async () => {
          order.push(id);
          return successResult();
        }),
      });

    // A depends on nothing; B depends on A; C depends on B
    registry.register(makeOrderedTask('A'));
    registry.register(makeOrderedTask('B', ['A']));
    registry.register(makeOrderedTask('C', ['B']));

    await registry.executeAll();

    // Topological order: A, B, C
    expect(order).toEqual(['A', 'B', 'C']);
  });

  it('executes tasks with complex dependency graph', async () => {
    const order: string[] = [];
    const makeOrderedTask = (id: string, deps?: string[]) =>
      makeTask({
        id,
        dependsOn: deps,
        executor: vi.fn().mockImplementation(async () => {
          order.push(id);
          return successResult();
        }),
      });

    //    A
    //   / \
    //  B   C
    //   \ /
    //    D
    registry.register(makeOrderedTask('A'));
    registry.register(makeOrderedTask('B', ['A']));
    registry.register(makeOrderedTask('C', ['A']));
    registry.register(makeOrderedTask('D', ['B', 'C']));

    await registry.executeAll();

    // A must be first; D must be last; B and C can be in either order after A
    expect(order[0]).toBe('A');
    expect(order[order.length - 1]).toBe('D');
    expect(order.indexOf('B')).toBeGreaterThan(order.indexOf('A'));
    expect(order.indexOf('C')).toBeGreaterThan(order.indexOf('A'));
    expect(order.indexOf('D')).toBeGreaterThan(order.indexOf('B'));
    expect(order.indexOf('D')).toBeGreaterThan(order.indexOf('C'));
  });

  it('skips task whose dependency failed', async () => {
    registry.register(
      makeTask({
        id: 'fail-dep',
        executor: vi.fn().mockResolvedValue(failResult('broken')),
      })
    );
    const childExec = vi.fn();
    registry.register(
      makeTask({
        id: 'child',
        dependsOn: ['fail-dep'],
        executor: childExec,
      })
    );

    const results = await registry.executeAll();

    // Child should be skipped
    expect(childExec).not.toHaveBeenCalled();
    expect(registry.getStatus('child')).toBe('failed');
    expect(results.get('child')).toEqual({ ok: false, error: 'Dependencies failed' });
    expect(logger.warn).toHaveBeenCalledWith(
      expect.stringContaining('Skipping task child due to failed dependencies')
    );
  });

  it('detects and throws on cyclic dependency', async () => {
    // A → B → C → A
    registry.register(makeTask({ id: 'A', dependsOn: ['C'] }));
    registry.register(makeTask({ id: 'B', dependsOn: ['A'] }));
    registry.register(makeTask({ id: 'C', dependsOn: ['B'] }));

    await expect(registry.executeAll()).rejects.toThrow('Cyclic task dependency detected');
  });

  it('detects self-referencing cyclic dependency', async () => {
    registry.register(makeTask({ id: 'self', dependsOn: ['self'] }));

    await expect(registry.executeAll()).rejects.toThrow('Cyclic task dependency detected');
  });

  // ---------------------------------------------------------------------------
  // executeAll — priority ordering
  // ---------------------------------------------------------------------------

  it('executes tasks in priority order when no dependencies', async () => {
    const order: string[] = [];
    const makeOrderedTask = (id: string, priority: number) =>
      makeTask({
        id,
        priority,
        executor: vi.fn().mockImplementation(async () => {
          order.push(id);
          return successResult();
        }),
      });

    registry.register(makeOrderedTask('low', 0));
    registry.register(makeOrderedTask('high', 10));
    registry.register(makeOrderedTask('mid', 5));

    await registry.executeAll();

    // Higher priority first: high, mid, low
    expect(order).toEqual(['high', 'mid', 'low']);
  });

  it('respects dependencies over priority', async () => {
    const order: string[] = [];
    const makeOrderedTask = (id: string, deps?: string[], priority = 0) =>
      makeTask({
        id,
        dependsOn: deps,
        priority,
        executor: vi.fn().mockImplementation(async () => {
          order.push(id);
          return successResult();
        }),
      });

    // B has higher priority than A, but B depends on A → A must run first
    registry.register(makeOrderedTask('A', undefined, 0));
    registry.register(makeOrderedTask('B', ['A'], 100));

    await registry.executeAll();

    expect(order).toEqual(['A', 'B']);
  });

  // ---------------------------------------------------------------------------
  // Multiple tasks without dependencies
  // ---------------------------------------------------------------------------

  it('executes multiple independent tasks (all succeed)', async () => {
    registry.register(
      makeTask({ id: 'a', executor: vi.fn().mockResolvedValue(successResult()) })
    );
    registry.register(
      makeTask({ id: 'b', executor: vi.fn().mockResolvedValue(successResult()) })
    );
    registry.register(
      makeTask({ id: 'c', executor: vi.fn().mockResolvedValue(successResult()) })
    );

    const results = await registry.executeAll();

    expect(results.size).toBe(3);
    for (const [, r] of results) {
      expect(r.ok).toBe(true);
    }
    expect(logger.info).toHaveBeenCalledWith(
      expect.stringContaining('3 succeeded, 0 failed')
    );
  });

  it('reports mixed success/failure counts', async () => {
    registry.register(
      makeTask({ id: 'ok1', executor: vi.fn().mockResolvedValue(successResult()) })
    );
    registry.register(
      makeTask({ id: 'ok2', executor: vi.fn().mockResolvedValue(successResult()) })
    );
    registry.register(
      makeTask({ id: 'bad', executor: vi.fn().mockResolvedValue(failResult('nope')) })
    );

    const results = await registry.executeAll();

    expect(results.size).toBe(3);
    expect(logger.info).toHaveBeenCalledWith(
      expect.stringContaining('2 succeeded, 1 failed')
    );
  });

  // ---------------------------------------------------------------------------
  // clear
  // ---------------------------------------------------------------------------

  it('clear removes all tasks', () => {
    registry.register(makeTask({ id: 'a' }));
    registry.register(makeTask({ id: 'b' }));

    expect(registry.has('a')).toBe(true);
    registry.clear();
    expect(registry.has('a')).toBe(false);
    expect(registry.has('b')).toBe(false);
  });

  it('clear on empty registry is a no-op', () => {
    expect(() => registry.clear()).not.toThrow();
  });

  // ---------------------------------------------------------------------------
  // getTaskIds
  // ---------------------------------------------------------------------------

  it('getTaskIds returns all registered IDs', () => {
    registry.register(makeTask({ id: 'x' }));
    registry.register(makeTask({ id: 'y' }));
    registry.register(makeTask({ id: 'z' }));

    const ids = registry.getTaskIds();
    expect(ids).toContain('x');
    expect(ids).toContain('y');
    expect(ids).toContain('z');
    expect(ids).toHaveLength(3);
  });

  it('getTaskIds returns empty array for empty registry', () => {
    expect(registry.getTaskIds()).toEqual([]);
  });

  // ---------------------------------------------------------------------------
  // Edge cases
  // ---------------------------------------------------------------------------

  it('tasks with default priority (undefined) sort to zero', async () => {
    const order: string[] = [];
    registry.register(
      makeTask({
        id: 'no-prio',
        executor: vi.fn().mockImplementation(async () => {
          order.push('no-prio');
          return successResult();
        }),
      })
    );
    registry.register(
      makeTask({
        id: 'with-prio',
        priority: 5,
        executor: vi.fn().mockImplementation(async () => {
          order.push('with-prio');
          return successResult();
        }),
      })
    );

    await registry.executeAll();

    expect(order).toEqual(['with-prio', 'no-prio']);
  });
});


describe('task execution boundaries', () => {
  it('rejects missing dependencies before executing any tasks', async () => {
    const registry = new TaskRegistry(mockLogger());
    const executor = vi.fn().mockResolvedValue({ ok: true });
    registry.register({ id: 'first', description: 'first', executor });
    registry.register({ id: 'second', description: 'second', executor, dependsOn: ['missing'] });
    await expect(registry.executeAll()).rejects.toThrow('Missing task dependency: missing');
    expect(executor).not.toHaveBeenCalled();
  });

  it('retains results without rerunning tasks on repeated execution', async () => {
    const registry = new TaskRegistry(mockLogger());
    const executor = vi.fn().mockResolvedValue({ ok: true });
    registry.register({ id: 'once', description: 'once', executor });
    await registry.executeAll();
    expect((await registry.executeAll()).get('once')).toEqual({ ok: true });
    expect(executor).toHaveBeenCalledOnce();
  });
});
