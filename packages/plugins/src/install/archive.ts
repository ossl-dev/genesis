import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { Readable, Transform } from "node:stream";
import { pipeline } from "node:stream/promises";
import { runCommand, type GenesisPluginContext } from "@ossl/genesis-core";

export class InstallationRecoveryError extends Error {}

export interface ArchiveRelease {
  url: string;
  sha256: string;
  format: "tar.gz" | "zip";
}

export async function fetchText(url: string): Promise<string> {
  const response = await fetch(url, { signal: AbortSignal.timeout(30_000) });
  if (!response.ok) throw new Error(`Download failed (${response.status}): ${url}`);
  return response.text();
}

export async function fetchJson(url: string): Promise<unknown> {
  return JSON.parse(await fetchText(url));
}

async function download(release: ArchiveRelease, target: string): Promise<void> {
  if (!/^[a-f0-9]{64}$/i.test(release.sha256)) throw new Error("Missing or invalid release SHA-256");
  const response = await fetch(release.url, { signal: AbortSignal.timeout(300_000) });
  if (!response.ok || !response.body) throw new Error(`Archive download failed (${response.status}): ${release.url}`);
  const hash = createHash("sha256");
  await pipeline(
    Readable.fromWeb(response.body as Parameters<typeof Readable.fromWeb>[0]),
    new Transform({ transform(chunk, _, done) { hash.update(chunk); done(null, chunk); } }),
    fs.createWriteStream(target, { flags: "wx" }),
  );
  if (hash.digest("hex") !== release.sha256.toLowerCase()) throw new Error("Archive SHA-256 mismatch");
}

export function prependPath(env: NodeJS.ProcessEnv, directory: string): void {
  const entries = (env.PATH ?? process.env.PATH ?? "").split(path.delimiter);
  env.PATH = [directory, ...entries.filter(entry => entry !== directory)].join(path.delimiter);
}

export async function singleDirectory(root: string): Promise<string> {
  const entries = await fs.promises.readdir(root, { withFileTypes: true });
  if (entries.length !== 1 || !entries[0].isDirectory()) throw new Error("Expected a single runtime directory in archive");
  return path.join(root, entries[0].name);
}

async function exists(target: string): Promise<boolean> {
  try { await fs.promises.lstat(target); return true; }
  catch (error) { if ((error as NodeJS.ErrnoException).code === "ENOENT") return false; throw error; }
}

// Keep staging and backup on the destination filesystem so promotion uses rename.
export async function installDirectory(options: {
  destination: string;
  executable?: string;
  context: GenesisPluginContext;
  prepare(stage: string): Promise<string>;
  verify(root: string): Promise<void>;
}): Promise<void> {
  const { context, prepare, verify } = options;
  const destination = path.resolve(options.destination);
  if (destination === path.parse(destination).root) throw new Error("Cannot install into a filesystem root");
  await fs.promises.mkdir(path.dirname(destination), { recursive: true });
  const lock = `${destination}.genesis-lock`;
  try { await fs.promises.mkdir(lock); }
  catch (error) {
    if ((error as NodeJS.ErrnoException).code === "EEXIST") throw new Error(`Installation already locked: ${lock}`);
    throw error;
  }
  let stage: string | undefined;
  let preserveStage = false;
  try {
    if (await exists(destination)) {
      const stat = await fs.promises.lstat(destination);
      if (!stat.isDirectory() || stat.isSymbolicLink()) throw new Error(`Installation destination must be a real directory: ${destination}`);
      if (options.executable && (await fs.promises.readdir(destination)).length && !await exists(path.join(destination, options.executable))) {
        throw new Error(`Refusing to replace a nonempty directory without ${options.executable}: ${destination}`);
      }
    }
    stage = await fs.promises.mkdtemp(path.join(path.dirname(destination), ".genesis-"));
    const previous = path.join(stage, "previous");
    const selected = await fs.promises.realpath(await prepare(stage));
    const stageReal = await fs.promises.realpath(stage);
    const relative = path.relative(stageReal, selected);
    if (!relative || relative.startsWith(`..${path.sep}`) || relative === ".." || path.isAbsolute(relative)) throw new Error("Runtime directory escapes staging");
    await verify(selected);
    const hadPrevious = await exists(destination);
    if (hadPrevious) await fs.promises.rename(destination, previous);
    let promoted = false;
    try {
      await fs.promises.rename(selected, destination);
      promoted = true;
      await verify(destination);
    } catch (error) {
      try {
        if (promoted) await fs.promises.rename(destination, path.join(stage, "failed"));
        if (hadPrevious) await fs.promises.rename(previous, destination);
      } catch (recoveryError) {
        preserveStage = true;
        throw new InstallationRecoveryError(`Installation failed: ${error}; recovery failed: ${recoveryError}; backup retained at ${previous}`);
      }
      throw error;
    }
  } finally {
    if (stage && !preserveStage) {
      await fs.promises.rm(stage, { recursive: true, force: true }).catch(error => context.logger.warn(`Could not remove staging directory ${stage}: ${error}`));
    }
    await fs.promises.rmdir(lock).catch(error => context.logger.warn(`Could not remove installation lock ${lock}: ${error}`));
  }
}

export async function installArchive(options: {
  release: ArchiveRelease;
  destination: string;
  executable?: string;
  context: GenesisPluginContext;
  select(root: string): Promise<string>;
  verify(root: string): Promise<void>;
}): Promise<void> {
  const { release, context, select } = options;
  await installDirectory({ ...options, async prepare(stage) {
    const archive = path.join(stage, `download.${release.format}`);
    const unpack = path.join(stage, "unpack");
    await download(release, archive);
    await fs.promises.mkdir(unpack);
    const commandOptions = { cwd: context.cwd, env: context.env };
    const zipped = release.format === "zip" && process.platform !== "win32";
    const listing = await runCommand(zipped ? "unzip" : "tar", zipped ? ["-Z1", archive] : ["-tf", archive], commandOptions);
    if (listing.code !== 0) throw new Error(`Cannot inspect archive: ${listing.stderr}`);
    const entries = listing.stdout.split(/\r?\n/).filter(Boolean);
    if (!entries.length || entries.some(entry => /^(?:[\\/]|[A-Za-z]:)/.test(entry) || entry.split(/[\\/]/).includes(".."))) {
      throw new Error("Archive contains unsafe or empty paths");
    }
    const extracted = await runCommand(zipped ? "unzip" : "tar", zipped ? ["-q", archive, "-d", unpack] : ["-xf", archive, "-C", unpack], commandOptions);
    if (extracted.code !== 0) throw new Error(`Archive extraction failed: ${extracted.stderr}`);
    return select(unpack);
  } });
}
