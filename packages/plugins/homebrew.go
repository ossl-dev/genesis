package plugins

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/ossl-dev/genesis/packages/core"
)

var brewVersionPattern = regexp.MustCompile(`Homebrew (\d+\.\d+\.\d+)`)

func detectHomebrew(runtime *core.Runtime, _ core.Instance) (core.DetectResult, error) {
	result, err := runtime.Run("brew", "--version")
	if err != nil || result.Code != 0 {
		return core.DetectResult{Details: "Homebrew is not available on PATH"}, platformContextError(runtime)
	}
	match := brewVersionPattern.FindStringSubmatch(platformOutput(result))
	if match == nil {
		return core.DetectResult{Details: "Homebrew version could not be determined"}, nil
	}
	return core.DetectResult{OK: true, Details: "Detected Homebrew " + match[1]}, nil
}
func installHomebrew(runtime *core.Runtime, instance core.Instance) (string, bool, error) {
	prefix := "/usr/local"
	if runtime.Arch == "arm64" {
		prefix = "/opt/homebrew"
	} else if runtime.Arch != "amd64" {
		return "", false, fmt.Errorf("unsupported Homebrew architecture: %s", runtime.Arch)
	}
	if err := platformCheck(runtime, "bash", "-c", "set -o pipefail; curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | bash"); err != nil {
		return "", true, fmt.Errorf("Homebrew installation failed: %w", err)
	}
	bin := filepath.Join(prefix, "bin")
	runtime.Env.PrependPath(bin)
	if optionBool(instance.Options, "add_to_path") {
		runtime.Logger.Printf("Add eval \"$(%s/brew shellenv)\" to your shell profile for future sessions.", bin)
	}
	return "Homebrew installed to " + prefix, true, nil
}
func updateHomebrew(runtime *core.Runtime, instance core.Instance) error {
	if err := platformCheck(runtime, "brew", "update"); err != nil {
		return err
	}
	if optionBool(instance.Options, "update_packages") {
		if err := platformCheck(runtime, "brew", "upgrade"); err != nil {
			return err
		}
		if optionBool(instance.Options, "install_cask") {
			return platformCheck(runtime, "brew", "upgrade", "--cask")
		}
	}
	return nil
}
func Homebrew() core.Plugin {
	return core.Plugin{ID: "homebrew", Category: "tool", ParseOptions: func(options map[string]any) (map[string]any, error) { return ParseOptions("homebrew", options) }, Detect: detectHomebrew,
		Prepare: func(runtime *core.Runtime, instance core.Instance) error {
			if runtime.Platform != "macos" {
				return nil
			}
			detected, err := detectHomebrew(runtime, instance)
			if err != nil || detected.OK {
				return err
			}
			if _, _, err := installHomebrew(runtime, instance); err != nil {
				return err
			}
			detected, err = detectHomebrew(runtime, instance)
			if err != nil {
				return err
			}
			if !detected.OK {
				return fmt.Errorf("Homebrew installation failed verification: %s", detected.Details)
			}
			return nil
		}, PostApply: func(runtime *core.Runtime, instance core.Instance) error {
			packages, _ := stringSlice(instance.Options["global_packages"])
			if len(packages) == 0 {
				return nil
			}
			return platformCheck(runtime, "brew", append([]string{"install"}, packages...)...)
		}, Apply: func(runtime *core.Runtime, instance core.Instance) (core.ApplyResult, error) {
			if runtime.Platform != "macos" {
				return core.ApplyResult{Details: "Homebrew is not supported on " + runtime.Platform}, nil
			}
			detected, err := detectHomebrew(runtime, instance)
			if err != nil {
				return core.ApplyResult{}, err
			}
			details := "Homebrew updated successfully"
			if !detected.OK {
				var attempted bool
				details, attempted, err = installHomebrew(runtime, instance)
				if err != nil {
					return failedApply(runtime, err.Error(), attempted, err)
				}
				verified, err := detectHomebrew(runtime, instance)
				if err != nil {
					return core.ApplyResult{DidChange: true}, err
				}
				if !verified.OK {
					return core.ApplyResult{DidChange: true, Details: verified.Details}, nil
				}
			}
			if err := updateHomebrew(runtime, instance); err != nil {
				return failedApply(runtime, "Homebrew update failed: "+err.Error(), true, err)
			}
			return core.ApplyResult{OK: true, DidChange: true, Details: details}, nil
		}, Validate: func(runtime *core.Runtime, instance core.Instance) (core.ValidateResult, error) {
			detected, err := detectHomebrew(runtime, instance)
			return core.ValidateResult{OK: detected.OK, Message: detected.Details}, err
		}}
}
