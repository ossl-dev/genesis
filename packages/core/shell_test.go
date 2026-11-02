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
