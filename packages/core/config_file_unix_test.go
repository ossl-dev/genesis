//go:build !windows

package core

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestConfigRejectsFIFOWithoutWaitingForWriter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genesis.config.yaml")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(dir, ""); err == nil {
		t.Fatal("FIFO config accepted")
	}
	if file, err := openRegularConfig(path); err == nil {
		_ = file.Close()
		t.Fatal("descriptor check accepted FIFO")
	}
}
func TestConfigAllowsSymlinkToRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.yaml")
	writeTestFile(t, path, "env: {MODE: regular}")
	if err := os.Symlink(path, filepath.Join(dir, "genesis.config.yaml")); err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfig(dir, "")
	if err != nil || config.Env["MODE"] != "regular" {
		t.Fatal(config, err)
	}
}
