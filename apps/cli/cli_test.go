package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func runTest(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return runAt(t, t.TempDir(), args...)
}
func runAt(t *testing.T, cwd string, args ...string) (string, string, error) {
	t.Helper()
	var out, errs bytes.Buffer
	err := Run(context.Background(), args, cwd, strings.NewReader(""), &out, &errs)
	return out.String(), errs.String(), err
}
func TestHelpAndFlags(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"--version"}, {"apply", "--help"}} {
		_, _, err := runTest(t, args...)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"unknown"}, {"apply", "--parallel=0"}, {"apply", "--json"}, {"apply", "env-id", "--dry-run"}, {"apply", "--cloud"}, {"list", "--format=xml"}, {"init", "--format=ts"}, {"diff", "oops"}} {
		if _, _, err := runTest(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
func TestListPluginsJSON(t *testing.T) {
	out, _, err := runTest(t, "list-plugins", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var plugins []map[string]any
	if json.Unmarshal([]byte(out), &plugins) != nil || len(plugins) != 11 {
		t.Fatal(out)
	}
}
func TestScaffoldPreservesFiles(t *testing.T) {
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "README.md"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := runAt(t, cwd, "init", "--format=yaml")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(cwd, "README.md"))
	if string(data) != "keep" {
		t.Fatal(string(data))
	}
	if _, err = os.Stat(filepath.Join(cwd, "package.json")); !os.IsNotExist(err) {
		t.Fatal("legacy manifest created")
	}
	out, _, err := runAt(t, cwd, "apply", "--dry-run", "--json")
	if err != nil || !json.Valid([]byte(out)) {
		t.Fatalf("%s %v", out, err)
	}
}
func TestExplicitConfigAndNoDryRunEffects(t *testing.T) {
	cwd := t.TempDir()
	config := `tools:
  - type: git
scripts:
  - name: forbidden
    command: "exit 93"
env:
  SECRET_VALUE: "do-not-print"
`
	os.WriteFile(filepath.Join(cwd, "custom.yml"), []byte(config), 0644)
	out, _, err := runAt(t, cwd, "apply", "--config", "custom.yml", "--dry-run", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) || strings.Contains(out, "do-not-print") {
		t.Fatal(out)
	}
}
func TestRealScriptAndFailureExit(t *testing.T) {
	cwd := t.TempDir()
	command := "printf hello > output"
	if runtime.GOOS == "windows" {
		command = "echo hello>output"
	}
	config := "scripts:\n  - name: write\n    command: " + command + "\n"
	os.WriteFile(filepath.Join(cwd, "genesis.config.yaml"), []byte(config), 0644)
	_, _, err := runAt(t, cwd, "apply")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cwd, "output"))
	if err != nil || !strings.Contains(string(data), "hello") {
		t.Fatalf("%q %v", data, err)
	}
	os.WriteFile(filepath.Join(cwd, "genesis.config.yaml"), []byte("scripts:\n  - name: fail\n    command: exit 17\n"), 0644)
	if _, _, err = runAt(t, cwd, "apply"); err == nil {
		t.Fatal("expected failed script")
	}
}
func TestPrivateLoginAndCleanListJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	out, _, err := runTest(t, "login", "--token", "secret-token")
	if err != nil || strings.Contains(out, "secret-token") {
		t.Fatalf("%s %v", out, err)
	}
	info, err := os.Stat(filepath.Join(home, ".genesis", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
	cache := filepath.Join(home, ".genesis", "cache")
	os.MkdirAll(cache, 0700)
	os.WriteFile(filepath.Join(cache, "broken.json"), []byte("{"), 0600)
	os.WriteFile(filepath.Join(cache, "ok.json"), []byte(`{"name":"Valid"}`), 0600)
	out, errs, err := runTest(t, "list", "--cloud", "--format=json")
	if err != nil || !json.Valid([]byte(out)) || !strings.Contains(errs, "not implemented") {
		t.Fatalf("%q %q %v", out, errs, err)
	}
}
func TestGoScaffoldLoads(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes Go compiler")
	}
	cwd := t.TempDir()
	_, _, err := runAt(t, cwd, "init", "--format=go")
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := runAt(t, cwd, "apply", "--dry-run", "--json")
	if err != nil || !json.Valid([]byte(out)) {
		t.Fatalf("%s %v", out, err)
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("GENESIS_CLI_TEST_PLUGIN") == "1" {
		var request struct {
			Method string `json:"method"`
		}
		if json.NewDecoder(os.Stdin).Decode(&request) != nil {
			os.Exit(3)
		}
		response := map[string]any{"protocol": 1}
		switch request.Method {
		case "describe":
			response["methods"] = []string{"detect", "validate"}
		case "detect":
			response["detect"] = map[string]any{"ok": true, "details": "client present"}
		case "validate":
			response["validate"] = map[string]any{"ok": false, "message": "daemon unavailable: start service"}
		}
		json.NewEncoder(os.Stdout).Encode(response)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestDoctorReportsValidationMessage(t *testing.T) {
	t.Setenv("GENESIS_CLI_TEST_PLUGIN", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	config := map[string]any{"tools": []map[string]any{{"id": "service", "category": "tool", "module": "exec:" + executable}}}
	data, _ := json.Marshal(config)
	if err = os.WriteFile(filepath.Join(cwd, "genesis.config.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	out, _, err := runAt(t, cwd, "doctor")
	if err == nil || !strings.Contains(out, "daemon unavailable: start service") || !strings.Contains(out, "error") {
		t.Fatalf("%q %v", out, err)
	}
	out, _, err = runAt(t, cwd, "doctor", "--json")
	if err == nil || !json.Valid([]byte(out)) {
		t.Fatalf("%q %v", out, err)
	}
}
