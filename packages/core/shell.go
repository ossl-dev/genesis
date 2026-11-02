package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// limitedBuffer drains output while retaining only its prefix, bounding child-process memory.
type limitedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Len() int       { return b.buffer.Len() }
func (b *limitedBuffer) String() string { return b.buffer.String() }
func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if n > b.limit-b.Len() {
		b.truncated = true
	}
	if left := b.limit - b.Len(); left > 0 {
		if len(p) > left {
			p = p[:left]
		}
		_, _ = b.buffer.Write(p)
	}
	return n, nil
}
func RunCommand(ctx context.Context, command string, args []string, options CommandOptions) (CommandResult, error) {
	if options.OutputLimit < 0 {
		return CommandResult{Code: 1}, fmt.Errorf("output limit must be nonnegative")
	}
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}
	if err := ctx.Err(); err != nil {
		return CommandResult{Code: 1}, err
	}
	// Resolve against the runtime PATH, not the parent's PATH. Later plugins rely on it.
	if !strings.ContainsAny(command, "/\\") && options.Env != nil {
		if found := lookPath(command, options.Env); found != "" {
			command = found
		} else {
			return CommandResult{Code: 1, Stderr: fmt.Sprintf("command %s: %v", command, exec.ErrNotFound)}, nil
		}
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = options.Cwd
	cmd.Stdin = options.Stdin
	cmd.WaitDelay = 2 * time.Second
	if options.Env != nil {
		keys := make([]string, 0, len(options.Env))
		for key := range options.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			cmd.Env = append(cmd.Env, key+"="+options.Env[key])
		}
	}
	if err := configureProcess(cmd); err != nil {
		return CommandResult{Code: 1, Stderr: err.Error()}, nil
	}
	limit := options.OutputLimit
	if limit == 0 {
		limit = 4 << 20
	}
	stdout := &limitedBuffer{limit: limit}
	stderr := &limitedBuffer{limit: limit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	result := CommandResult{Stdout: strings.TrimSpace(stdout.String()), Stderr: strings.TrimSpace(stderr.String()), StdoutTruncated: stdout.truncated, StderrTruncated: stderr.truncated}
	if err == nil {
		return result, nil
	}
	result.Code = 1
	if exit := new(exec.ExitError); errors.As(err, &exit) {
		result.Code = exit.ExitCode()
		if result.Code < 0 {
			result.Code = 1
		}
		if ctx.Err() == nil {
			return result, nil
		}
	}
	if ctx.Err() != nil {
		if result.Stderr == "" {
			result.Stderr = ctx.Err().Error()
		}
		return result, fmt.Errorf("command %s: %w", command, ctx.Err())
	}
	result.Stderr = strings.TrimSpace(result.Stderr + "\n" + fmt.Sprintf("command %s: %v", command, err))
	return result, nil
}
func lookPath(command string, env map[string]string) string {
	p := env["PATH"]
	if runtime.GOOS == "windows" {
		for k, v := range env {
			if strings.EqualFold(k, "PATH") {
				p = v
			}
		}
	}
	extensions := []string{""}
	if runtime.GOOS == "windows" && filepath.Ext(command) == "" {
		extensions = []string{".exe", ".com", ".bat", ".cmd"}
	}
	for _, dir := range filepath.SplitList(p) {
		if dir == "" {
			continue
		}
		for _, ext := range extensions {
			candidate := filepath.Join(dir, command+ext)
			if found, err := exec.LookPath(candidate); err == nil {
				return found
			}
		}
	}
	return ""
}
