import fs from "node:fs";

export interface LinuxDistribution {
  id: string;
  like: string[];
  codename?: string;
}

export function getLinuxDistribution(): LinuxDistribution {
  let release: string;
  try { release = fs.readFileSync("/etc/os-release", "utf8"); }
  catch (error) {
    if ((error as NodeJS.ErrnoException).code !== "ENOENT") throw error;
    release = fs.readFileSync("/usr/lib/os-release", "utf8");
  }
  const fields: Record<string, string> = {};
  for (const line of release.split(/\r?\n/)) {
    const match = line.match(/^([A-Z_]+)=(?:"([^"\n]*)"|'([^'\n]*)'|([^\s'"#]+))\s*$/);
    if (match) fields[match[1]] = match[2] ?? match[3] ?? match[4];
  }
  if (!fields.ID || !/^[a-z0-9._-]+$/.test(fields.ID)) throw new Error("Cannot determine Linux distribution from os-release");
  return { id: fields.ID, like: (fields.ID_LIKE ?? "").split(/\s+/).filter(Boolean), codename: fields.VERSION_CODENAME ?? fields.UBUNTU_CODENAME };
}
