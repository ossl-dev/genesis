import { z } from "zod";

const version = z.string().regex(/^\d+(?:\.\d+){0,2}$/, "Expected a numeric version such as 22 or 3.11.9");
const packages = z.array(z.string().regex(/^[^-\s][^\s]*$/, "Expected a package name, not a command option"));

export const optionSchemas = {
  node: z.object({
    version,
    use_nvm: z.boolean().default(true),
    global_packages: packages.optional(),
  }).strict(),
  python: z.object({ version }).strict(),
  go: z.object({ version: z.string().regex(/^\d+\.\d+\.\d+$/, "Go archive installation requires a full version such as 1.22.5") }).strict(),
  java: z.object({ version, distribution: z.enum(["openjdk", "oracle"]).default("openjdk") }).strict(),
  git: z.object({
    version: z.union([version, z.literal("latest")]).default("latest"),
    install_method: z.enum(["package", "source", "binary"]).default("package"),
  }).strict().default({ version: "latest", install_method: "package" }),
  docker: z.object({
    version: z.union([version, z.literal("latest")]).default("latest"),
    include_compose: z.boolean().default(true),
    install_desktop: z.boolean().default(false),
  }).strict().default({ version: "latest", include_compose: true, install_desktop: false }),
  homebrew: z.object({
    update_packages: z.boolean().default(true),
    install_cask: z.boolean().default(true),
    add_to_path: z.boolean().default(true),
    global_packages: packages.optional(),
  }).strict().default({ update_packages: true, install_cask: true, add_to_path: true }),
};
