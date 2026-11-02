package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

var gitVersionPattern = regexp.MustCompile(`git version (\d+\.\d+\.\d+)`)

func gitDirectory(runtime *core.Runtime, instance core.Instance) string {
	return managedDirectory(runtime, instance, "git")
}
func gitExecutable(runtime *core.Runtime, root string) string {
	if runtime.Platform == "windows" {
		return filepath.Join(root, "cmd", "git.exe")
	}
	return filepath.Join(root, "bin", "git")
}
func checkGit(runtime *core.Runtime, command, requested string) (core.DetectResult, error) {
	result, err := runtime.Run(command, "--version")
	if err != nil || result.Code != 0 {
		return core.DetectResult{Details: "Git is not available on PATH or in its installation directory"}, platformContextError(runtime)
	}
	match := gitVersionPattern.FindStringSubmatch(platformOutput(result))
	if match == nil {
		return core.DetectResult{Details: "Git version could not be determined"}, nil
	}
	if requested == "latest" || MatchesVersion(match[1], requested) {
		return core.DetectResult{OK: true, Details: "Detected Git " + match[1]}, nil
	}
	return core.DetectResult{Details: fmt.Sprintf("Detected Git %s but %s is requested", match[1], requested)}, nil
}
func detectGit(runtime *core.Runtime, instance core.Instance) (core.DetectResult, error) {
	managed := optionString(instance.Options, "install_method") != "package"
	target := gitExecutable(runtime, gitDirectory(runtime, instance))
	found := managed && exists(target)
	command := "git"
	if found || (managed && optionString(instance.Options, "install_dir") != "") {
		command = target
	}
	detected, err := checkGit(runtime, command, optionString(instance.Options, "version"))
	if detected.OK && found {
		runtime.Env.PrependPath(filepath.Dir(target))
	}
	return detected, err
}
func gitUnsupported(runtime *core.Runtime, instance core.Instance) string {
	switch method := optionString(instance.Options, "install_method"); {
	case method == "binary" && runtime.Platform != "windows":
		return "Git publishes no standalone Unix binaries; use package or source installation"
	case method == "source" && runtime.Platform == "windows":
		return "Git source builds on Windows are unsupported; use install_method: binary for MinGit"
	case method == "package" && runtime.Platform == "windows":
		return "Use install_method: binary for automatic MinGit installation on Windows, or preinstall Git"
	}
	return ""
}
func Git() core.Plugin {
	return core.Plugin{ID: "git", Category: "tool", ParseOptions: func(options map[string]any) (map[string]any, error) { return ParseOptions("git", options) }, Detect: detectGit,
		Prepare: func(runtime *core.Runtime, instance core.Instance) error {
			reason := gitUnsupported(runtime, instance)
			if reason == "" {
				return nil
			}
			detected, err := detectGit(runtime, instance)
			if err != nil {
				return err
			}
			if !detected.OK {
				return errors.New(reason)
			}
			return nil
		},
		RegisterTasks: func(runtime *core.Runtime, instance core.Instance) error {
			if gitUnsupported(runtime, instance) != "" || optionString(instance.Options, "install_method") == "binary" {
				return nil
			}
			detected, err := detectGit(runtime, instance)
			if err != nil || detected.OK {
				return err
			}
			packages := []string{"git"}
			if optionString(instance.Options, "install_method") == "source" {
				packages = []string{"make"}
				if runtime.Platform != "macos" {
					packages = []string{"build-essential", "libcurl4-openssl-dev", "zlib1g-dev", "libexpat1-dev", "perl"}
				}
			}
			return core.RegisterPackageTasks(runtime, packages...)
		}, Apply: applyGit, Validate: func(runtime *core.Runtime, instance core.Instance) (core.ValidateResult, error) {
			detected, err := detectGit(runtime, instance)
			return core.ValidateResult{OK: detected.OK, Message: detected.Details}, err
		}}
}
func applyGit(runtime *core.Runtime, instance core.Instance) (core.ApplyResult, error) {
	detected, err := detectGit(runtime, instance)
	if err != nil {
		return core.ApplyResult{}, err
	}
	if detected.OK {
		return core.ApplyResult{OK: true, Details: detected.Details}, nil
	}
	if reason := gitUnsupported(runtime, instance); reason != "" {
		return core.ApplyResult{Details: reason}, nil
	}
	if optionString(instance.Options, "install_method") == "package" {
		return core.ApplyResult{Details: "Git is still unavailable after package installation: " + detected.Details + "; use source installation for a release pin"}, nil
	}
	windows := runtime.Platform == "windows"
	var release ArchiveRelease
	if windows {
		release, err = GitWindowsRelease(runtime.Context, optionString(instance.Options, "version"), runtime.Arch)
	} else {
		release, err = GitSourceRelease(runtime.Context, optionString(instance.Options, "version"))
	}
	destination := gitDirectory(runtime, instance)
	if err == nil {
		executable, _ := filepath.Rel(destination, gitExecutable(runtime, destination))
		err = InstallArchive(runtime, destination, executable, release, func(root string) (string, error) {
			if windows {
				return root, nil
			}
			return buildGit(runtime, root)
		}, func(root string) error { return verifyGit(runtime, root, release.Version) })
	}
	if err != nil {
		var recovery *InstallationRecoveryError
		return core.ApplyResult{DidChange: errors.As(err, &recovery), Details: "Git installation failed: " + err.Error()}, nil
	}
	runtime.Env.PrependPath(filepath.Dir(gitExecutable(runtime, destination)))
	runtime.Logger.Printf("Add %s to your shell PATH for future sessions.", filepath.Dir(gitExecutable(runtime, destination)))
	return core.ApplyResult{OK: true, DidChange: true, Details: fmt.Sprintf("Git %s installed to %s", release.Version, destination)}, nil
}
func buildGit(runtimeContext *core.Runtime, root string) (string, error) {
	extracted, err := SingleDirectory(root)
	if err != nil {
		return "", err
	}
	staged := filepath.Join(root, "install")
	if strings.ContainsAny(os.TempDir(), " \t\r\n") {
		return "", fmt.Errorf("Git source builds require a temporary directory without whitespace; set TMPDIR")
	}
	work, err := os.MkdirTemp("", "genesis-git-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(work); err != nil {
			runtimeContext.Logger.Printf("Could not remove Git build directory %s: %v", work, err)
		}
	}()
	source := filepath.Join(work, "source")
	if err := platformCopyTree(runtimeContext.Context, extracted, source); err != nil {
		return "", err
	}
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	args := []string{"prefix=/usr/local", "RUNTIME_PREFIX=YesPlease", "NO_GETTEXT=YesPlease", "NO_TCLTK=YesPlease", "NO_OPENSSL=YesPlease"}
	steps := [][]string{append(append([]string{fmt.Sprintf("-j%d", workers)}, args...), "all"), append(append([]string{}, args...), "DESTDIR="+staged, "install")}
	for _, step := range steps {
		result, err := platformRunAt(runtimeContext, source, "make", step...)
		if err != nil || result.Code != 0 {
			return "", fmt.Errorf("Git build failed: %s", platformFailure(result, err))
		}
	}
	return filepath.Join(staged, "usr", "local"), nil
}
func verifyGit(runtime *core.Runtime, root, version string) error {
	executable := gitExecutable(runtime, root)
	detected, err := checkGit(runtime, executable, version)
	if err != nil {
		return err
	}
	if !detected.OK {
		return errors.New(detected.Details)
	}
	helpers, err := runtime.Run(executable, "--exec-path")
	if err != nil || helpers.Code != 0 {
		return fmt.Errorf("Git HTTPS helper is unavailable inside the installation directory")
	}
	helperDirectory := strings.TrimSpace(helpers.Stdout)
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolvedHelpers, err := filepath.EvalSymlinks(helperDirectory)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedHelpers)
	helper := "git-remote-https"
	if runtime.Platform == "windows" {
		helper += ".exe"
	}
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || !exists(filepath.Join(helperDirectory, helper)) {
		return fmt.Errorf("Git HTTPS helper is unavailable inside the installation directory")
	}
	return nil
}
