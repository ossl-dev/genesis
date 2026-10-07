import { describe, expect, it } from "vitest";
import { optionSchemas } from "./options.js";

describe("plugin config validation", () => {
  it("applies the same defaults to YAML options as the helper factories", () => {
    expect(optionSchemas.node.parse({ version: "22" })).toEqual({ version: "22", use_nvm: true });
    expect(optionSchemas.java.parse({ version: "17" })).toEqual({ version: "17", distribution: "openjdk" });
    expect(optionSchemas.git.parse(undefined)).toEqual({ version: "latest", install_method: "package" });
    expect(optionSchemas.docker.parse({})).toEqual({ version: "latest", include_compose: true, install_desktop: false });
  });

  it.each([undefined, {}, { version: 22 }, { version: "22; echo unsafe" }, { version: "22", use_nvm: "false" }, { version: "22", use_nvmm: true }])("rejects malformed Node options %j", options => {
    expect(() => optionSchemas.node.parse(options)).toThrow();
  });

  it("rejects incomplete archive versions and package flags", () => {
    expect(() => optionSchemas.go.parse({ version: "1.22" })).toThrow("full version");
    expect(() => optionSchemas.homebrew.parse({ global_packages: ["--force"] })).toThrow();
  });
});
