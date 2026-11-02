package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCommandTimeout(t *testing.T) {
	command, args := "sh", []string{"-c", "sleep 20 & wait"}
	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/d", "/c", "ping -n 20 127.0.0.1 >NUL"}
	}
	start := time.Now()
	result, err := RunCommand(context.Background(), command, args, CommandOptions{Timeout: 100 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) || result.Code == 0 || time.Since(start) > 4*time.Second {
		t.Fatal(result, err, time.Since(start))
	}
}
func TestCommandOutputAndExit(t *testing.T) {
	command, args := "sh", []string{"-c", "printf stdout; printf stderr >&2; exit 7"}
	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/d", "/c", "echo stdout & echo stderr 1>&2 & exit /b 7"}
	}
	result, err := RunCommand(context.Background(), command, args, CommandOptions{})
	if err != nil || result.Code != 7 || !strings.Contains(result.Stdout, "stdout") || !strings.Contains(result.Stderr, "stderr") {
		t.Fatal(result, err)
	}
	result, err = RunCommand(context.Background(), "genesis-command-definitely-missing", nil, CommandOptions{})
	if err != nil || result.Code == 0 {
		t.Fatal(result, err)
	}
}
func TestCommandUsesRuntimePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "runtime-tool")
	if err := os.WriteFile(file, []byte("#!/bin/sh\nprintf '%s' \"$VALUE\""), 0755); err != nil {
		t.Fatal(err)
	}
	result, err := RunCommand(context.Background(), "runtime-tool", nil, CommandOptions{Env: map[string]string{"PATH": dir, "VALUE": "safe"}, Cwd: dir})
	if err != nil || result.Code != 0 || result.Stdout != "safe" {
		t.Fatal(result, err)
	}
	result, err = RunCommand(context.Background(), "git", nil, CommandOptions{Env: map[string]string{"PATH": dir}})
	if err != nil || result.Code == 0 {
		t.Fatal("fell back to host PATH", result, err)
	}
}
func TestLimitedOutputBuffer(t *testing.T) {
	buffer := &limitedBuffer{limit: 4}
	if n, err := buffer.Write([]byte("123456")); n != 6 || err != nil || buffer.String() != "1234" {
		t.Fatal(buffer.String(), n, err)
	}
	_, _ = buffer.Write([]byte("789"))
	if buffer.String() != "1234" {
		t.Fatal(buffer.String())
	}
}
func TestCommandStdinAndOutputLimit(t *testing.T) {
	command, args := "sh", []string{"-c", "cat; printf 123456 >&2"}
	if runtime.GOOS == "windows" {
		command = "cmd.exe"
		args = []string{"/d", "/c", "more & echo 123456 1>&2"}
	}
	result, err := RunCommand(context.Background(), command, args, CommandOptions{Stdin: strings.NewReader("abcdef\n"), OutputLimit: 3})
	if err != nil || result.Code != 0 || result.Stdout != "abc" || result.Stderr != "123" || !result.StdoutTruncated || !result.StderrTruncated {
		t.Fatal(result, err)
	}
	if _, err := RunCommand(context.Background(), command, args, CommandOptions{OutputLimit: -1}); err == nil {
		t.Fatal("negative output limit accepted")
	}
}

func TestCommandResolvesRelativeRuntimePathFromWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable; native batch fixture covers Windows")
	}
	directory := filepath.Join(t.TempDir(), "project space")
	if err := os.MkdirAll(filepath.Join(directory, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, "bin", "runtime-relative-tool")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf '%s' \"$VALUE\""), 0755); err != nil {
		t.Fatal(err)
	}
	processDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relativeDirectory, err := filepath.Rel(processDirectory, directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, cwd := range []string{directory, relativeDirectory} {
		result, err := RunCommand(context.Background(), "runtime-relative-tool", nil, CommandOptions{Cwd: cwd, Env: map[string]string{"PATH": "./bin", "VALUE": "child cwd"}})
		if err != nil || result.Code != 0 || result.Stdout != "child cwd" {
			t.Fatalf("cwd %q: %+v %v", cwd, result, err)
		}
	}
	result, err := RunCommand(context.Background(), filepath.Join(".", "bin", "runtime-relative-tool"), nil, CommandOptions{Cwd: relativeDirectory, Env: map[string]string{"PATH": "/missing", "VALUE": "explicit path"}})
	if err != nil || result.Code != 0 || result.Stdout != "explicit path" {
		t.Fatalf("explicit relative path: %+v %v", result, err)
	}
}
func TestCommandExplicitCurrentDirectoryPathAndEmptyEntries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses POSIX executable")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "runtime-local-tool")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf selected"), 0755); err != nil {
		t.Fatal(err)
	}
	result, err := RunCommand(context.Background(), "runtime-local-tool", nil, CommandOptions{Cwd: directory, Env: map[string]string{"PATH": "."}})
	if err != nil || result.Code != 0 || result.Stdout != "selected" {
		t.Fatalf("explicit dot PATH: %+v %v", result, err)
	}
	for _, path := range []string{"", string(os.PathListSeparator), string(os.PathListSeparator) + filepath.Join(directory, "missing")} {
		result, err = RunCommand(context.Background(), "runtime-local-tool", nil, CommandOptions{Cwd: directory, Env: map[string]string{"PATH": path}})
		if err != nil || result.Code == 0 {
			t.Fatalf("empty PATH entry searched working directory: %+v %v", result, err)
		}
	}
}
