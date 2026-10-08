import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import { getLinuxDistribution } from "../os/linux.js";
import { createPackageInstallTask, createPackageManagerUpdateTask } from "./system-tasks.js";
import { getPlatform } from "../os/platform.js";
import { runCommand } from "../os/shell.js";

vi.mock("node:fs", () => ({ default: { readFileSync: vi.fn() } }));
vi.mock("../os/platform.js", () => ({ getPlatform: vi.fn() }));
vi.mock("../os/shell.js", () => ({ runCommand: vi.fn() }));
const getuidDescriptor = Object.getOwnPropertyDescriptor(process, "getuid");
afterEach(() => {
  if (getuidDescriptor) Object.defineProperty(process, "getuid", getuidDescriptor);
  else Reflect.deleteProperty(process, "getuid");
});
const env = { PATH: "/custom/bin" };

beforeEach(() => {
  vi.restoreAllMocks();
  vi.clearAllMocks();
  vi.mocked(getPlatform).mockReturnValue("linux");
  vi.mocked(fs.readFileSync).mockReturnValue('ID=ubuntu\nVERSION_CODENAME="noble"');
  vi.mocked(runCommand).mockReset().mockResolvedValue({ code: 0, stdout: "", stderr: "" });
  Object.defineProperty(process, "getuid", { configurable: true, writable: true, value: vi.fn(() => 0) });
});

describe("distribution detection", () => {
  it("reads exact fields without evaluating shell content", () => {
    vi.mocked(fs.readFileSync).mockReturnValue('ID=custom\nID_LIKE="fedora rhel"\nVERSION_CODENAME=$(bad)\nPRETTY_NAME="Ubuntu Arch"');
    expect(getLinuxDistribution()).toEqual({ id: "custom", like: ["fedora", "rhel"], codename: "$(bad)" });
  });
  it("falls back only when /etc/os-release is missing", () => {
    vi.mocked(fs.readFileSync).mockImplementationOnce(() => { throw Object.assign(new Error(), { code: "ENOENT" }); });
    expect(getLinuxDistribution().id).toBe("ubuntu");
    expect(fs.readFileSync).toHaveBeenLastCalledWith("/usr/lib/os-release", "utf8");
  });
  it("rejects missing IDs and unknown package managers", () => {
    vi.mocked(fs.readFileSync).mockReturnValue('PRETTY_NAME="Ubuntu"');
    expect(() => createPackageManagerUpdateTask("/work", env)).toThrow("Cannot determine");
    vi.mocked(fs.readFileSync).mockReturnValue("ID=unknown");
    expect(() => createPackageInstallTask("git", "/work", env)).toThrow("Unsupported Linux");
  });
});

it.each([
  ["ubuntu", "apt-get", ["install", "-y", "git"], "apt"],
  ["fedora", "dnf", ["install", "-y", "git"], "dnf"],
  ["rocky", "dnf", ["install", "-y", "git"], "dnf"],
  ["arch", "pacman", ["-S", "--needed", "--noconfirm", "git"], "pacman"],
  ["alpine", "apk", ["add", "git"], "apk"],
])("installs on %s without sudo as root", async (id, command, args, manager) => {
  vi.mocked(fs.readFileSync).mockReturnValue(`ID=${id}`);
  const task = createPackageInstallTask("git", "/work", env);
  expect(task.dependsOn).toEqual([`linux:package-manager:${manager}-update`]);
  expect((await task.executor()).ok).toBe(true);
  expect(runCommand).toHaveBeenCalledWith(command, args, { cwd: "/work", env });
});

it("uses ID_LIKE for derivatives and maps build dependencies", async () => {
  vi.mocked(fs.readFileSync).mockReturnValue('ID=custom\nID_LIKE="fedora rhel"');
  vi.spyOn(process, "getuid").mockReturnValue(1000);
  await createPackageInstallTask("build-essential", "/work", env).executor();
  expect(runCommand).toHaveBeenCalledWith("sudo", ["dnf", "install", "-y", "gcc", "gcc-c++", "make"], { cwd: "/work", env });
});

it("refreshes DNF metadata without upgrading installed packages", async () => {
  vi.mocked(fs.readFileSync).mockReturnValue("ID=fedora");
  await createPackageManagerUpdateTask("/work", env).executor();
  expect(runCommand).toHaveBeenCalledWith("dnf", ["makecache"], { cwd: "/work", env });
});

it.each(["arch", "windows"])("does not upgrade the host during a %s cache task", async id => {
  if (id === "windows") vi.mocked(getPlatform).mockReturnValue("windows");
  else vi.mocked(fs.readFileSync).mockReturnValue(`ID=${id}`);
  expect((await createPackageManagerUpdateTask("/work", env).executor()).ok).toBe(true);
  expect(runCommand).not.toHaveBeenCalled();
});

it("uses Homebrew independently of os-release", async () => {
  vi.mocked(getPlatform).mockReturnValue("macos");
  await createPackageInstallTask("git", "/work", env).executor();
  expect(fs.readFileSync).not.toHaveBeenCalled();
  expect(runCommand).toHaveBeenCalledWith("brew", ["install", "git"], { cwd: "/work", env });
});

it.each(["--help", "git;rm", "git\nother", ""])("rejects invalid system package %j", name => {
  expect(() => createPackageInstallTask(name, "/work", env)).toThrow("Invalid system package");
});

it("reports command failures and thrown errors", async () => {
  vi.mocked(runCommand).mockResolvedValueOnce({ code: 1, stdout: "", stderr: "locked" });
  expect(await createPackageInstallTask("git", "/work", env).executor()).toMatchObject({ ok: false, error: "locked" });
  vi.mocked(runCommand).mockRejectedValueOnce(new Error("spawn failed"));
  expect(await createPackageManagerUpdateTask("/work", env).executor()).toMatchObject({ ok: false, error: "spawn failed" });
});
