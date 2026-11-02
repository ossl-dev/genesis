package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

func platformOutput(result core.CommandResult) string {
	if result.Stdout != "" {
		return result.Stdout
	}
	return result.Stderr
}
func platformFailure(result core.CommandResult, err error) string {
	if err != nil {
		return err.Error()
	}
	if output := platformOutput(result); output != "" {
		return strings.TrimSpace(output)
	}
	return fmt.Sprintf("exit %d", result.Code)
}
func platformCheck(runtime *core.Runtime, command string, args ...string) error {
	result, err := runtime.Run(command, args...)
	if err != nil || result.Code != 0 {
		return fmt.Errorf("%s failed: %s", command, platformFailure(result, err))
	}
	return nil
}
func platformContextError(runtime *core.Runtime) error {
	if runtime.Context != nil {
		return runtime.Context.Err()
	}
	return nil
}
func platformRunAt(runtime *core.Runtime, cwd, command string, args ...string) (core.CommandResult, error) {
	runtime.Ensure()
	return runtime.Runner(runtime.Context, command, args, core.CommandOptions{Cwd: cwd, Env: runtime.Env.Snapshot()})
}
func platformVersionCompare(left, right string) int {
	a, b := strings.Split(left, "."), strings.Split(right, ".")
	for i := 0; i < len(a) || i < len(b); i++ {
		x, y := 0, 0
		if i < len(a) {
			x, _ = strconv.Atoi(a[i])
		}
		if i < len(b) {
			y, _ = strconv.Atoi(b[i])
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

var platformNaturalParts = regexp.MustCompile(`[0-9]+|[^0-9]+`)

func platformNaturalCompare(left, right string) int {
	a, b := platformNaturalParts.FindAllString(left, -1), platformNaturalParts.FindAllString(right, -1)
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] == b[i] {
			continue
		}
		x, y := strings.TrimLeft(a[i], "0"), strings.TrimLeft(b[i], "0")
		if a[i][0] >= '0' && a[i][0] <= '9' && b[i][0] >= '0' && b[i][0] <= '9' {
			if len(x) < len(y) {
				return -1
			}
			if len(x) > len(y) {
				return 1
			}
			return strings.Compare(x, y)
		}
		return strings.Compare(a[i], b[i])
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}
func platformCopyTree(ctx context.Context, source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported Git source entry: %s", relative)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
