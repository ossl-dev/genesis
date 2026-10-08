import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createHash } from "node:crypto";
import { runCommand } from "@ossl/genesis-core";
import { installArchive, singleDirectory } from "./archive.js";

let root: string;
let bytes: Buffer;
let destination: string;
const context = { cwd: process.cwd(), env: { ...process.env }, logger: { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() }, taskRegistry: {} as any };

beforeEach(async () => {
  root = await fs.promises.mkdtemp(path.join(os.tmpdir(), "genesis archive test-"));
  destination = path.join(root, "installed");
  const source = path.join(root, "source", "runtime");
  await fs.promises.mkdir(source, { recursive: true });
  await fs.promises.writeFile(path.join(source, "version"), "new");
  const archive = path.join(root, "fixture.tar.gz");
  const result = await runCommand("tar", ["-czf", archive, "-C", path.dirname(source), "runtime"], { cwd: root });
  if (result.code) throw new Error(result.stderr);
  bytes = await fs.promises.readFile(archive);
  vi.stubGlobal("fetch", vi.fn(async () => new Response(new Uint8Array(bytes))));
  await fs.promises.mkdir(destination);
  await fs.promises.writeFile(path.join(destination, "version"), "old");
});
afterEach(async () => {
  vi.restoreAllMocks(); vi.unstubAllGlobals();
  await fs.promises.rm(root, { recursive: true, force: true });
});

function options() {
  return {
    destination, context, select: singleDirectory,
    release: { url: "https://example.com/runtime.tar.gz", sha256: createHash("sha256").update(bytes).digest("hex"), format: "tar.gz" as const },
    async verify(directory: string) { expect(await fs.promises.readFile(path.join(directory, "version"), "utf8")).toBe("new"); },
  };
}
const installedVersion = () => fs.promises.readFile(path.join(destination, "version"), "utf8");
async function expectClean() {
  expect((await fs.promises.readdir(root)).filter(name => name.startsWith(".genesis-") || name.endsWith(".genesis-lock"))).toEqual([]);
}

describe("staged archive installation", () => {
  it("verifies both staged and promoted files with a real archive", async () => {
    const config = options(); const verify = vi.fn(config.verify); config.verify = verify;
    await installArchive(config);
    expect(await installedVersion()).toBe("new");
    expect(verify).toHaveBeenCalledTimes(2);
    await expectClean();
  });
  it("installs into an absent destination", async () => {
    await fs.promises.rm(destination, { recursive: true });
    await installArchive(options());
    expect(await installedVersion()).toBe("new");
  });
  it("keeps the existing install on HTTP failure", async () => {
    vi.mocked(fetch).mockResolvedValue(new Response("missing", { status: 404 }));
    await expect(installArchive(options())).rejects.toThrow("404");
    expect(await installedVersion()).toBe("old");
    await expectClean();
  });
  it("rejects a checksum mismatch before extraction", async () => {
    const config = options(); config.release.sha256 = "0".repeat(64);
    await expect(installArchive(config)).rejects.toThrow("SHA-256 mismatch");
    expect(await installedVersion()).toBe("old");
    await expectClean();
  });
  it("keeps the existing install on corrupt archive contents", async () => {
    bytes = Buffer.from("not an archive");
    await expect(installArchive(options())).rejects.toThrow("Cannot inspect archive");
    expect(await installedVersion()).toBe("old");
  });
  it("keeps the existing install when staged verification fails", async () => {
    const config = options(); config.verify = async () => { throw new Error("wrong runtime"); };
    await expect(installArchive(config)).rejects.toThrow("wrong runtime");
    expect(await installedVersion()).toBe("old");
    await expectClean();
  });
  it("restores the previous install if promotion fails", async () => {
    const rename = fs.promises.rename.bind(fs.promises);
    vi.spyOn(fs.promises, "rename").mockImplementation(async (source, target) => {
      if (String(source).includes(`${path.sep}unpack${path.sep}`)) throw new Error("promotion denied");
      return rename(source, target);
    });
    await expect(installArchive(options())).rejects.toThrow("promotion denied");
    expect(await installedVersion()).toBe("old");
    await expectClean();
  });
  it("restores the previous install if promoted verification fails", async () => {
    const config = options(); const verify = config.verify;
    config.verify = async directory => { if (directory === destination) throw new Error("relocation failed"); await verify(directory); };
    await expect(installArchive(config)).rejects.toThrow("relocation failed");
    expect(await installedVersion()).toBe("old");
    await expectClean();
  });
  it("retains a recoverable backup if rollback fails", async () => {
    const config = options(); config.verify = async directory => { if (directory === destination) throw new Error("bad promotion"); };
    const rename = fs.promises.rename.bind(fs.promises);
    vi.spyOn(fs.promises, "rename").mockImplementation(async (source, target) => {
      if (String(source).endsWith(`${path.sep}previous`)) throw new Error("recovery denied");
      return rename(source, target);
    });
    await expect(installArchive(config)).rejects.toThrow("backup retained at");
    const stage = (await fs.promises.readdir(root)).find(name => name.startsWith(".genesis-"))!;
    expect(await fs.promises.readFile(path.join(root, stage, "previous", "version"), "utf8")).toBe("old");
  });
  it("rejects an occupied install lock without modifying files", async () => {
    await fs.promises.mkdir(`${destination}.genesis-lock`);
    await expect(installArchive(options())).rejects.toThrow("already locked");
    expect(await installedVersion()).toBe("old");
    expect(fetch).not.toHaveBeenCalled();
  });
  it("refuses a symlink destination", async () => {
    await fs.promises.rename(destination, `${destination}-real`);
    await fs.promises.symlink(`${destination}-real`, destination, process.platform === "win32" ? "junction" : "dir");
    await expect(installArchive(options())).rejects.toThrow("real directory");
    expect(await installedVersion()).toBe("old");
    expect(fetch).not.toHaveBeenCalled();
  });
  it("rejects an extraction selector outside staging", async () => {
    const config = options(); config.select = async () => destination;
    await expect(installArchive(config)).rejects.toThrow("escapes staging");
    expect(await installedVersion()).toBe("old");
  });
});


it('installs ZIP archives through the platform archive tool', async () => {
  bytes = Buffer.from("UEsDBBQAAAAAAO+QSF1FRONrAwAAAAMAAAAPAAAAcnVudGltZS92ZXJzaW9ubmV3UEsBAhQDFAAAAAAA75BIXUVE42sDAAAAAwAAAA8AAAAAAAAAAAAAAIABAAAAAHJ1bnRpbWUvdmVyc2lvblBLBQYAAAAAAQABAD0AAAAwAAAAAAA=", "base64");
  const config = options();
  await installArchive({ ...config, release: { ...config.release, format: 'zip' } });
  expect(await installedVersion()).toBe('new');
  await expectClean();
});


it('refuses to replace a nonempty directory belonging to something else', async () => {
  await expect(installArchive({ ...options(), executable: path.join('bin', 'go') })).rejects.toThrow('Refusing to replace a nonempty directory');
  expect(await installedVersion()).toBe('old');
  expect(fetch).not.toHaveBeenCalled();
  await expectClean();
});
