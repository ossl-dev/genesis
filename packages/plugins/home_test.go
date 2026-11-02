package plugins

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

type homePluginCase struct {
	plugin  core.Plugin
	options map[string]any
	output  string
}

func homePluginCases(platform string) []homePluginCase {
	method := "source"
	if platform == "windows" {
		method = "binary"
	}
	cases := []homePluginCase{
		{Node(), map[string]any{"version": "22.18.0", "use_nvm": false}, "v22.18.0"},
		{Bun(), map[string]any{"version": "1.3.2"}, "1.3.2"},
		{Deno(), map[string]any{"version": "2.5.4"}, "deno 2.5.4"},
		{Pnpm(), map[string]any{"version": "10.19.0", "node_plugin": nil}, "10.19.0"},
		{Yarn(), map[string]any{"version": "4.9.4", "node_plugin": nil}, "4.9.4"},
		{Git(), map[string]any{"version": "2.53.0", "install_method": method}, "git version 2.53.0"},
	}
	if platform == "windows" {
		cases = append(cases, homePluginCase{Go(), map[string]any{"version": "1.22.5"}, "go version go1.22.5"}, homePluginCase{Java(), map[string]any{"version": "17.0.9"}, `openjdk version "17.0.9"`})
	}
	return cases
}
func TestManagedInstallsRejectInvalidHomesBeforeSideEffects(t *testing.T) {
	platformHTTP(t, func(*http.Request) ([]byte, error) {
		t.Fatal("home validation must precede release downloads")
		return nil, nil
	})
	for _, platform := range []string{"macos", "windows"} {
		for _, test := range homePluginCases(platform) {
			for _, home := range []string{"", "relative home"} {
				t.Run(platform+"/"+test.plugin.ID+"/"+home, func(t *testing.T) {
					r, commands := platformTestRuntime(t, platform, nil)
					r.Env = core.NewEnvironment(map[string]string{"HOME": home, "USERPROFILE": home})
					i := platformTestInstance(t, test.plugin, test.options)
					if err := test.plugin.Prepare(r, i); err == nil {
						t.Fatal("invalid home accepted during preparation")
					}
					applied, err := test.plugin.Apply(r, i)
					if err == nil || applied.OK || applied.DidChange {
						t.Fatalf("%+v %v", applied, err)
					}
					for _, call := range *commands {
						if len(call.Args) == 0 || (call.Args[0] != "--version" && call.Args[0] != "-v" && call.Args[0] != "-version" && call.Args[0] != "version") {
							t.Fatalf("unexpected mutation command: %+v", call)
						}
					}
					files, err := os.ReadDir(r.Cwd)
					if err != nil || len(files) != 0 {
						t.Fatalf("unexpected files: %+v %v", files, err)
					}
				})
			}
		}
	}
}
func TestInstalledHostToolsRemainUsableWithoutHome(t *testing.T) {
	for _, platform := range []string{"macos", "windows"} {
		for _, test := range homePluginCases(platform) {
			t.Run(platform+"/"+test.plugin.ID, func(t *testing.T) {
				r, _ := platformTestRuntime(t, platform, func(command string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
					if command == "javac" {
						return core.CommandResult{Stdout: "javac 17.0.9"}, nil
					}
					if strings.ContainsAny(command, "/\\") {
						t.Fatalf("invalid default home must not produce a path: %s", command)
					}
					return core.CommandResult{Stdout: test.output}, nil
				})
				r.Env = core.NewEnvironment(map[string]string{"HOME": "", "USERPROFILE": ""})
				i := platformTestInstance(t, test.plugin, test.options)
				if err := test.plugin.Prepare(r, i); err != nil {
					t.Fatal(err)
				}
				found, err := test.plugin.Detect(r, i)
				if err != nil || !found.OK {
					t.Fatalf("%+v %v", found, err)
				}
				applied, err := test.plugin.Apply(r, i)
				if err != nil || !applied.OK || applied.DidChange {
					t.Fatalf("%+v %v", applied, err)
				}
			})
		}
	}
}
func TestExplicitInstallDirectoriesDoNotRequireHome(t *testing.T) {
	for _, platform := range []string{"macos", "windows"} {
		for _, test := range homePluginCases(platform) {
			t.Run(platform+"/"+test.plugin.ID, func(t *testing.T) {
				r, _ := platformTestRuntime(t, platform, func(command string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
					if strings.Contains(command, "javac") {
						return core.CommandResult{Stdout: "javac 17.0.9"}, nil
					}
					return core.CommandResult{Stdout: test.output}, nil
				})
				r.Env = core.NewEnvironment(map[string]string{"HOME": "", "USERPROFILE": ""})
				options := map[string]any{}
				for key, value := range test.options {
					options[key] = value
				}
				options["install_dir"] = filepath.Join(r.Cwd, "managed tool")
				i := platformTestInstance(t, test.plugin, options)
				if err := test.plugin.Prepare(r, i); err != nil {
					t.Fatal(err)
				}
				applied, err := test.plugin.Apply(r, i)
				if err != nil || !applied.OK || applied.DidChange {
					t.Fatalf("%+v %v", applied, err)
				}
			})
		}
	}
}
func TestRuntimeHomeFallbackRequiresNativeAbsoluteHome(t *testing.T) {
	variable := "HOME"
	if os.PathSeparator == '\\' {
		variable = "USERPROFILE"
	}
	r, _ := platformTestRuntime(t, core.GetPlatform(), nil)
	r.Env = core.NewEnvironment(map[string]string{})
	t.Setenv(variable, "")
	if _, err := runtimeHome(r); err == nil {
		t.Fatal("missing native home accepted")
	}
	t.Setenv(variable, "relative")
	if _, err := runtimeHome(r); err == nil {
		t.Fatal("relative native home accepted")
	}
	home := t.TempDir()
	t.Setenv(variable, home)
	resolved, err := runtimeHome(r)
	if err != nil || resolved != home {
		t.Fatalf("%s %v", resolved, err)
	}
}
func TestNVMPathsValidatedBeforePrerequisites(t *testing.T) {
	for _, test := range []struct{ key, value string }{{"HOME", ""}, {"HOME", "relative home"}, {"NVM_DIR", "relative nvm"}, {"XDG_CONFIG_HOME", "relative xdg"}} {
		t.Run(test.key+test.value, func(t *testing.T) {
			r, _ := platformTestRuntime(t, "macos", nil)
			r.Env.Set(test.key, test.value)
			i := platformTestInstance(t, Node(), map[string]any{"version": "22"})
			if err := Node().Prepare(r, i); err == nil {
				t.Fatal("invalid NVM directory accepted")
			}
			if err := Node().RegisterTasks(r, i); err == nil {
				t.Fatal("invalid NVM directory registered package tasks")
			}
			if len(r.Tasks.IDs()) != 0 {
				t.Fatal(r.Tasks.IDs())
			}
			applied, err := Node().Apply(r, i)
			if err == nil || applied.DidChange {
				t.Fatalf("%+v %v", applied, err)
			}
		})
	}
	r, _ := platformTestRuntime(t, "macos", nil)
	r.Env.Set("HOME", "")
	nvm := filepath.Join(r.Cwd, "nvm home")
	r.Env.Set("NVM_DIR", nvm)
	i := platformTestInstance(t, Node(), map[string]any{"version": "22"})
	if err := Node().Prepare(r, i); err != nil {
		t.Fatal(err)
	}
	directory, err := nvmDirectory(r)
	if err != nil || directory != nvm {
		t.Fatalf("%s %v", directory, err)
	}
	r.Env.Set("NVM_DIR", "")
	r.Env.Set("XDG_CONFIG_HOME", r.Cwd)
	directory, err = nvmDirectory(r)
	if err != nil || directory != filepath.Join(r.Cwd, "nvm") {
		t.Fatalf("%s %v", directory, err)
	}
}
func TestNVMInstalledHostSkipsHomeRequirement(t *testing.T) {
	r, _ := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
		return core.CommandResult{Stdout: "v22.18.0"}, nil
	})
	r.Env = core.NewEnvironment(map[string]string{"HOME": ""})
	i := platformTestInstance(t, Node(), map[string]any{"version": "22"})
	if err := Node().Prepare(r, i); err != nil {
		t.Fatal(err)
	}
	if err := Node().RegisterTasks(r, i); err != nil {
		t.Fatal(err)
	}
	if len(r.Tasks.IDs()) != 0 {
		t.Fatal(r.Tasks.IDs())
	}
	applied, err := Node().Apply(r, i)
	if err != nil || !applied.OK || applied.DidChange {
		t.Fatalf("%+v %v", applied, err)
	}
}
func TestManagedHomeCancellationPropagates(t *testing.T) {
	r, _ := platformTestRuntime(t, "macos", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.Context = ctx
	r.Env.Set("HOME", "")
	i := platformTestInstance(t, Bun(), map[string]any{"version": "1.3.2"})
	if _, err := Bun().Apply(r, i); err != context.Canceled {
		t.Fatal(err)
	}
}
