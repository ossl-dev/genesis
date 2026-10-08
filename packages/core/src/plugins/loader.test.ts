import { describe, expect, it, vi } from "vitest";
import { loadPlugin } from "./loader.js";

vi.mock("valid-plugin", () => ({ createPlugin: (instance: { id: string; category: string }) => ({
  id: instance.id, category: instance.category,
  parseOptions: (options: unknown) => ({ ...options as object, defaulted: true }),
}) }));
vi.mock("invalid-options-plugin", () => ({ createPlugin: (instance: { id: string; category: string }) => ({
  id: instance.id, category: instance.category,
  parseOptions: () => { throw new Error("version is required"); },
}) }));
vi.mock("invalid-plugin", () => ({ createPlugin: "not a function" }));

describe("plugin loading", () => {
  it("normalizes options without mutating the input instance", async () => {
    const instance = { id: "custom", category: "tool" as const, module: "valid-plugin", options: { version: "1" } };
    const node = await loadPlugin(instance);
    expect(node.instance.options).toEqual({ version: "1", defaulted: true });
    expect(instance.options).toEqual({ version: "1" });
  });

  it("reports options errors with plugin identity", async () => {
    await expect(loadPlugin({ id: "custom", category: "tool", module: "invalid-options-plugin" })).rejects.toThrow("Invalid options for plugin 'custom': version is required");
  });

  it("rejects an invalid factory", async () => {
    await expect(loadPlugin({ id: "custom", category: "tool", module: "invalid-plugin" })).rejects.toThrow("does not export createPlugin");
  });

  it("reports failed imports with both ID and module", async () => {
    await expect(loadPlugin({ id: "custom", category: "tool", module: "nonexistent-genesis-test-plugin" })).rejects.toThrow("Failed to load plugin 'custom' from 'nonexistent-genesis-test-plugin'");
  });
});


it('resolves modules through the supplied consumer import scope', async () => {
  const instance = { id: 'scoped', category: 'tool' as const, module: 'consumer-only-plugin' };
  const importer = vi.fn(async () => ({ createPlugin: () => ({ id: instance.id, category: instance.category }) }));
  expect((await loadPlugin(instance, importer)).plugin.id).toBe('scoped');
  expect(importer).toHaveBeenCalledWith('consumer-only-plugin');
});
