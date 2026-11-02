// Check Genesis source for formatting drift and retired runtime toolchain artifacts.
// The independent VitePress documentation project keeps its own JavaScript tooling.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := check(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func check() error {
	data, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		return err
	}
	var sources []string
	for _, item := range bytes.Split(data, []byte{0}) {
		name := string(item)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "apps/docs/") {
			continue
		}
		base := filepath.Base(name)
		ext := filepath.Ext(name)
		switch ext {
		case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts":
			return fmt.Errorf("retired source remains: %s", name)
		}
		if base == "package.json" || base == "turbo.json" || strings.HasPrefix(base, "tsconfig") && ext == ".json" || base == "bun.lock" || base == "bun.lockb" || base == "package-lock.json" || base == "pnpm-lock.yaml" || base == "yarn.lock" || strings.Contains("/"+name+"/", "/node_modules/") {
			return fmt.Errorf("retired toolchain artifact remains: %s", name)
		}
		if ext == ".go" {
			sources = append(sources, name)
		}
	}
	if len(sources) == 0 {
		return fmt.Errorf("no Go source found")
	}
	args := append([]string{"-l"}, sources...)
	output, err := exec.Command("gofmt", args...).Output()
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(output)) != 0 {
		return fmt.Errorf("unformatted Go files:\n%s", output)
	}
	return nil
}
