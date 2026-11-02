package plugins

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ossl-dev/genesis/packages/core"
)

type dockerDetection struct {
	core.DetectResult
	ComposeMissing bool
}

var dockerVersionPattern = regexp.MustCompile(`Docker version (\d+\.\d+\.\d+)`)
var dockerComposePattern = regexp.MustCompile(`(?i)(?:Docker Compose|docker-compose) version v?(\d+\.\d+\.\d+)`)
var dockerDaemonPattern = regexp.MustCompile(`^(\d+\.\d+\.\d+)(?:[-+\s]|$)`)

func detectDocker(runtime *core.Runtime, instance core.Instance) (dockerDetection, error) {
	result, err := runtime.Run("docker", "--version")
	if err != nil || result.Code != 0 {
		return dockerDetection{DetectResult: core.DetectResult{Details: "Docker is not available on PATH"}}, platformContextError(runtime)
	}
	match := dockerVersionPattern.FindStringSubmatch(platformOutput(result))
	if match == nil {
		return dockerDetection{DetectResult: core.DetectResult{Details: "Docker version could not be determined"}}, nil
	}
	requested := optionString(instance.Options, "version")
	if requested != "latest" && !MatchesVersion(match[1], requested) {
		return dockerDetection{DetectResult: core.DetectResult{Details: fmt.Sprintf("Detected Docker %s but %s is requested", match[1], requested)}}, nil
	}
	details := "Detected Docker " + match[1]
	if !optionBool(instance.Options, "include_compose") {
		return dockerDetection{DetectResult: core.DetectResult{OK: true, Details: details}}, nil
	}
	for _, command := range [][]string{{"docker", "compose", "version"}, {"docker-compose", "--version"}} {
		compose, err := runtime.Run(command[0], command[1:]...)
		if err != nil {
			if err := platformContextError(runtime); err != nil {
				return dockerDetection{}, err
			}
			continue
		}
		composeMatch := dockerComposePattern.FindStringSubmatch(platformOutput(compose))
		if compose.Code == 0 && composeMatch != nil {
			return dockerDetection{DetectResult: core.DetectResult{OK: true, Details: details + " and Docker Compose " + composeMatch[1]}}, nil
		}
	}
	return dockerDetection{ComposeMissing: true, DetectResult: core.DetectResult{Details: details + " but Docker Compose is unavailable"}}, nil
}
func validateDocker(runtime *core.Runtime, instance core.Instance, detected dockerDetection) (core.DetectResult, error) {
	if !detected.OK {
		return detected.DetectResult, nil
	}
	runtime.Ensure()
	ctx, cancel := context.WithTimeout(runtime.Context, 15*time.Second)
	defer cancel()
	result, err := runtime.Runner(ctx, "docker", []string{"info", "--format", "{{.ServerVersion}}"}, core.CommandOptions{Cwd: runtime.Cwd, Env: runtime.Env.Snapshot(), Timeout: 15 * time.Second})
	if err != nil || result.Code != 0 {
		if runtime.Context.Err() != nil {
			return core.DetectResult{}, runtime.Context.Err()
		}
		return core.DetectResult{Details: "Docker daemon is unavailable: " + platformFailure(result, err) + "; start Docker and check socket permissions"}, nil
	}
	match := dockerDaemonPattern.FindStringSubmatch(strings.TrimSpace(result.Stdout))
	if match == nil {
		return core.DetectResult{Details: "Docker daemon version could not be determined"}, nil
	}
	requested := optionString(instance.Options, "version")
	if requested != "latest" && !MatchesVersion(match[1], requested) {
		return core.DetectResult{Details: fmt.Sprintf("Docker daemon %s does not match requested %s", match[1], requested)}, nil
	}
	return core.DetectResult{OK: true, Details: detected.Details + "; daemon " + match[1] + " is available"}, nil
}
func dockerUnsupported(runtime *core.Runtime, instance core.Instance) string {
	if runtime.Platform == "windows" || optionBool(instance.Options, "install_desktop") {
		return "Install and start Docker Desktop manually (https://docs.docker.com/desktop/), then rerun Genesis"
	}
	if runtime.Platform == "macos" && optionString(instance.Options, "version") != "latest" {
		return "Automatic Docker version pinning on macOS is unsupported; preinstall a matching client and daemon or use version: latest"
	}
	return ""
}
func Docker() core.Plugin {
	return core.Plugin{ID: "docker", Category: "tool", ParseOptions: func(options map[string]any) (map[string]any, error) { return ParseOptions("docker", options) },
		Detect: func(runtime *core.Runtime, instance core.Instance) (core.DetectResult, error) {
			detected, err := detectDocker(runtime, instance)
			return detected.DetectResult, err
		},
		Prepare: func(runtime *core.Runtime, instance core.Instance) error {
			detected, err := detectDocker(runtime, instance)
			if err != nil || detected.OK {
				return err
			}
			if reason := dockerUnsupported(runtime, instance); reason != "" {
				return errors.New(reason)
			}
			if runtime.Platform == "linux" {
				_, err = DockerDistribution(runtime)
			}
			return err
		}, RegisterTasks: func(runtime *core.Runtime, instance core.Instance) error {
			if dockerUnsupported(runtime, instance) != "" {
				return nil
			}
			detected, err := detectDocker(runtime, instance)
			if err != nil {
				return err
			}
			if detected.OK {
				if runtime.Platform != "macos" {
					return nil
				}
				verified, err := validateDocker(runtime, instance, detected)
				if err != nil || verified.OK {
					return err
				}
			}
			if runtime.Platform == "linux" {
				if _, err := DockerDistribution(runtime); err != nil {
					return err
				}
			}
			packages := []string{"ca-certificates"}
			if runtime.Platform == "macos" {
				packages = []string{"colima"}
				if !detected.OK {
					packages = append(packages, "docker")
					if optionBool(instance.Options, "include_compose") {
						packages = append(packages, "docker-compose")
					}
				}
			}
			return core.RegisterPackageTasks(runtime, packages...)
		}, Apply: applyDocker, Validate: func(runtime *core.Runtime, instance core.Instance) (core.ValidateResult, error) {
			detected, err := detectDocker(runtime, instance)
			if err != nil {
				return core.ValidateResult{}, err
			}
			verified, err := validateDocker(runtime, instance, detected)
			return core.ValidateResult{OK: verified.OK, Message: verified.Details}, err
		}}
}
func applyDocker(runtime *core.Runtime, instance core.Instance) (core.ApplyResult, error) {
	detected, err := detectDocker(runtime, instance)
	if err != nil {
		return core.ApplyResult{}, err
	}
	if detected.OK {
		validated, err := validateDocker(runtime, instance, detected)
		if err != nil {
			return core.ApplyResult{}, err
		}
		if validated.OK || runtime.Platform != "macos" || dockerUnsupported(runtime, instance) != "" || !strings.HasPrefix(validated.Details, "Docker daemon is unavailable") {
			return core.ApplyResult{OK: validated.OK, Details: validated.Details}, nil
		}
	}
	if reason := dockerUnsupported(runtime, instance); reason != "" {
		return core.ApplyResult{Details: reason}, nil
	}
	didChange := false
	if runtime.Platform == "macos" {
		didChange = true
		err = platformCheck(runtime, "colima", "start")
	} else {
		err = InstallDockerLinux(runtime, instance, func() { didChange = true }, detected.ComposeMissing)
	}
	if err != nil {
		if runtime.Context.Err() != nil {
			return core.ApplyResult{DidChange: didChange}, runtime.Context.Err()
		}
		return core.ApplyResult{DidChange: didChange, Details: "Docker installation failed: " + err.Error()}, nil
	}
	installed, err := detectDocker(runtime, instance)
	if err != nil {
		return core.ApplyResult{DidChange: didChange}, err
	}
	validated, err := validateDocker(runtime, instance, installed)
	return core.ApplyResult{OK: validated.OK, DidChange: didChange, Details: validated.Details}, err
}
