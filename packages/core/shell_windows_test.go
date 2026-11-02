package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsBatchRuntimePathAndLiteralArguments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "folder space & percent% !")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "runtime-tool.cmd")
	if err := os.WriteFile(script, []byte("@echo off\r\necho %~1\r\nexit /b 0\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	env := NewEnvironment(nil)
	env.PrependPath(dir)
	arg := "value & literal %PATH% !"
	result, err := RunCommand(context.Background(), "runtime-tool", []string{arg}, CommandOptions{Env: env.Snapshot()})
	if err != nil || result.Code != 0 || strings.TrimSpace(result.Stdout) != arg {
		t.Fatal(result, err)
	}
}
func TestWindowsEnvironmentKeysCaseInsensitive(t *testing.T) {
	env := NewEnvironment(map[string]string{"Path": "original", "Home": "home"})
	env.Set("PATH", "new")
	if env.Get("path") != "new" || env.Get("HOME") != "home" {
		t.Fatal(env.Snapshot())
	}
	env.PrependPath("managed")
	if strings.Contains(env.Get("Path"), "original") {
		t.Fatal("stale PATH duplicate", env.Snapshot())
	}
	paths := 0
	for key := range env.Snapshot() {
		if strings.EqualFold(key, "path") {
			paths++
		}
	}
	if paths != 1 {
		t.Fatal("PATH duplicates", env.Snapshot())
	}
}
