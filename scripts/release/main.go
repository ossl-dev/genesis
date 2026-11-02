package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := build(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build() error {
	var sums strings.Builder
	for _, platform := range []string{"darwin", "linux", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			name := "genesis-" + platform + "-" + arch
			if platform == "windows" {
				name += ".exe"
			}
			cmd := exec.CommandContext(context.Background(), "go", "build", "-trimpath", "-ldflags=-s -w", "-o", filepath.Join("dist", name), "./apps/cli")
			cmd.Env = append(os.Environ(), "GOOS="+platform, "GOARCH="+arch, "CGO_ENABLED=0")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			file, err := os.Open(filepath.Join("dist", name))
			if err != nil {
				return err
			}
			hash := sha256.New()
			_, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(hash.Sum(nil)), name)
		}
	}
	if err := os.WriteFile(filepath.Join("dist", "SHA256SUMS"), []byte(sums.String()), 0644); err != nil {
		return err
	}
	return nil
}
