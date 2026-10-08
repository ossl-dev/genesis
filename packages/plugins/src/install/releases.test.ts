import { afterEach, describe, expect, it, vi } from "vitest";
import os from "node:os";
import { goRelease, javaRelease, releaseArchitecture } from "./releases.js";
const hash = "a".repeat(64);
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });
function json(data: unknown) { vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify(data)))); }

describe("release resolution", () => {
  it.each([['macos', 'darwin', 'arm64', 'arm64', 'tar.gz'], ['linux', 'linux', 'x64', 'amd64', 'tar.gz'], ['windows', 'windows', 'arm64', 'arm64', 'zip']] as const)("resolves Go %s archives with published checksums", async (platform, system, arch, downloadArch, format) => {
    vi.spyOn(os, "arch").mockReturnValue(arch);
    const filename = `go1.22.5.${system}-${downloadArch}.${format}`;
    json([{ files: [{ filename, sha256: hash }] }]);
    expect(await goRelease('1.22.5', platform)).toEqual({ url: `https://go.dev/dl/${filename}`, sha256: hash, format });
  });
  it("rejects nonexistent Go releases", async () => {
    json([]);
    await expect(goRelease('1.22.999', 'linux')).rejects.toThrow('no archive');
  });
  it("rejects unsupported architectures rather than selecting x64", () => {
    expect(() => releaseArchitecture('ppc64')).toThrow('Unsupported architecture');
  });
  it.each(['17', '17.0', '17.0.9', '1.8'])("resolves Java %s using a version range, not invented build URLs", async version => {
    vi.spyOn(os, "arch").mockReturnValue('arm64');
    json([{ binaries: [{ package: { link: 'https://github.com/adoptium/jdk-17.0.9+9.tar.gz', checksum: hash } }] }]);
    expect(await javaRelease(version, 'macos')).toMatchObject({ sha256: hash, format: 'tar.gz' });
    const url = new URL(String(vi.mocked(fetch).mock.calls[0][0]));
    expect(url.searchParams.get('architecture')).toBe('aarch64');
    expect(url.searchParams.get('os')).toBe('mac');
    expect(decodeURIComponent(url.pathname)).toContain(version === '1.8' ? '[8,9)' : version === '17' ? '[17,18)' : version === '17.0' ? '[17.0,17.1)' : '[17.0.9,17.0.10)');
  });
  it("rejects nonexistent Java builds", async () => {
    json([]);
    await expect(javaRelease('17.0.999', 'linux')).rejects.toThrow('no archive');
  });
  it("selects Windows Java zip archives", async () => {
    json([{ binaries: [{ package: { link: 'https://example.com/jdk.zip', checksum: hash } }] }]);
    expect((await javaRelease('17', 'windows')).format).toBe('zip');
  });
  it("reports HTTP failures rather than parsing an error page", async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('unavailable', { status: 503 })));
    await expect(javaRelease('17', 'linux')).rejects.toThrow('503');
  });
});
