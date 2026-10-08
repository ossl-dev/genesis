import os from "node:os";
import fs from "node:fs";
import { z } from "zod";
import { type Platform } from "@ossl/genesis-core";
import { fetchJson, fetchText, type ArchiveRelease } from "./archive.js";

export function releaseArchitecture(arch = os.arch()): "x64" | "arm64" {
  if (arch !== "x64" && arch !== "arm64") throw new Error(`Unsupported architecture: ${arch}`);
  return arch;
}

const checksum = z.string().regex(/^[a-f0-9]{64}$/i);
const downloadPackage = z.object({ link: z.url(), checksum });

export async function goRelease(version: string, platform: Platform): Promise<ArchiveRelease> {
  const arch = releaseArchitecture() === "x64" ? "amd64" : "arm64";
  const system = platform === "macos" ? "darwin" : platform;
  const format = platform === "windows" ? "zip" : "tar.gz";
  const filename = `go${version}.${system}-${arch}.${format}`;
  const releases = z.array(z.object({ files: z.array(z.object({ filename: z.string(), sha256: z.string() })) }))
    .parse(await fetchJson("https://go.dev/dl/?mode=json&include=all"));
  const file = releases.flatMap(release => release.files).find(file => file.filename === filename);
  if (!file) throw new Error(`Go ${version} has no archive for ${platform}/${arch}`);
  return { url: `https://go.dev/dl/${filename}`, sha256: checksum.parse(file.sha256), format };
}

export function normalizeJavaVersion(version: string): string {
  return version.replace(/^1\./, "").replace("_", ".");
}

export async function javaRelease(version: string, platform: Platform): Promise<ArchiveRelease> {
  const arch = releaseArchitecture() === "arm64" ? "aarch64" : "x64";
  const system = platform === "macos" ? "mac" : platform;
  const normalized = normalizeJavaVersion(version);
  const upper = normalized.split(".");
  upper[upper.length - 1] = String(Number(upper[upper.length - 1]) + 1);
  const range = encodeURIComponent(`[${normalized},${upper.join(".")})`);
  const query = new URLSearchParams({ architecture: arch, os: system, image_type: "jdk", jvm_impl: "hotspot", vendor: "eclipse", release_type: "ga", project: "jdk", sort_method: "DATE", sort_order: "DESC", page_size: "1" });
  const releases = z.array(z.object({ binaries: z.array(z.object({ package: downloadPackage })) }))
    .parse(await fetchJson(`https://api.adoptium.net/v3/assets/version/${range}?${query}`));
  const asset = releases[0]?.binaries[0]?.package;
  if (!asset) throw new Error(`Temurin Java ${version} has no archive for ${platform}/${arch}`);
  return { url: asset.link, sha256: asset.checksum, format: platform === "windows" ? "zip" : "tar.gz" };
}

export function isMusl(): boolean {
  if (process.platform !== "linux") return false;
  if (fs.existsSync("/etc/alpine-release")) return true;
  const report = process.report?.getReport();
  if (!report || typeof report !== "object" || !("header" in report) || !report.header || typeof report.header !== "object") return false;
  return !("glibcVersionRuntime" in report.header);
}

function fileChecksum(text: string, filename: string): string {
  for (const line of text.split(/\r?\n/)) {
    const match = line.trim().match(/^([a-f0-9]{64})(?:\s+\*?(.+))?$/i);
    if (match && (!match[2] || match[2] === filename)) return match[1];
  }
  throw new Error(`No SHA-256 published for ${filename}`);
}

export async function bunRelease(version: string, platform: Platform, libc: "auto" | "glibc" | "musl" = "auto"): Promise<ArchiveRelease> {
  const arch = releaseArchitecture();
  const system = platform === "macos" ? "darwin" : platform;
  const musl = platform === "linux" && (libc === "musl" || (libc === "auto" && isMusl()));
  const filename = `bun-${system}-${arch === "arm64" ? "aarch64" : "x64"}${musl ? "-musl" : ""}${arch === "x64" ? "-baseline" : ""}.zip`;
  const release = z.object({ assets: z.array(z.object({ name: z.string(), browser_download_url: z.url(), digest: z.string().nullable().optional() })) })
    .parse(await fetchJson(`https://api.github.com/repos/oven-sh/bun/releases/tags/bun-v${version}`));
  const asset = release.assets.find(asset => asset.name === filename);
  if (!asset) throw new Error(`Bun ${version} has no archive for ${platform}/${arch}${musl ? "/musl" : ""}`);
  let sha256 = asset.digest?.match(/^sha256:([a-f0-9]{64})$/i)?.[1];
  if (!sha256) {
    const sums = release.assets.find(asset => asset.name === "SHASUMS256.txt");
    if (!sums) throw new Error(`No SHA-256 published for ${filename}`);
    sha256 = fileChecksum(await fetchText(sums.browser_download_url), filename);
  }
  return { url: asset.browser_download_url, sha256, format: "zip" };
}

export async function denoRelease(version: string, platform: Platform): Promise<ArchiveRelease> {
  const arch = releaseArchitecture() === "arm64" ? "aarch64" : "x86_64";
  if (platform === "linux" && isMusl()) throw new Error("Deno's official Linux archives require glibc; musl is unsupported");
  const target = platform === "macos" ? "apple-darwin" : platform === "windows" ? "pc-windows-msvc" : "unknown-linux-gnu";
  const filename = `deno-${arch}-${target}.zip`;
  const url = `https://github.com/denoland/deno/releases/download/v${version}/${filename}`;
  return { url, sha256: fileChecksum(await fetchText(`${url}.sha256sum`), filename), format: "zip" };
}


export async function nodeRelease(requested: string, platform: Platform): Promise<ArchiveRelease> {
  const arch = releaseArchitecture();
  if (platform === "linux" && isMusl()) throw new Error("Node's official Linux archives require glibc; use a compatible runtime on musl");
  const releases = z.array(z.object({ version: z.string() })).parse(await fetchJson("https://nodejs.org/dist/index.json"));
  const release = releases.find(release => {
    const version = release.version.replace(/^v/, "");
    return /^\d+\.\d+\.\d+$/.test(version) && requested.split(".").every((part, index) => part === version.split(".")[index]);
  });
  if (!release) throw new Error(`No Node release matches ${requested}`);
  const version = release.version;
  const system = platform === "macos" ? "darwin" : platform === "windows" ? "win" : "linux";
  const format = platform === "windows" ? "zip" : "tar.gz";
  const filename = `node-${version}-${system}-${arch}.${format}`;
  const base = `https://nodejs.org/dist/${version}`;
  return { url: `${base}/${filename}`, sha256: fileChecksum(await fetchText(`${base}/SHASUMS256.txt`), filename), format };
}
