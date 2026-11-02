package plugins

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

func TestGitVersionDetection(t *testing.T) {
	for _, test := range []struct {
		output string
		ok     bool
	}{{"git version 2.53.0", true}, {"git version 2.53.0.windows.1", true}, {"git version 2.53.0 (Apple Git-1)", true}, {"git version 2.5.0", false}, {"unparseable", false}, {"", false}} {
		t.Run(test.output, func(t *testing.T) {
			runtime, _ := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
				return core.CommandResult{Stdout: test.output}, nil
			})
			instance := platformTestInstance(t, Git(), map[string]any{"version": "2.53"})
			detected, err := Git().Detect(runtime, instance)
			if err != nil || detected.OK != test.ok {
				t.Fatalf("%+v %v", detected, err)
			}
			validated, err := Git().Validate(runtime, instance)
			if err != nil || validated.OK != test.ok {
				t.Fatalf("%+v %v", validated, err)
			}
		})
	}
}
func TestGitExistingSkipsTasksAndApply(t *testing.T) {
	runtime, commands := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
		return core.CommandResult{Stdout: "git version 2.53.0"}, nil
	})
	instance := platformTestInstance(t, Git(), nil)
	if err := Git().RegisterTasks(runtime, instance); err != nil || len(runtime.Tasks.IDs()) != 0 {
		t.Fatalf("%v %+v", err, runtime.Tasks.IDs())
	}
	result, err := Git().Apply(runtime, instance)
	if err != nil || !result.OK || result.DidChange {
		t.Fatalf("%+v %v", result, err)
	}
	for _, command := range *commands {
		if command.Command != "git" || command.Args[0] != "--version" {
			t.Fatal(command)
		}
	}
}
func TestGitManagedDiscoveryAndExplicitMissingPrefix(t *testing.T) {
	for _, platform := range []string{"macos", "windows"} {
		t.Run(platform, func(t *testing.T) {
			prefix := filepath.Join(t.TempDir(), "managed git")
			runtime, commands := platformTestRuntime(t, platform, func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
				return core.CommandResult{Stdout: "git version 2.53.0"}, nil
			})
			instance := platformTestInstance(t, Git(), map[string]any{"install_method": "binary", "install_dir": prefix})
			target := gitExecutable(runtime, prefix)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, nil, 0755); err != nil {
				t.Fatal(err)
			}
			detected, err := Git().Detect(runtime, instance)
			if err != nil || !detected.OK {
				t.Fatalf("%+v %v", detected, err)
			}
			if (*commands)[0].Command != target || !strings.HasPrefix(runtime.Env.Get("PATH"), filepath.Dir(target)) {
				t.Fatalf("%+v %s", *commands, runtime.Env.Get("PATH"))
			}
			if err := os.Remove(target); err != nil {
				t.Fatal(err)
			}
			if _, err := Git().Detect(runtime, instance); err != nil {
				t.Fatal(err)
			}
			if (*commands)[1].Command != target {
				t.Fatal(*commands)
			}
		})
	}
}
func TestGitUnsupportedModesAndInvalidOptions(t *testing.T) {
	for _, test := range []struct{ platform, method, message string }{{"macos", "binary", "no standalone Unix binaries"}, {"linux", "binary", "no standalone Unix binaries"}, {"windows", "source", "MinGit"}, {"windows", "package", "MinGit"}} {
		t.Run(test.platform+test.method, func(t *testing.T) {
			runtime, _ := platformTestRuntime(t, test.platform, nil)
			instance := platformTestInstance(t, Git(), map[string]any{"install_method": test.method})
			if err := Git().Prepare(runtime, instance); err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatal(err)
			}
			if err := Git().RegisterTasks(runtime, instance); err != nil || len(runtime.Tasks.IDs()) != 0 {
				t.Fatalf("%v %+v", err, runtime.Tasks.IDs())
			}
			result, err := Git().Apply(runtime, instance)
			if err != nil || result.OK || result.DidChange || !strings.Contains(result.Details, test.message) {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
	if _, err := Git().ParseOptions(map[string]any{"install_dir": t.TempDir()}); err == nil {
		t.Fatal("package installation accepted install_dir")
	}
}
func TestGitPackageFailureAndPrerequisites(t *testing.T) {
	runtime, _ := platformTestRuntime(t, "macos", nil)
	instance := platformTestInstance(t, Git(), nil)
	if err := Git().RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	ids := strings.Join(runtime.Tasks.IDs(), ",")
	if !strings.Contains(ids, "install:git") {
		t.Fatal(ids)
	}
	result, err := Git().Apply(runtime, instance)
	if err != nil || result.OK || result.DidChange || !strings.Contains(result.Details, "still unavailable") {
		t.Fatalf("%+v %v", result, err)
	}
	runtime.Tasks = core.NewTaskRegistry()
	instance = platformTestInstance(t, Git(), map[string]any{"install_method": "source"})
	if err := Git().RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runtime.Tasks.IDs(), ","), "install:make") {
		t.Fatal(runtime.Tasks.IDs())
	}
}
func TestGitVerifyContainsHTTPSHelper(t *testing.T) {
	root := t.TempDir()
	helperRoot := filepath.Join(root, "libexec", "git-core")
	if err := os.MkdirAll(helperRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperRoot, "git-remote-https"), nil, 0755); err != nil {
		t.Fatal(err)
	}
	helper := helperRoot
	version := "2.53.0"
	runtime, _ := platformTestRuntime(t, "macos", func(_ string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
		if args[0] == "--version" {
			return core.CommandResult{Stdout: "git version " + version}, nil
		}
		return core.CommandResult{Stdout: helper}, nil
	})
	if err := verifyGit(runtime, root, "2.53.0"); err != nil {
		t.Fatal(err)
	}
	helper = t.TempDir()
	if err := verifyGit(runtime, root, "2.53.0"); err == nil {
		t.Fatal("helper outside prefix accepted")
	}
	helper = helperRoot
	version = "2.52.0"
	if err := verifyGit(runtime, root, "2.53.0"); err == nil {
		t.Fatal("wrong release accepted")
	}
}
func platformGitArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	compressor := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(compressor)
	data := []byte("all:\n")
	if err := writer.WriteHeader(&tar.Header{Name: "git-2.53.0/", Typeflag: tar.TypeDir, Mode: 0755}); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteHeader(&tar.Header{Name: "git-2.53.0/Makefile", Size: int64(len(data)), Mode: 0644, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
func TestGitSourceStagedInstallAndRediscovery(t *testing.T) {
	archive := platformGitArchive(t)
	digest := sha256.Sum256(archive)
	platformHTTP(t, func(request *http.Request) ([]byte, error) {
		if strings.HasSuffix(request.URL.Path, "sha256sums.asc") {
			return []byte(hex.EncodeToString(digest[:]) + "  git-2.53.0.tar.gz"), nil
		}
		return archive, nil
	})
	var buildDirectory string
	runtime, commands := platformTestRuntime(t, "macos", func(command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
		if command == "make" {
			buildDirectory = options.Cwd
			if strings.ContainsAny(options.Cwd, " \t\r\n") {
				t.Fatal("build path contains whitespace")
			}
			if args[len(args)-1] == "install" {
				destination := ""
				for _, arg := range args {
					if strings.HasPrefix(arg, "DESTDIR=") {
						destination = strings.TrimPrefix(arg, "DESTDIR=")
					}
				}
				if destination == "" {
					return core.CommandResult{}, fmt.Errorf("missing DESTDIR")
				}
				for _, relative := range []string{"usr/local/bin/git", "usr/local/libexec/git-core/git-remote-https"} {
					path := filepath.Join(destination, relative)
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						return core.CommandResult{}, err
					}
					if err := os.WriteFile(path, nil, 0755); err != nil {
						return core.CommandResult{}, err
					}
				}
			}
			return core.CommandResult{}, nil
		}
		if command == "git" || !exists(command) {
			return core.CommandResult{Code: 1}, nil
		}
		if args[0] == "--version" {
			return core.CommandResult{Stdout: "git version 2.53.0"}, nil
		}
		return core.CommandResult{Stdout: filepath.Join(filepath.Dir(filepath.Dir(command)), "libexec", "git-core")}, nil
	})
	prefix := filepath.Join(t.TempDir(), "managed git")
	instance := platformTestInstance(t, Git(), map[string]any{"version": "2.53", "install_method": "source", "install_dir": prefix})
	result, err := Git().Apply(runtime, instance)
	if err != nil || !result.OK || !result.DidChange {
		t.Fatalf("%+v %v", result, err)
	}
	if exists(filepath.Dir(buildDirectory)) {
		t.Fatal("build temp leaked")
	}
	if !strings.HasPrefix(runtime.Env.Get("PATH"), filepath.Join(prefix, "bin")) {
		t.Fatal(runtime.Env.Get("PATH"))
	}
	count := len(*commands)
	result, err = Git().Apply(runtime, instance)
	if err != nil || !result.OK || result.DidChange || len(*commands) != count+1 {
		t.Fatalf("%+v %v", result, err)
	}
	for _, call := range *commands {
		if call.Command == "git" && call.Args[0] == "config" {
			t.Fatal("global configuration mutation")
		}
	}
}
func TestGitBuildFailureCleansTemporarySource(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "source"), 0755); err != nil {
		t.Fatal(err)
	}
	var work string
	runtime, _ := platformTestRuntime(t, "macos", func(command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
		work = filepath.Dir(options.Cwd)
		if command != "make" || args[len(args)-1] != "all" || !strings.Contains(strings.Join(args, " "), "RUNTIME_PREFIX=YesPlease") {
			t.Fatalf("%s %+v", command, args)
		}
		return core.CommandResult{Code: 1, Stderr: "compiler failed"}, nil
	})
	if _, err := buildGit(runtime, root); err == nil || !strings.Contains(err.Error(), "Git build failed") {
		t.Fatal(err)
	}
	if work == "" || exists(work) {
		t.Fatalf("temporary build leaked: %s", work)
	}
}

func TestGitVerifyMinGitRelocatedHTTPSHelper(t *testing.T) {
	for _, prefix := range []string{"mingw64", "ucrt64", "clangarm64"} {
		t.Run(prefix, func(t *testing.T) {
			root := t.TempDir()
			execPath := filepath.Join(root, prefix, "libexec", "git-core")
			helperPath := filepath.Join(root, prefix, "bin", "git-remote-https.exe")
			for _, directory := range []string{execPath, filepath.Dir(helperPath)} {
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(helperPath, nil, 0755); err != nil {
				t.Fatal(err)
			}
			runtime, _ := platformTestRuntime(t, "windows", func(_ string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				if args[0] == "--version" {
					return core.CommandResult{Stdout: "git version 2.53.0.windows.1"}, nil
				}
				return core.CommandResult{Stdout: execPath}, nil
			})
			if err := verifyGit(runtime, root, "2.53.0"); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(helperPath); err != nil {
				t.Fatal(err)
			}
			if err := verifyGit(runtime, root, "2.53.0"); err == nil || !strings.Contains(err.Error(), "git-remote-https.exe") {
				t.Fatalf("missing bundled helper not reported: %v", err)
			}
		})
	}
}

func TestGitVerifyRejectsExternalHTTPSHelperSymlink(t *testing.T) {
	root := t.TempDir()
	execPath := filepath.Join(root, "mingw64", "libexec", "git-core")
	helperPath := filepath.Join(root, "mingw64", "bin", "git-remote-https.exe")
	for _, directory := range []string{execPath, filepath.Dir(helperPath)} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	external := filepath.Join(t.TempDir(), "git-remote-https.exe")
	if err := os.WriteFile(external, nil, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, helperPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	runtime, _ := platformTestRuntime(t, "windows", func(_ string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
		if args[0] == "--version" {
			return core.CommandResult{Stdout: "git version 2.53.0.windows.1"}, nil
		}
		return core.CommandResult{Stdout: execPath}, nil
	})
	if err := verifyGit(runtime, root, "2.53.0"); err == nil || !strings.Contains(err.Error(), "outside installation") {
		t.Fatalf("external helper accepted: %v", err)
	}
}

func TestGitVerifyExecPathCommandFailure(t *testing.T) {
	runtime, _ := platformTestRuntime(t, "windows", func(_ string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
		if args[0] == "--version" {
			return core.CommandResult{Stdout: "git version 2.53.0.windows.1"}, nil
		}
		return core.CommandResult{Code: 19, Stderr: "exec-path unavailable"}, nil
	})
	if err := verifyGit(runtime, t.TempDir(), "2.53.0"); err == nil || !strings.Contains(err.Error(), "exit 19") || !strings.Contains(err.Error(), "exec-path unavailable") {
		t.Fatalf("command diagnosis missing: %v", err)
	}
}
