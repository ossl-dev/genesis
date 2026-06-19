import inquirer from "inquirer";
import fs from "node:fs";
import path from "node:path";

export async function promptCreateConfig(): Promise<void> {
  const answers = await inquirer.prompt([
    { name: "name", message: "Environment name", type: "input" },
    {
      name: "format",
      message: "Config format",
      type: "list",
      choices: ["ts", "yaml"],
    },
  ]);
  await scaffoldEnvironment(
    process.cwd(),
    answers.name as string,
    answers.format as string
  );
}

export async function promptInitConfig(): Promise<void> {
  const answers = await inquirer.prompt([
    {
      name: "format",
      message: "Config format",
      type: "list",
      choices: ["ts", "yaml"],
    },
  ]);
  await scaffoldEnvironment(
    process.cwd(),
    "genesis-env",
    answers.format as string
  );
}

async function scaffoldEnvironment(
  cwd: string,
  name: string,
  format: string
): Promise<void> {
  const readmePath = path.join(cwd, "README.md");
  const pkgPath = path.join(cwd, "package.json");
  const hasPkg = fs.existsSync(pkgPath);
  if (!hasPkg) {
    const pkg = {
      name,
      private: true,
      type: "module",
      scripts: {
        genesis: "genesis apply",
      },
      devDependencies: {
        "@ossl/genesis-cli": "^0.1.0",
      },
    };
    await fs.promises.writeFile(pkgPath, JSON.stringify(pkg, null, 2));
  }
  if (!fs.existsSync(readmePath)) {
    const content = `# ${name}\n\nManaged by Genesis.\n`;
    await fs.promises.writeFile(readmePath, content);
  }
  if (format === "ts") {
    const configPath = path.join(cwd, "genesis.config.ts");
    if (!fs.existsSync(configPath)) {
      const source =
        'import { defineConfig } from "@ossl/genesis-core";\nimport { node } from "@ossl/genesis-plugins/node";\n\nexport default defineConfig({\n  tools: [\n    node({ version: "20" })\n  ],\n  sdks: [],\n  languages: [],\n  repositories: [],\n  scripts: [],\n  env: {}\n});\n';
      await fs.promises.writeFile(configPath, source);
    }
  } else {
    const configPath = path.join(cwd, "genesis.config.yaml");
    if (!fs.existsSync(configPath)) {
      const source =
        "tools: []\nsdks: []\nlanguages: []\nrepositories: []\nscripts: []\nenv: {}\n";
      await fs.promises.writeFile(configPath, source);
    }
  }
}
