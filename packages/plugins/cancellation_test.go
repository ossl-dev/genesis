package plugins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

func TestArchivePluginCancellationAndDeadlinesPreserveIdentity(t *testing.T) {
	cases := []homePluginCase{
		{Node(), map[string]any{"version": "22.18.0", "use_nvm": false}, ""},
		{Bun(), map[string]any{"version": "1.3.2"}, ""},
		{Deno(), map[string]any{"version": "2.5.4"}, ""},
		{Go(), map[string]any{"version": "1.22.5"}, ""},
		{Java(), map[string]any{"version": "17.0.9"}, ""},
		{Git(), map[string]any{"version": "2.53.0", "install_method": "source"}, ""},
	}
	for _, test := range cases {
		for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
			t.Run(test.plugin.ID+"/"+cause.Error(), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				r, _ := platformTestRuntime(t, "macos", nil)
				r.Context = ctx
				destination := filepath.Join(r.Cwd, "managed")
				options := map[string]any{}
				for key, value := range test.options {
					options[key] = value
				}
				options["install_dir"] = destination
				i := platformTestInstance(t, test.plugin, options)
				platformHTTP(t, func(*http.Request) ([]byte, error) {
					if cause == context.Canceled {
						cancel()
					}
					return nil, fmt.Errorf("release interrupted: %w", cause)
				})
				applied, err := test.plugin.Apply(r, i)
				if !errors.Is(err, cause) || applied.OK || applied.DidChange || !strings.Contains(applied.Details, "installation failed") {
					t.Fatalf("%+v %v", applied, err)
				}
				if _, err := os.Stat(destination); !os.IsNotExist(err) {
					t.Fatalf("destination changed: %v", err)
				}
			})
		}
	}
}
func TestGitDownloadCancellationPreservesPreviousPrefix(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, _ := platformTestRuntime(t, "macos", nil)
	r.Context = ctx
	destination := filepath.Join(r.Cwd, "git")
	previous := filepath.Join(destination, "bin", "git")
	if err := os.MkdirAll(filepath.Dir(previous), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(previous, []byte("previous"), 0755); err != nil {
		t.Fatal(err)
	}
	platformHTTP(t, func(request *http.Request) ([]byte, error) {
		if strings.HasSuffix(request.URL.Path, "sha256sums.asc") {
			return []byte(strings.Repeat("a", 64) + "  git-2.53.0.tar.gz"), nil
		}
		cancel()
		return nil, ctx.Err()
	})
	i := platformTestInstance(t, Git(), map[string]any{"version": "2.53.0", "install_method": "source", "install_dir": destination})
	applied, err := Git().Apply(r, i)
	if !errors.Is(err, context.Canceled) || applied.DidChange || applied.OK {
		t.Fatalf("%+v %v", applied, err)
	}
	data, err := os.ReadFile(previous)
	if err != nil || string(data) != "previous" {
		t.Fatalf("previous prefix not preserved: %s %v", data, err)
	}
	entries, err := os.ReadDir(r.Cwd)
	if err != nil || len(entries) != 1 || entries[0].Name() != "git" {
		t.Fatalf("staging or lock leaked: %+v %v", entries, err)
	}
}
func TestNpmStagingCancellationPreservesPreviousPrefix(t *testing.T) {
	for _, plugin := range []core.Plugin{Pnpm(), Yarn()} {
		t.Run(plugin.ID, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, _ := platformTestRuntime(t, "macos", nil)
			r.Context = ctx
			destination := filepath.Join(r.Cwd, plugin.ID)
			previous := filepath.Join(destination, "bin", plugin.ID)
			if err := os.MkdirAll(filepath.Dir(previous), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(previous, []byte("previous"), 0755); err != nil {
				t.Fatal(err)
			}
			r.Runner = func(_ context.Context, command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				if command == "npm" && args[0] == "--version" {
					return core.CommandResult{Stdout: "10.0.0"}, nil
				}
				if command == "npm" && args[0] == "install" {
					prefix := ""
					for index, arg := range args {
						if arg == "--prefix" {
							prefix = args[index+1]
						}
					}
					if err := os.MkdirAll(prefix, 0755); err != nil {
						return core.CommandResult{}, err
					}
					if err := os.WriteFile(filepath.Join(prefix, "partial"), nil, 0644); err != nil {
						return core.CommandResult{}, err
					}
					cancel()
					return core.CommandResult{Code: 1}, fmt.Errorf("npm canceled: %w", ctx.Err())
				}
				return core.CommandResult{Code: 1}, nil
			}
			i := platformTestInstance(t, plugin, map[string]any{"version": "1.2.3", "node_plugin": nil, "install_dir": destination})
			applied, err := plugin.Apply(r, i)
			if !errors.Is(err, context.Canceled) || applied.DidChange || applied.OK {
				t.Fatalf("%+v %v", applied, err)
			}
			data, err := os.ReadFile(previous)
			if err != nil || string(data) != "previous" {
				t.Fatalf("%s %v", data, err)
			}
			entries, err := os.ReadDir(r.Cwd)
			if err != nil || len(entries) != 1 {
				t.Fatalf("staging leaked: %+v %v", entries, err)
			}
		})
	}
}
func TestHomebrewCancellationRetainsAttemptedChanges(t *testing.T) {
	for _, step := range []string{"bootstrap", "update", "upgrade", "upgrade --cask"} {
		t.Run(step, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, _ := platformTestRuntime(t, "macos", nil)
			r.Context = ctx
			r.Runner = func(_ context.Context, command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				if command == "brew" && args[0] == "--version" {
					if step == "bootstrap" {
						return core.CommandResult{Code: 1}, nil
					}
					return core.CommandResult{Stdout: "Homebrew 4.3.0"}, nil
				}
				if command == "bash" || strings.Join(args, " ") == step {
					cancel()
					return core.CommandResult{Code: 1}, fmt.Errorf("command interrupted: %w", ctx.Err())
				}
				return core.CommandResult{}, nil
			}
			applied, err := Homebrew().Apply(r, platformTestInstance(t, Homebrew(), nil))
			if !errors.Is(err, context.Canceled) || !applied.DidChange || applied.OK {
				t.Fatalf("%+v %v", applied, err)
			}
		})
	}
	r, _ := platformTestRuntime(t, "macos", nil)
	r.Arch = "386"
	applied, err := Homebrew().Apply(r, platformTestInstance(t, Homebrew(), nil))
	if err != nil || applied.DidChange || applied.OK {
		t.Fatalf("preflight changed: %+v %v", applied, err)
	}
}
func TestNVMCommandsReportPartialChangesOnCancellation(t *testing.T) {
	for _, installedNVM := range []bool{false, true} {
		t.Run(fmt.Sprint(installedNVM), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r, _ := platformTestRuntime(t, "macos", nil)
			r.Context = ctx
			nvm := filepath.Join(r.Cwd, "nvm")
			r.Env.Set("NVM_DIR", nvm)
			if installedNVM {
				if err := os.Mkdir(nvm, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(nvm, "nvm.sh"), nil, 0644); err != nil {
					t.Fatal(err)
				}
			}
			r.Runner = func(_ context.Context, command string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
				if command == "bash" {
					cancel()
					return core.CommandResult{Code: 1}, fmt.Errorf("NVM interrupted: %w", ctx.Err())
				}
				return core.CommandResult{Code: 1}, nil
			}
			applied, err := Node().Apply(r, platformTestInstance(t, Node(), map[string]any{"version": "22"}))
			if !errors.Is(err, context.Canceled) || !applied.DidChange || applied.OK {
				t.Fatalf("%+v %v", applied, err)
			}
		})
	}
}
func TestGitBuildRespectsCPUQuotaAndPreservesCancellation(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, _ := platformTestRuntime(t, "macos", nil)
	r.Context = ctx
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "source"), 0755); err != nil {
		t.Fatal(err)
	}
	var work string
	r.Runner = func(_ context.Context, command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
		if command != "make" || args[0] != "-j1" {
			t.Fatalf("ignored CPU quota: %s %+v", command, args)
		}
		work = filepath.Dir(options.Cwd)
		cancel()
		return core.CommandResult{Code: 1}, fmt.Errorf("build interrupted: %w", ctx.Err())
	}
	if _, err := buildGit(r, root); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if work == "" || exists(work) {
		t.Fatalf("build directory leaked: %s", work)
	}
}
func TestDockerDaemonTimeoutRemainsDiagnostic(t *testing.T) {
	r, _ := platformTestRuntime(t, "linux", dockerTestRunner("Docker version 29.1.0", "Docker Compose version v2.40.0", "", "29.1.0"))
	runner := r.Runner
	r.Runner = func(ctx context.Context, command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
		if args[0] == "info" {
			return core.CommandResult{Code: 1}, fmt.Errorf("probe: %w", context.DeadlineExceeded)
		}
		return runner(ctx, command, args, options)
	}
	validated, err := Docker().Validate(r, platformTestInstance(t, Docker(), nil))
	if err != nil || validated.OK || !strings.Contains(validated.Message, "deadline exceeded") {
		t.Fatalf("%+v %v", validated, err)
	}
}
