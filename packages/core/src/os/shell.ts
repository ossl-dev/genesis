import { execa } from "execa";

export interface RunCommandOptions {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  timeout?: number;
}

export interface RunCommandResult {
  code: number;
  stdout: string;
  stderr: string;
}

export async function runCommand(
  command: string,
  args: string[],
  options?: RunCommandOptions
): Promise<RunCommandResult> {
  const subprocess = await execa(command, args, {
    cwd: options?.cwd,
    env: options?.env,
    reject: false,
    timeout: options?.timeout,
  });
  return {
    code: subprocess.exitCode ?? 1,
    stdout: subprocess.stdout,
    stderr: subprocess.stderr || (subprocess.timedOut ? `Command timed out after ${options?.timeout}ms` : ""),
  };
}

