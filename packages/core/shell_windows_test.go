package core

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWindowsBatchRuntimePathAndLiteralArguments(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "folder space & percent% !")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "runtime-tool.cmd")
	source := `@echo off
"%GENESIS_WINDOWS_TEST_EXE%" "-test.run=^TestWindowsBatchArgumentProbe$" -- %*
exit /b %errorlevel%
`
	if err = os.WriteFile(script, []byte(strings.ReplaceAll(source, "\n", "\r\n")), 0644); err != nil {
		t.Fatal(err)
	}
	env := NewEnvironment(nil)
	env.PrependPath(dir)
	env.Set("GENESIS_WINDOWS_TEST_EXE", executable)
	env.Set("GENESIS_WINDOWS_ARG_PROBE", "1")
	args := []string{"", "value & literal %PATH% !", "", "two words", "parentheses (literal)", "日本語", ""}
	result, err := RunCommand(context.Background(), "runtime-tool", args, CommandOptions{Env: env.Snapshot()})
	if err != nil || result.Code != 0 || result.Stderr != "" {
		t.Fatalf("%+v %v", result, err)
	}
	var actual []string
	if err = json.Unmarshal([]byte(result.Stdout), &actual); err != nil {
		t.Fatalf("invalid argument probe output %q: %v", result.Stdout, err)
	}
	if !reflect.DeepEqual(actual, args) {
		t.Fatalf("arguments changed: got %#v, want %#v", actual, args)
	}
}

func TestWindowsBatchArgumentProbe(t *testing.T) {
	if os.Getenv("GENESIS_WINDOWS_ARG_PROBE") != "1" {
		return
	}
	for index, arg := range os.Args {
		if arg == "--" {
			if err := json.NewEncoder(os.Stdout).Encode(os.Args[index+1:]); err != nil {
				os.Exit(2)
			}
			os.Exit(0)
		}
	}
	os.Exit(2)
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

func TestWindowsRelativeBatchRuntimePath(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "project space")
	if err := os.MkdirAll(filepath.Join(directory, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(directory, "bin", "runtime-relative-tool.cmd")
	if err := os.WriteFile(script, []byte("@echo off\r\necho relative selected\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	environment := NewEnvironment(nil).Snapshot()
	environment["PATH"] = "./bin"
	result, err := RunCommand(context.Background(), "runtime-relative-tool", nil, CommandOptions{Cwd: directory, Env: environment})
	if err != nil || result.Code != 0 || result.Stdout != "relative selected" {
		t.Fatal(result, err)
	}
	result, err = RunCommand(context.Background(), filepath.Join(".", "bin", "runtime-relative-tool.cmd"), nil, CommandOptions{Cwd: directory, Env: environment})
	if err != nil || result.Code != 0 || result.Stdout != "relative selected" {
		t.Fatal(result, err)
	}
}

func TestWindowsBatchRejectsUnsupportedArguments(t *testing.T) {
	for _, test := range []struct{ name, value string }{{"quote", `embedded"quote`}, {"newline", "line\nbreak"}, {"carriage return", "line\rbreak"}, {"NUL", "zero\x00byte"}} {
		t.Run(test.name, func(t *testing.T) {
			result, err := RunCommand(context.Background(), filepath.Join(t.TempDir(), "unused.cmd"), []string{test.value}, CommandOptions{})
			if err != nil || result.Code == 0 || !strings.Contains(result.Stderr, "batch command arguments cannot contain") {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
}
