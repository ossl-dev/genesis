package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

type platformCommand struct {
	Command string
	Args    []string
	Options core.CommandOptions
}

func platformTestRuntime(t *testing.T, platform string, run func(string, []string, core.CommandOptions) (core.CommandResult, error)) (*core.Runtime, *[]platformCommand) {
	t.Helper()
	commands := []platformCommand{}
	runtime := &core.Runtime{Context: context.Background(), Platform: platform, Arch: "arm64", Cwd: t.TempDir(), Env: core.NewEnvironment(map[string]string{"HOME": t.TempDir(), "USERPROFILE": t.TempDir(), "PATH": "/host/bin"})}
	runtime.Ensure()
	runtime.Runner = func(ctx context.Context, command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
		commands = append(commands, platformCommand{command, append([]string{}, args...), options})
		if ctx.Err() != nil {
			return core.CommandResult{Code: 1}, ctx.Err()
		}
		if run == nil {
			return core.CommandResult{Code: 1, Stderr: "missing"}, nil
		}
		return run(command, args, options)
	}
	return runtime, &commands
}
func platformTestInstance(t *testing.T, plugin core.Plugin, options map[string]any) core.Instance {
	t.Helper()
	parsed, err := plugin.ParseOptions(options)
	if err != nil {
		t.Fatal(err)
	}
	return core.Instance{ID: plugin.ID, Category: plugin.Category, Options: parsed}
}
func TestHomebrewDetect(t *testing.T) {
	for _, test := range []struct {
		name, output string
		code         int
		ok           bool
	}{{"installed", "Homebrew 4.3.0", 0, true}, {"missing", "", 1, false}, {"malformed", "brew", 0, false}} {
		t.Run(test.name, func(t *testing.T) {
			runtime, _ := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
				return core.CommandResult{Code: test.code, Stdout: test.output}, nil
			})
			detected, err := Homebrew().Detect(runtime, platformTestInstance(t, Homebrew(), nil))
			if err != nil || detected.OK != test.ok {
				t.Fatalf("%+v %v", detected, err)
			}
		})
	}
}
func TestHomebrewBootstrapBeforeTasks(t *testing.T) {
	for _, arch := range []string{"arm64", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			installed := false
			runtime, commands := platformTestRuntime(t, "macos", func(command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				if command == "bash" {
					installed = true
					return core.CommandResult{}, nil
				}
				if installed {
					return core.CommandResult{Stdout: "Homebrew 4.3.0"}, nil
				}
				return core.CommandResult{Code: 1}, nil
			})
			runtime.Arch = arch
			instance := platformTestInstance(t, Homebrew(), nil)
			if err := Homebrew().Prepare(runtime, instance); err != nil {
				t.Fatal(err)
			}
			prefix := filepath.Join("/opt", "homebrew", "bin")
			if arch == "amd64" {
				prefix = filepath.Join("/usr", "local", "bin")
			}
			paths := filepath.SplitList(runtime.Env.Get("PATH"))
			if len(paths) == 0 || paths[0] != prefix {
				t.Fatal(runtime.Env.Get("PATH"))
			}
			if len(*commands) != 3 || (*commands)[0].Command != "brew" || (*commands)[1].Command != "bash" || (*commands)[2].Command != "brew" {
				t.Fatal(*commands)
			}
			if !strings.Contains((*commands)[1].Args[1], "pipefail") {
				t.Fatal("bootstrap pipeline must propagate download failures")
			}
		})
	}
}
func TestHomebrewBootstrapVerificationFailure(t *testing.T) {
	runtime, _ := platformTestRuntime(t, "macos", func(command string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
		if command == "bash" {
			return core.CommandResult{}, nil
		}
		return core.CommandResult{Code: 1}, nil
	})
	if err := Homebrew().Prepare(runtime, platformTestInstance(t, Homebrew(), nil)); err == nil || !strings.Contains(err.Error(), "failed verification") {
		t.Fatal(err)
	}
}
func TestHomebrewUpdateFailuresAndFlags(t *testing.T) {
	for _, failure := range []string{"update", "upgrade", "upgrade --cask"} {
		t.Run(failure, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, "macos", func(_ string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				if args[0] == "--version" {
					return core.CommandResult{Stdout: "Homebrew 4.3.0"}, nil
				}
				if strings.Join(args, " ") == failure {
					return core.CommandResult{Code: 1, Stderr: "network failed"}, nil
				}
				return core.CommandResult{}, nil
			})
			result, err := Homebrew().Apply(runtime, platformTestInstance(t, Homebrew(), nil))
			if err != nil || result.OK || !result.DidChange || !strings.Contains(result.Details, "network failed") {
				t.Fatalf("%+v %v", result, err)
			}
			expected := 2
			if failure == "upgrade" {
				expected = 3
			}
			if failure == "upgrade --cask" {
				expected = 4
			}
			if len(*commands) != expected {
				t.Fatal(*commands)
			}
		})
	}
	runtime, commands := platformTestRuntime(t, "macos", func(_ string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
		if args[0] == "--version" {
			return core.CommandResult{Stdout: "Homebrew 4.3.0"}, nil
		}
		return core.CommandResult{}, nil
	})
	result, err := Homebrew().Apply(runtime, platformTestInstance(t, Homebrew(), map[string]any{"update_packages": false}))
	if err != nil || !result.OK || len(*commands) != 2 {
		t.Fatalf("%+v %v %+v", result, err, *commands)
	}
}
func TestHomebrewPostApplyPackagesAndUnsupportedPlatforms(t *testing.T) {
	runtime, commands := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
		return core.CommandResult{}, nil
	})
	instance := platformTestInstance(t, Homebrew(), map[string]any{"global_packages": []any{"wget", "jq"}})
	if err := Homebrew().PostApply(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual((*commands)[0].Args, []string{"install", "wget", "jq"}) {
		t.Fatal(*commands)
	}
	for _, platform := range []string{"linux", "windows"} {
		runtime.Platform = platform
		result, err := Homebrew().Apply(runtime, instance)
		if err != nil || result.OK || result.DidChange || !strings.Contains(result.Details, "not supported") {
			t.Fatalf("%+v %v", result, err)
		}
	}
}
func TestPlatformCopyTreePreservesFilesAndCancellation(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "Makefile"), []byte("all:"), 0755); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "source")
	if err := platformCopyTree(context.Background(), source, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "Makefile"))
	if err != nil || string(data) != "all:" {
		t.Fatalf("%s %v", data, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := platformCopyTree(ctx, source, filepath.Join(t.TempDir(), "source")); err == nil {
		t.Fatal("canceled copy succeeded")
	}
}
func TestPlatformNaturalCompare(t *testing.T) {
	for _, test := range [][2]string{{"29.1.0-10", "29.1.0-9"}, {"3:29.1.0-2", "3:29.1.0-1"}} {
		if platformNaturalCompare(test[0], test[1]) <= 0 {
			t.Fatal(fmt.Sprint(test))
		}
	}
}
