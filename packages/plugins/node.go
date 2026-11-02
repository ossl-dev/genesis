package plugins

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

var nodeVersion = regexp.MustCompile(`^v?(\d+\.\d+\.\d+)$`)

func nvmDirectory(r *core.Runtime) (string, error) {
	if directory := r.Env.Get("NVM_DIR"); directory != "" {
		return checkedInstallDirectory(directory, "NVM_DIR")
	}
	if directory := r.Env.Get("XDG_CONFIG_HOME"); directory != "" {
		home, err := checkedInstallDirectory(directory, "XDG_CONFIG_HOME")
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "nvm"), nil
	}
	home, err := runtimeHome(r)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nvm"), nil
}

func Node() core.Plugin {
	standalone := simpleRuntime(runtimeSpec{name: "node", category: "tool", bin: func(r *core.Runtime, root string) string {
		if r.Platform == "windows" {
			return root
		}
		return filepath.Join(root, "bin")
	}, check: func(r *core.Runtime, i core.Instance, command string) (core.DetectResult, error) {
		return checkVersion(r, i, command, "Node", []string{"-v"}, nodeVersion)
	}, release: func(r *core.Runtime, i core.Instance) (ArchiveRelease, error) {
		return NodeRelease(r.Context, optionString(i.Options, "version"), r.Platform, r.Arch)
	}})
	useNvm := func(r *core.Runtime, i core.Instance) bool {
		return r.Platform != "windows" && optionBool(i.Options, "use_nvm") && optionString(i.Options, "install_dir") == ""
	}
	detect := func(r *core.Runtime, i core.Instance) (core.DetectResult, error) {
		r.Ensure()
		if useNvm(r, i) {
			return checkVersion(r, i, "node", "Node", []string{"-v"}, nodeVersion)
		}
		return standalone.Detect(r, i)
	}
	plugin := standalone
	plugin.Detect = detect
	plugin.Prepare = func(r *core.Runtime, i core.Instance) error {
		r.Ensure()
		if !useNvm(r, i) {
			return standalone.Prepare(r, i)
		}
		detected, err := detect(r, i)
		if err != nil || detected.OK {
			return err
		}
		_, err = nvmDirectory(r)
		return err
	}
	plugin.RegisterTasks = func(r *core.Runtime, i core.Instance) error {
		r.Ensure()
		if !useNvm(r, i) {
			return nil
		}
		found, err := detect(r, i)
		if err != nil || found.OK {
			return err
		}
		if _, err := nvmDirectory(r); err != nil {
			return err
		}
		return core.RegisterPackageTasks(r, "curl")
	}
	plugin.Apply = func(r *core.Runtime, i core.Instance) (core.ApplyResult, error) {
		r.Ensure()
		found, err := detect(r, i)
		if err != nil {
			return core.ApplyResult{}, err
		}
		if found.OK {
			return core.ApplyResult{OK: true, Details: found.Details}, nil
		}
		if !useNvm(r, i) {
			return standalone.Apply(r, i)
		}
		directory, err := nvmDirectory(r)
		if err != nil {
			return core.ApplyResult{}, err
		}
		script := filepath.Join(directory, "nvm.sh")
		changed := false
		if !exists(script) {
			result, err := r.Run("bash", "-c", "set -o pipefail; curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash")
			if err != nil {
				return core.ApplyResult{}, err
			}
			if result.Code != 0 {
				return core.ApplyResult{Details: "NVM installation failed: " + result.Stderr}, nil
			}
			changed = true
		}
		version := optionString(i.Options, "version")
		commands := [][]string{{"-c", `source "$1" && nvm install "$2"`, "genesis-nvm", script, version}, {"-c", `source "$1" && nvm alias default "$2"`, "genesis-nvm", script, version}, {"-c", `source "$1" && nvm which "$2"`, "genesis-nvm", script, version}}
		var executable string
		for _, args := range commands {
			result, err := r.Run("bash", args...)
			if err != nil {
				return core.ApplyResult{DidChange: changed}, err
			}
			if result.Code != 0 {
				return core.ApplyResult{DidChange: true, Details: "Node NVM installation failed: " + result.Stderr}, nil
			}
			changed = true
			executable = strings.TrimSpace(result.Stdout)
		}
		if !filepath.IsAbs(executable) {
			return core.ApplyResult{DidChange: true, Details: "Node installed but executable could not be resolved"}, nil
		}
		r.Env.PrependPath(filepath.Dir(executable))
		verified, err := checkVersion(r, i, executable, "Node", []string{"-v"}, nodeVersion)
		if err != nil {
			return core.ApplyResult{DidChange: true}, err
		}
		return core.ApplyResult{OK: verified.OK, DidChange: true, Details: verified.Details}, nil
	}
	plugin.PostApply = func(r *core.Runtime, i core.Instance) error {
		packages, _ := stringSlice(i.Options["global_packages"])
		if len(packages) == 0 {
			return nil
		}
		result, err := r.Run("npm", append([]string{"install", "-g", "--"}, packages...)...)
		if err != nil {
			return err
		}
		if result.Code != 0 {
			return fmt.Errorf("failed to install npm packages: %s", result.Stderr)
		}
		return nil
	}
	plugin.Validate = func(r *core.Runtime, i core.Instance) (core.ValidateResult, error) {
		result, err := detect(r, i)
		return core.ValidateResult{OK: result.OK, Message: result.Details}, err
	}
	return plugin
}
