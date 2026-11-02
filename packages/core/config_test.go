package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigYAML(t *testing.T) {
	for _, id := range []string{"node", "bun", "deno", "pnpm", "yarn", "go", "java", "python", "git", "docker", "homebrew"} {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			text := "tools:\n - type: " + id + "\n   version: '${VERSION}'\nrepositories:\n - url: example\n   path: target\nscripts:\n - name: first\n   command: echo yes\n   when: before\nenv:\n HOME_PATH: '${HOME}/repo'\n SHELL_LITERAL: '$(bad)'\n"
			writeTestFile(t, filepath.Join(dir, "custom.yml"), text)
			config, err := LoadConfigWithEnv(dir, "custom.yml", func(k string) (string, bool) { return map[string]string{"VERSION": "22", "HOME": "/safe"}[k], true })
			if err != nil {
				t.Fatal(err)
			}
			if len(config.Tools) != 1 || config.Tools[0].ID != id || config.Tools[0].Options["version"] != "22" || config.Repositories[0].Folder != "target" || config.Env["HOME_PATH"] != "/safe/repo" || config.Env["SHELL_LITERAL"] != "$(bad)" {
				t.Fatalf("unexpected config %#v", config)
			}
		})
	}
}
func TestConfigMalformed(t *testing.T) {
	for _, input := range []string{"null", "7", "[]", "tools: {}", "tools: [null]", "tools: [{type: missing}]", "tools: [{type: 7}]", "tools: [{id: foo,category: tool,module: 42}]", "tools: [{id: foo,category: invalid,module: pkg}]", "tools: [{id: '',category: tool,module: pkg}]", "tools: [{id: foo,category: tool,module: ''}]", "tools: [{type: node},{type: node}]", "scripts: [{command: echo}]", "scripts: [{name: test}]", "scripts: [{name: test,command: echo,when: during}]", "repositories: [{url: repo}]", "env: []", "env: {VERSION: 22}", "env: {TOKEN: '${MISSING}'}", "{}\n---\n{}"} {
		t.Run(input, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "genesis.config.yaml")
			writeTestFile(t, path, input)
			_, err := LoadConfigWithEnv(dir, "", func(string) (string, bool) { return "", false })
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Fatalf("expected contextual error, got %v", err)
			}
		})
	}
}
func TestConfigDiscoveryAndGoHook(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadConfig(dir, ""); err == nil {
		t.Fatal("missing config accepted")
	}
	writeTestFile(t, filepath.Join(dir, "genesis.config.yaml"), "env: {MODE: yaml}")
	writeTestFile(t, filepath.Join(dir, "override.yml"), "env: {MODE: explicit}")
	config, err := LoadConfig(dir, "override.yml")
	if err != nil || config.Env["MODE"] != "explicit" {
		t.Fatal(config, err)
	}
	writeTestFile(t, filepath.Join(dir, "genesis.config.go"), "package main")
	previous := GoConfigLoader
	defer func() { GoConfigLoader = previous }()
	GoConfigLoader = func(cwd, path string) (Config, error) { return Config{Env: map[string]string{"MODE": "go"}}, nil }
	config, err = LoadConfig(dir, "")
	if err != nil || config.Env["MODE"] != "go" {
		t.Fatal(config, err)
	}
}
func TestCollectInstances(t *testing.T) {
	got := CollectInstances(Config{Tools: []Instance{{ID: "one"}}, SDKs: []Instance{{ID: "two"}}, Languages: []Instance{{ID: "three"}}})
	if len(got) != 3 || got[2].ID != "three" {
		t.Fatal(got)
	}
}
func writeTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestConfigTrimsPluginIdentityAndRejectsEmptyBranch(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "genesis.config.yaml"), "tools: [{id: ' node ', category: tool, module: ' builtin:node '}]")
	config, err := LoadConfig(dir, "")
	if err != nil || config.Tools[0].ID != "node" || config.Tools[0].Module != "builtin:node" {
		t.Fatal(config, err)
	}
	writeTestFile(t, filepath.Join(dir, "genesis.config.yaml"), "repositories: [{url: repo,folder: here,branch: ''}]")
	if _, err := LoadConfig(dir, ""); err == nil {
		t.Fatal("empty branch accepted")
	}
	if err := ValidateConfig(Config{Env: map[string]string{"VALUE": "nul\x00"}}); err == nil {
		t.Fatal("NUL accepted")
	}
}
