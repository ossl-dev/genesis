package plugins

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

var pythonVersionPattern = regexp.MustCompile(`Python (\d+\.\d+\.\d+)`)

func detectPython(runtime *core.Runtime, instance core.Instance) (core.DetectResult, error) {
	result, err := runtime.Run("python3", "--version")
	if err != nil || result.Code != 0 {
		return core.DetectResult{Details: "Python is not available on PATH"}, platformContextError(runtime)
	}
	match := pythonVersionPattern.FindStringSubmatch(result.Stdout + "\n" + result.Stderr)
	if match == nil {
		return core.DetectResult{Details: "Python version could not be determined"}, nil
	}
	requested := optionString(instance.Options, "version")
	if MatchesVersion(match[1], requested) {
		return core.DetectResult{OK: true, Details: "Detected Python " + match[1]}, nil
	}
	return core.DetectResult{Details: fmt.Sprintf("Detected Python %s but %s is requested", match[1], requested)}, nil
}
func Python() core.Plugin {
	return core.Plugin{ID: "python", Category: "language", ParseOptions: func(options map[string]any) (map[string]any, error) { return ParseOptions("python", options) }, Detect: detectPython,
		RegisterTasks: func(runtime *core.Runtime, instance core.Instance) error {
			if runtime.Platform == "windows" {
				return nil
			}
			detected, err := detectPython(runtime, instance)
			if err != nil || detected.OK {
				return err
			}
			parts := strings.Split(optionString(instance.Options, "version"), ".")
			if len(parts) > 2 {
				parts = parts[:2]
			}
			version := strings.Join(parts, ".")
			name := "python" + version
			if runtime.Platform == "macos" {
				name = "python@" + version
			}
			return core.RegisterPackageTasks(runtime, name)
		}, Apply: func(runtime *core.Runtime, instance core.Instance) (core.ApplyResult, error) {
			detected, err := detectPython(runtime, instance)
			if err != nil {
				return core.ApplyResult{}, err
			}
			if detected.OK {
				return core.ApplyResult{OK: true, Details: detected.Details}, nil
			}
			if runtime.Platform == "windows" {
				return core.ApplyResult{Details: "Manual installation required. Visit https://www.python.org/"}, nil
			}
			return core.ApplyResult{Details: fmt.Sprintf("Python %s is still unavailable after package installation: %s", optionString(instance.Options, "version"), detected.Details)}, nil
		}, Validate: func(runtime *core.Runtime, instance core.Instance) (core.ValidateResult, error) {
			result, err := detectPython(runtime, instance)
			return core.ValidateResult{OK: result.OK, Message: result.Details}, err
		}}
}
