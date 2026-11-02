package plugins

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

func instanceFor(t *testing.T, plugin core.Plugin, options map[string]any) core.Instance {
	t.Helper()
	parsed, err := plugin.ParseOptions(options)
	if err != nil {
		t.Fatal(err)
	}
	return core.Instance{ID: plugin.ID, Category: plugin.Category, Module: "builtin:" + plugin.ID, Options: parsed}
}
func TestRuntimeDetectionAndIdempotence(t *testing.T) {
	cases := []struct {
		plugin                  core.Plugin
		version, stdout, stderr string
	}{{Node(), "22", "v22.18.0", ""}, {Bun(), "1.3.2", "1.3.2", ""}, {Deno(), "2.5.4", "deno 2.5.4 (stable, release)\nv8 14", ""}, {Go(), "1.22.5", "go version go1.22.5 linux/amd64", ""}, {Java(), "17.0.9", "", `openjdk version "17.0.9"`}, {Pnpm(), "10.19.0", "10.19.0", ""}, {Yarn(), "4.9.4", "4.9.4", ""}}
	for _, c := range cases {
		t.Run(c.plugin.ID, func(t *testing.T) {
			r := testRuntime(t)
			i := instanceFor(t, c.plugin, map[string]any{"version": c.version, "install_dir": filepath.Join(r.Cwd, "managed")})
			r.Runner = func(_ context.Context, command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
				if strings.Contains(command, "javac") {
					return core.CommandResult{Stdout: "javac 17.0.9"}, nil
				}
				return core.CommandResult{Stdout: c.stdout, Stderr: c.stderr}, nil
			}
			found, err := c.plugin.Detect(r, i)
			if err != nil || !found.OK {
				t.Fatal(found, err)
			}
			result, err := c.plugin.Apply(r, i)
			if err != nil || !result.OK || result.DidChange {
				t.Fatal(result, err)
			}
			validated, err := c.plugin.Validate(r, i)
			if err != nil || !validated.OK {
				t.Fatal(validated, err)
			}
		})
	}
}
func TestJavaRequiresMatchingCompiler(t *testing.T) {
	for _, compiler := range []string{"", "javac 17.0.8", "javac 17.0.9"} {
		t.Run(compiler, func(t *testing.T) {
			r := testRuntime(t)
			r.Runner = func(_ context.Context, command string, args []string, o core.CommandOptions) (core.CommandResult, error) {
				if strings.Contains(command, "javac") {
					return core.CommandResult{Stdout: compiler}, nil
				}
				return core.CommandResult{Stderr: `openjdk version "17.0.9"`}, nil
			}
			i := instanceFor(t, Java(), map[string]any{"version": "17.0.9"})
			result, err := Java().Detect(r, i)
			if err != nil || result.OK != (compiler == "javac 17.0.9") {
				t.Fatal(result, err)
			}
		})
	}
}
func TestLegacyJavaVersions(t *testing.T) {
	for _, version := range []string{"8", "1.8"} {
		r := testRuntime(t)
		r.Runner = func(_ context.Context, command string, args []string, o core.CommandOptions) (core.CommandResult, error) {
			if command == "javac" {
				return core.CommandResult{Stdout: "javac 1.8.0_202"}, nil
			}
			return core.CommandResult{Stderr: `java version "1.8.0_202"`}, nil
		}
		i := instanceFor(t, Java(), map[string]any{"version": version})
		found, err := Java().Detect(r, i)
		if err != nil || !found.OK {
			t.Fatal(found, err)
		}
	}
}
func TestRuntimeInstallsFromRealArchives(t *testing.T) {
	for _, name := range []string{"node", "bun", "deno", "go", "java"} {
		for _, platform := range []string{"macos", "linux", "windows"} {
			t.Run(name+platform, func(t *testing.T) {
				r := testRuntime(t)
				r.Platform = platform
				r.Arch = "arm64"
				var plugin core.Plugin
				var version, archivePath string
				files := map[string]string{}
				format := "tar.gz"
				suffix := ""
				if platform == "windows" {
					suffix = ".exe"
					format = "zip"
				}
				switch name {
				case "node":
					plugin = Node()
					version = "22.18.0"
					archivePath = "node-v22.18.0/bin/node" + suffix
					if platform == "windows" {
						archivePath = "node-v22.18.0/node.exe"
					}
					files[archivePath] = "v22.18.0"
				case "go":
					plugin = Go()
					version = "1.22.5"
					archivePath = "go/bin/go" + suffix
					files[archivePath] = "go version go1.22.5"
				case "java":
					plugin = Java()
					version = "17.0.9"
					prefix := "jdk/bin/"
					if platform == "macos" {
						prefix = "jdk/Contents/Home/bin/"
					}
					archivePath = prefix + "java" + suffix
					files[archivePath] = `openjdk version "17.0.9"`
					files[prefix+"javac"+suffix] = "javac 17.0.9"
				case "bun":
					plugin = Bun()
					version = "1.3.2"
					format = "zip"
					archivePath = "bun-target/bun" + suffix
					files[archivePath] = version
				case "deno":
					plugin = Deno()
					version = "2.5.4"
					format = "zip"
					archivePath = "deno" + suffix
					files[archivePath] = "deno " + version + " (stable, release)"
				}
				data := tarFixture(t, files)
				if format == "zip" {
					data = zipFixture(t, files)
				}
				hash := fmt.Sprintf("%x", sha256.Sum256(data))
				filename := ""
				mockHTTP(t, func(request *http.Request) (string, error) {
					uri := request.URL.String()
					if request.URL.Host == "archive.test" {
						return string(data), nil
					}
					switch name {
					case "node":
						if strings.HasSuffix(request.URL.Path, "index.json") {
							return `[{"version":"v22.18.0"}]`, nil
						}
						if strings.HasSuffix(request.URL.Path, "SHASUMS256.txt") {
							system := platform
							if platform == "macos" {
								system = "darwin"
							}
							if platform == "windows" {
								system = "win"
							}
							filename = "node-v22.18.0-" + system + "-arm64." + format
							return hash + "  " + filename, nil
						}
						return string(data), nil
					case "go":
						if strings.Contains(uri, "mode=json") {
							system := platform
							if system == "macos" {
								system = "darwin"
							}
							filename = "go1.22.5." + system + "-arm64." + format
							return fmt.Sprintf(`[{"files":[{"filename":%q,"sha256":%q}]}]`, filename, hash), nil
						}
						return string(data), nil
					case "java":
						return fmt.Sprintf(`[{"binaries":[{"package":{"link":"https://archive.test/archive","checksum":%q}}]}]`, hash), nil
					case "bun":
						system := platform
						if system == "macos" {
							system = "darwin"
						}
						filename = "bun-" + system + "-aarch64.zip"
						return fmt.Sprintf(`{"assets":[{"name":%q,"browser_download_url":"https://archive.test/archive","digest":"sha256:%s"}]}`, filename, hash), nil
					case "deno":
						if strings.HasSuffix(uri, ".sha256sum") {
							return hash, nil
						}
						return string(data), nil
					}
					return "", fmt.Errorf("unexpected URL %s", uri)
				})
				r.Runner = func(_ context.Context, command string, args []string, o core.CommandOptions) (core.CommandResult, error) {
					content, err := os.ReadFile(command)
					if err != nil {
						return core.CommandResult{Code: 1}, nil
					}
					return core.CommandResult{Stdout: string(content)}, nil
				}
				destination := filepath.Join(r.Cwd, "installed runtime")
				i := instanceFor(t, plugin, map[string]any{"version": version, "install_dir": destination})
				result, err := plugin.Apply(r, i)
				if err != nil || !result.OK || !result.DidChange {
					t.Fatal(result, err)
				}
				result, err = plugin.Apply(r, i)
				if err != nil || !result.OK || result.DidChange {
					t.Fatal("not idempotent", result, err)
				}
				if !strings.Contains(r.Env.Get("PATH"), destination) {
					t.Fatal("runtime not exposed")
				}
				if name == "java" && r.Env.Get("JAVA_HOME") != destination {
					t.Fatal("JAVA_HOME missing")
				}
				if name == "go" && r.Env.Get("GOROOT") != destination {
					t.Fatal("GOROOT missing")
				}
			})
		}
	}
}
func TestNodeNvmAndGlobalPackages(t *testing.T) {
	for _, fail := range []string{"", "bootstrap", "install", "alias", "resolve", "verify"} {
		t.Run(fail, func(t *testing.T) {
			r := testRuntime(t)
			r.Platform = "linux"
			i := instanceFor(t, Node(), map[string]any{"version": "22", "global_packages": []string{"typescript"}})
			var calls []string
			r.Runner = func(_ context.Context, command string, args []string, o core.CommandOptions) (core.CommandResult, error) {
				call := command + " " + strings.Join(args, " ")
				calls = append(calls, call)
				if command == "node" {
					return core.CommandResult{Code: 1}, nil
				}
				if strings.Contains(call, "curl -fsSL") {
					if fail == "bootstrap" {
						return core.CommandResult{Code: 1, Stderr: "bootstrap failed"}, nil
					}
					return core.CommandResult{}, nil
				}
				if strings.Contains(call, "nvm install") && fail == "install" || strings.Contains(call, "nvm alias") && fail == "alias" {
					return core.CommandResult{Code: 1, Stderr: "nvm failed"}, nil
				}
				if strings.Contains(call, "nvm which") {
					path := filepath.Join(r.Cwd, "node", "bin", "node")
					if fail == "resolve" {
						path = "relative"
					}
					return core.CommandResult{Stdout: path}, nil
				}
				if filepath.IsAbs(command) {
					version := "v22.18.0"
					if fail == "verify" {
						version = "v18.0.0"
					}
					return core.CommandResult{Stdout: version}, nil
				}
				if command == "npm" {
					if !strings.Contains(o.Env["PATH"], filepath.Join(r.Cwd, "node", "bin")) {
						t.Fatal("global packages PATH not selected")
					}
					return core.CommandResult{}, nil
				}
				return core.CommandResult{}, nil
			}
			result, err := Node().Apply(r, i)
			if err != nil || result.OK != (fail == "") {
				t.Fatal(result, err, calls)
			}
			if fail == "" {
				if err = Node().PostApply(r, i); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(calls[len(calls)-1], "npm install -g -- typescript") {
					t.Fatal(calls)
				}
			}
		})
	}
}
func TestNpmPrefixInstallation(t *testing.T) {
	for _, name := range []string{"pnpm", "yarn"} {
		for _, platform := range []string{"linux", "windows"} {
			t.Run(name+platform, func(t *testing.T) {
				r := testRuntime(t)
				r.Platform = platform
				plugin := Pnpm()
				version := "10.19.0"
				packageName := "pnpm"
				if name == "yarn" {
					plugin = Yarn()
					version = "4.9.4"
					packageName = "@yarnpkg/cli-dist"
				}
				i := instanceFor(t, plugin, map[string]any{"version": version, "install_dir": filepath.Join(r.Cwd, "prefix"), "node_plugin": nil})
				r.Runner = func(_ context.Context, command string, args []string, o core.CommandOptions) (core.CommandResult, error) {
					if command == "npm" {
						if args[0] == "--version" {
							return core.CommandResult{Stdout: "10.9.0"}, nil
						}
						for _, required := range []string{"--ignore-scripts", "--engine-strict", packageName + "@" + version} {
							if !strings.Contains(strings.Join(args, " "), required) {
								t.Fatal(args)
							}
						}
						var prefix string
						for index, arg := range args {
							if arg == "--prefix" {
								prefix = args[index+1]
							}
						}
						filename := filepath.Join(prefix, "bin", name)
						if platform == "windows" {
							filename = filepath.Join(prefix, name+".cmd")
						}
						os.MkdirAll(filepath.Dir(filename), 0755)
						os.WriteFile(filename, []byte(version), 0755)
						return core.CommandResult{}, nil
					}
					if o.Env["YARN_IGNORE_PATH"] != "1" || o.Env["COREPACK_ENABLE_PROJECT_SPEC"] != "0" {
						t.Fatal("project config not isolated")
					}
					data, err := os.ReadFile(command)
					if err != nil {
						return core.CommandResult{Code: 1}, nil
					}
					return core.CommandResult{Stdout: string(data)}, nil
				}
				result, err := plugin.Apply(r, i)
				if err != nil || !result.OK || !result.DidChange {
					t.Fatal(result, err)
				}
				result, err = plugin.Apply(r, i)
				if err != nil || !result.OK || result.DidChange {
					t.Fatal(result, err)
				}
			})
		}
	}
}
func TestRegistryDependencies(t *testing.T) {
	config := core.Config{Tools: []core.Instance{{ID: "custom-node", Module: "builtin:node", Options: map[string]any{"version": "22"}}, {ID: "yarn", Module: "builtin:yarn", Options: map[string]any{"version": "4.9.4", "node_plugin": "custom-node"}}}}
	nodes, err := Load(config)
	if err != nil || len(nodes) != 2 || nodes[1].Plugin.DependsOn[0] != "custom-node" {
		t.Fatal(nodes, err)
	}
	config.Tools = config.Tools[1:]
	config.Tools[0].Options["node_plugin"] = nil
	nodes, err = Load(config)
	if err != nil || len(nodes[0].Plugin.DependsOn) != 0 {
		t.Fatal(nodes, err)
	}
	if len(Builtins()) != 11 {
		t.Fatal("catalog lost plugin")
	}
}
