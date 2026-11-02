package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

type runtimeSpec struct {
	name, category string
	directory      func(*core.Runtime, core.Instance) (string, error)
	bin            func(*core.Runtime, string) string
	check          func(*core.Runtime, core.Instance, string) (core.DetectResult, error)
	release        func(*core.Runtime, core.Instance) (ArchiveRelease, error)
	selectRoot     func(*core.Runtime, string) (string, error)
	expose         func(*core.Runtime, string)
}

func simpleRuntime(spec runtimeSpec) core.Plugin {
	bin := func(r *core.Runtime, root string) string {
		if spec.bin != nil {
			return spec.bin(r, root)
		}
		return filepath.Join(root, "bin")
	}
	directory := func(r *core.Runtime, i core.Instance) (string, error) {
		if spec.directory != nil {
			return spec.directory(r, i)
		}
		return managedDirectory(r, i, spec.name)
	}
	expose := func(r *core.Runtime, root string) {
		r.Env.PrependPath(bin(r, root))
		if spec.expose != nil {
			spec.expose(r, root)
		}
	}
	detect := func(r *core.Runtime, i core.Instance) (core.DetectResult, error) {
		r.Ensure()
		root, err := directory(r, i)
		if err != nil {
			return spec.check(r, i, spec.name)
		}
		executable := filepath.Join(bin(r, root), executableName(r, spec.name))
		managed := exists(executable)
		command := spec.name
		if managed || optionString(i.Options, "install_dir") != "" {
			command = executable
		}
		result, err := spec.check(r, i, command)
		if err == nil && result.OK && managed {
			expose(r, root)
		}
		return result, err
	}
	return core.Plugin{ID: spec.name, Category: spec.category, Prepare: func(r *core.Runtime, i core.Instance) error {
		r.Ensure()
		if _, err := directory(r, i); err != nil {
			detected, detectErr := spec.check(r, i, spec.name)
			if detectErr != nil {
				return detectErr
			}
			if !detected.OK {
				return err
			}
		}
		return nil
	}, ParseOptions: func(o map[string]any) (map[string]any, error) { return ParseOptions(spec.name, o) }, Detect: detect, Apply: func(r *core.Runtime, i core.Instance) (core.ApplyResult, error) {
		r.Ensure()
		found, err := detect(r, i)
		if err != nil {
			return core.ApplyResult{}, err
		}
		if found.OK {
			return core.ApplyResult{OK: true, Details: found.Details}, nil
		}
		if spec.name == "java" && optionString(i.Options, "distribution") == "oracle" {
			return core.ApplyResult{Details: "Oracle JDK requires manual download and license acceptance"}, nil
		}
		destination, err := directory(r, i)
		if err != nil {
			return core.ApplyResult{}, err
		}
		release, err := spec.release(r, i)
		if err == nil {
			err = InstallArchive(r, destination, filepath.Join(strings.TrimPrefix(bin(r, ""), string(filepath.Separator)), executableName(r, spec.name)), release, func(root string) (string, error) {
				if spec.selectRoot != nil {
					return spec.selectRoot(r, root)
				}
				return SingleDirectory(root)
			}, func(root string) error {
				result, e := spec.check(r, i, filepath.Join(bin(r, root), executableName(r, spec.name)))
				if e != nil {
					return e
				}
				if !result.OK {
					return errors.New(result.Details)
				}
				return nil
			})
			if err == nil {
				expose(r, destination)
				r.Logger.Printf("Add %s to shell PATH for future sessions", bin(r, destination))
				return core.ApplyResult{OK: true, DidChange: true, Details: fmt.Sprintf("%s %s installed to %s", spec.name, optionString(i.Options, "version"), destination)}, nil
			}
		}
		var recovery *InstallationRecoveryError
		return failedApply(r, fmt.Sprintf("%s installation failed: %v", spec.name, err), errors.As(err, &recovery), err)
	}, Validate: func(r *core.Runtime, i core.Instance) (core.ValidateResult, error) {
		result, err := detect(r, i)
		return core.ValidateResult{OK: result.OK, Message: result.Details}, err
	}}
}
func checkVersion(r *core.Runtime, i core.Instance, command, name string, args []string, pattern *regexp.Regexp) (core.DetectResult, error) {
	result, err := r.Run(command, args...)
	if err != nil {
		return core.DetectResult{}, err
	}
	output := result.Stdout
	if output == "" {
		output = result.Stderr
	}
	match := pattern.FindStringSubmatch(strings.TrimSpace(output))
	if result.Code != 0 || len(match) < 2 {
		return core.DetectResult{Details: name + " is unavailable or version could not be determined"}, nil
	}
	version := match[1]
	return core.DetectResult{OK: MatchesVersion(version, optionString(i.Options, "version")), Details: fmt.Sprintf("Detected %s %s; requested %s", name, version, optionString(i.Options, "version"))}, nil
}

var bunVersion = regexp.MustCompile(`^(\d+\.\d+\.\d+)(?:\s|$)`)
var denoVersion = regexp.MustCompile(`^deno (\d+\.\d+\.\d+)(?:\s|$)`)
var goVersion = regexp.MustCompile(`go version go(\d+\.\d+(?:\.\d+)?)`)

func binaryRuntime(name string) core.Plugin {
	pattern := bunVersion
	if name == "deno" {
		pattern = denoVersion
	}
	return simpleRuntime(runtimeSpec{name: name, category: "tool", check: func(r *core.Runtime, i core.Instance, command string) (core.DetectResult, error) {
		return checkVersion(r, i, command, name, []string{"--version"}, pattern)
	}, release: func(r *core.Runtime, i core.Instance) (ArchiveRelease, error) {
		if name == "bun" {
			return BunRelease(r.Context, optionString(i.Options, "version"), r.Platform, r.Arch, optionString(i.Options, "libc"))
		}
		return DenoRelease(r.Context, optionString(i.Options, "version"), r.Platform, r.Arch)
	}, selectRoot: func(r *core.Runtime, root string) (string, error) {
		source := root
		var err error
		if name == "bun" {
			source, err = SingleDirectory(root)
			if err != nil {
				return "", err
			}
		}
		destination := filepath.Join(root, "runtime")
		if err = os.MkdirAll(filepath.Join(destination, "bin"), 0755); err != nil {
			return "", err
		}
		exe := executableName(r, name)
		target := filepath.Join(destination, "bin", exe)
		if err = os.Rename(filepath.Join(source, exe), target); err != nil {
			return "", err
		}
		if r.Platform != "windows" {
			if err = os.Chmod(target, 0755); err != nil {
				return "", err
			}
		}
		return destination, nil
	}, expose: func(r *core.Runtime, root string) { r.Env.Set(strings.ToUpper(name)+"_INSTALL", root) }})
}
func Bun() core.Plugin  { return binaryRuntime("bun") }
func Deno() core.Plugin { return binaryRuntime("deno") }
func Go() core.Plugin {
	return simpleRuntime(runtimeSpec{name: "go", category: "language", directory: func(r *core.Runtime, i core.Instance) (string, error) {
		if _, exists := i.Options["install_dir"]; exists {
			return managedDirectory(r, i, "go")
		}
		if r.Platform == "windows" {
			return managedDirectory(r, i, "go")
		}
		return "/usr/local/go", nil
	}, check: func(r *core.Runtime, i core.Instance, command string) (core.DetectResult, error) {
		return checkVersion(r, i, command, "Go", []string{"version"}, goVersion)
	}, release: func(r *core.Runtime, i core.Instance) (ArchiveRelease, error) {
		return GoRelease(r.Context, optionString(i.Options, "version"), r.Platform, r.Arch)
	}, expose: func(r *core.Runtime, root string) { r.Env.Set("GOROOT", root) }})
}

var javaVersion = regexp.MustCompile(`(?:version "|build )(\d+(?:\.\d+)*(?:_\d+)?)`)
var javacVersion = regexp.MustCompile(`^javac (\d+(?:\.\d+)*(?:_\d+)?)`)

func Java() core.Plugin {
	return simpleRuntime(runtimeSpec{name: "java", category: "language", directory: func(r *core.Runtime, i core.Instance) (string, error) {
		if _, exists := i.Options["install_dir"]; exists {
			return managedDirectory(r, i, "java")
		}
		parent := "/opt/java"
		if r.Platform == "windows" {
			var err error
			parent, err = managedDirectory(r, i, "java")
			if err != nil {
				return "", err
			}
		}
		if r.Platform == "macos" {
			parent = "/usr/local/java"
		}
		return filepath.Join(parent, "jdk-"+optionString(i.Options, "version")), nil
	}, check: checkJava, release: func(r *core.Runtime, i core.Instance) (ArchiveRelease, error) {
		return JavaRelease(r.Context, optionString(i.Options, "version"), r.Platform, r.Arch)
	}, selectRoot: func(r *core.Runtime, root string) (string, error) {
		directory, err := SingleDirectory(root)
		if err != nil {
			return "", err
		}
		if r.Platform == "macos" {
			directory = filepath.Join(directory, "Contents", "Home")
		}
		return directory, nil
	}, expose: func(r *core.Runtime, root string) { r.Env.Set("JAVA_HOME", root) }})
}
func checkJava(r *core.Runtime, i core.Instance, command string) (core.DetectResult, error) {
	result, err := r.Run(command, "-version")
	if err != nil {
		return core.DetectResult{}, err
	}
	output := result.Stderr
	if output == "" {
		output = result.Stdout
	}
	match := javaVersion.FindStringSubmatch(output)
	if result.Code != 0 || len(match) < 2 {
		return core.DetectResult{Details: "Java is unavailable or version could not be determined"}, nil
	}
	version := match[1]
	requested := NormalizeJavaVersion(optionString(i.Options, "version"))
	if !MatchesVersion(NormalizeJavaVersion(version), requested) {
		return core.DetectResult{Details: fmt.Sprintf("Detected Java %s; requested %s", version, requested)}, nil
	}
	compiler := "javac"
	if command != "java" {
		compiler = filepath.Join(filepath.Dir(command), executableName(r, "javac"))
	}
	compiled, err := r.Run(compiler, "-version")
	if err != nil {
		return core.DetectResult{}, err
	}
	compilerOutput := compiled.Stdout
	if compilerOutput == "" {
		compilerOutput = compiled.Stderr
	}
	compilerMatch := javacVersion.FindStringSubmatch(strings.TrimSpace(compilerOutput))
	if compiled.Code != 0 || len(compilerMatch) < 2 {
		return core.DetectResult{Details: "A Java JDK is required; javac is unavailable or unrecognized"}, nil
	}
	return core.DetectResult{OK: MatchesVersion(NormalizeJavaVersion(compilerMatch[1]), requested), Details: fmt.Sprintf("Detected Java %s and javac %s; requested %s", version, compilerMatch[1], requested)}, nil
}
