//go:build windows

package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func configureProcess(cmd *exec.Cmd) error {
	extension := strings.ToLower(filepath.Ext(cmd.Path))
	if extension == ".cmd" || extension == ".bat" {
		if cmd.Env == nil {
			cmd.Env = os.Environ()
		}
		args := append([]string{cmd.Path}, cmd.Args[1:]...)
		parts := make([]string, len(args))
		for i, arg := range args {
			if strings.ContainsAny(arg, "\"\r\n\x00") {
				return fmt.Errorf("batch command arguments cannot contain quotes, newlines or NUL")
			}
			key := "GENESIS_BATCH_ARG_" + strconv.Itoa(i)
			cmd.Env = append(cmd.Env, key+"="+arg)
			parts[i] = "\"%" + key + "%\""
		}
		shell := os.Getenv("ComSpec")
		if shell == "" {
			shell = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
		}
		cmd.Path = shell
		cmd.Args = []string{shell, "/d", "/s", "/v:off", "/c", strings.Join(parts, " ")}
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: syscall.EscapeArg(shell) + " /d /s /v:off /c \"" + strings.Join(parts, " ") + "\""}
	}
	cmd.Cancel = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exec.CommandContext(ctx, "taskkill.exe", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		return cmd.Process.Kill()
	}
	return nil
}
