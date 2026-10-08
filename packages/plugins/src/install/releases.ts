import os from "node:os";
import { z } from "zod";
import { type Platform } from "@ossl/genesis-core";
import { fetchJson, type ArchiveRelease } from "./archive.js";

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
  const releases = z.array(z.object({ files: z.array(z.object({ filename: z.string(), sha256: checksum })) }))
    .parse(await fetchJson("https://go.dev/dl/?mode=json&include=all"));
  const file = releases.flatMap(release => release.files).find(file => file.filename === filename);
  if (!file) throw new Error(`Go ${version} has no archive for ${platform}/${arch}`);
  return { url: `https://go.dev/dl/${filename}`, sha256: file.sha256, format };
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
