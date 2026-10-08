import { afterEach, expect, it, vi } from "vitest";
import os from "node:os";
import { gitSourceRelease, gitWindowsRelease } from "./git-release.js";
const hash = "a".repeat(64);
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); });
function respond(value: unknown) { vi.stubGlobal("fetch", vi.fn(async () => new Response(typeof value === "string" ? value : JSON.stringify(value)))); }
it("selects stable numeric Git releases and exact checksum entries", async () => {
  respond(`${hash}  git-2.9.9.tar.gz\n${hash}  git-2.53.1.tar.gz\n${hash}  git-2.53.0.tar.gz\n${hash}  git-2.54.0.rc0.tar.gz`);
  expect(await gitSourceRelease("latest")).toMatchObject({ version: "2.53.1", sha256: hash });
  expect(await gitSourceRelease("2.53.0")).toMatchObject({ version: "2.53.0", url: "https://www.kernel.org/pub/software/scm/git/git-2.53.0.tar.gz" });
  await expect(gitSourceRelease("2.5")).rejects.toThrow("No stable");
});
it("does not accept malformed source checksums", async () => {
  respond("bad  git-2.53.0.tar.gz");
  await expect(gitSourceRelease("2.53.0")).rejects.toThrow("No stable");
});
function release(assets: unknown[]) { return { tag_name: "v2.53.0.windows.2", draft: false, prerelease: false, assets }; }
it.each([["x64", "64-bit"], ["arm64", "arm64"]] as const)("selects standard MinGit for %s", async (arch, suffix) => {
  vi.spyOn(os, "arch").mockReturnValue(arch);
  respond(release([{ name: `MinGit-2.53.0.2-busybox-${suffix}.zip`, browser_download_url: "https://example.com/busybox", digest: `sha256:${hash}` }, { name: `MinGit-2.53.0.2-${suffix}.zip`, browser_download_url: "https://example.com/git.zip", digest: `sha256:${hash}` }]));
  expect(await gitWindowsRelease("latest")).toEqual({ version: "2.53.0", url: "https://example.com/git.zip", sha256: hash, format: "zip" });
});
it("rejects unavailable architectures and assets without checksums", async () => {
  vi.spyOn(os, "arch").mockReturnValue("arm64");
  respond(release([]));
  await expect(gitWindowsRelease("latest")).rejects.toThrow("no MinGit");
  respond(release([{ name: "MinGit-2.53.0.2-arm64.zip", browser_download_url: "https://example.com/git.zip" }]));
  await expect(gitWindowsRelease("latest")).rejects.toThrow("No SHA-256");
});
it("ignores previews and matches version components in the Windows catalog", async () => {
  respond([{ ...release([]), prerelease: true }, { ...release([]), tag_name: "v2.5.0.windows.1" }]);
  await expect(gitWindowsRelease("2.53")).rejects.toThrow("No stable");
});
it("paginates the Windows release catalog", async () => {
  vi.spyOn(os, "arch").mockReturnValue("x64");
  vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(new Response(JSON.stringify(Array(100).fill({ ...release([]), prerelease: true })))).mockResolvedValueOnce(new Response(JSON.stringify([release([{ name: "MinGit-2.53.0.2-64-bit.zip", browser_download_url: "https://example.com/git.zip", digest: `sha256:${hash}` }])]))));
  expect((await gitWindowsRelease("2.53")).version).toBe("2.53.0");
  expect(fetch).toHaveBeenLastCalledWith(expect.stringContaining("page=2"), expect.anything());
});
