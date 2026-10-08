import { z } from "zod";
import { matchesVersion } from "../options.js";
import { fetchJson, fetchText, type ArchiveRelease } from "./archive.js";
import { releaseArchitecture } from "./releases.js";

type GitRelease = ArchiveRelease & { version: string };

export async function gitSourceRelease(requested: string): Promise<GitRelease> {
  const base = "https://www.kernel.org/pub/software/scm/git";
  const sums = await fetchText(`${base}/sha256sums.asc`);
  const releases = sums.split(/\r?\n/).flatMap(line => {
    const match = line.match(/^([a-f0-9]{64})\s+git-(\d+\.\d+\.\d+)\.tar\.gz$/i);
    return match ? [{ sha256: match[1], version: match[2] }] : [];
  }).filter(release => requested === "latest" || matchesVersion(release.version, requested));
  releases.sort((a, b) => {
    const left = a.version.split(".").map(Number);
    const right = b.version.split(".").map(Number);
    return right[0] - left[0] || right[1] - left[1] || right[2] - left[2];
  });
  const release = releases[0];
  if (!release) throw new Error(`No stable Git source release matches ${requested}`);
  return { ...release, url: `${base}/git-${release.version}.tar.gz`, format: "tar.gz" };
}

const windowsRelease = z.object({
  tag_name: z.string(), draft: z.boolean(), prerelease: z.boolean(),
  assets: z.array(z.object({ name: z.string(), browser_download_url: z.url(), digest: z.string().nullable().optional() })),
});

export async function gitWindowsRelease(requested: string): Promise<GitRelease> {
  const arch = releaseArchitecture();
  const base = "https://api.github.com/repos/git-for-windows/git/releases";
  for (let page = 1; page <= 5; page++) {
    const releases = requested === "latest"
      ? [windowsRelease.parse(await fetchJson(`${base}/latest`))]
      : z.array(windowsRelease).parse(await fetchJson(`${base}?per_page=100&page=${page}`));
    for (const release of releases) {
      const version = release.tag_name.match(/^v(\d+\.\d+\.\d+)\.windows\.\d+$/)?.[1];
      if (!version || release.draft || release.prerelease || (requested !== "latest" && !matchesVersion(version, requested))) continue;
      const suffix = arch === "arm64" ? "arm64" : "64-bit";
      const asset = release.assets.find(asset => new RegExp(`^MinGit-[0-9.]+-${suffix}\\.zip$`).test(asset.name));
      if (!asset) throw new Error(`Git ${version} has no MinGit archive for Windows/${arch}`);
      const sha256 = asset.digest?.match(/^sha256:([a-f0-9]{64})$/i)?.[1];
      if (!sha256) throw new Error(`No SHA-256 published for ${asset.name}; choose a newer Git for Windows release`);
      return { version, url: asset.browser_download_url, sha256, format: "zip" };
    }
    if (requested === "latest" || releases.length < 100) break;
  }
  throw new Error(`No stable Git for Windows release matches ${requested} in the release catalog`);
}
