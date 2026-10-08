import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts", "src/plugins/*/index.ts"],
  format: ["esm"],
  dts: true,
  sourcemap: false,
  clean: true,
  minify: true,
  target: "node18"
});

