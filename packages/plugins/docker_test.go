package plugins

import (
	"strings"
	"testing"
	"time"

	"github.com/ossl-dev/genesis/packages/core"
)

func dockerTestRunner(client, modern, legacy, daemon string) func(string, []string, core.CommandOptions) (core.CommandResult, error) {
	return func(command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
		output := ""
		switch {
		case command == "docker" && args[0] == "--version":
			output = client
		case command == "docker" && args[0] == "compose":
			output = modern
		case command == "docker-compose":
			output = legacy
		case command == "docker" && args[0] == "info":
			output = daemon
		}
		if output == "" {
			return core.CommandResult{Code: 1, Stderr: "unavailable"}, nil
		}
		return core.CommandResult{Stdout: output}, nil
	}
}
func TestDockerDetection(t *testing.T) {
	for _, test := range []struct {
		name, requested, client, modern, legacy string
		include, ok, missing                    bool
	}{
		{"modern", "29.1", "Docker version 29.1.0", "Docker Compose version v2.40.0", "", true, true, false},
		{"legacy", "latest", "Docker version 29.1.0", "", "docker-compose version 1.29.2, build x", true, true, false},
		{"malformed modern falls back", "latest", "Docker version 29.1.0", "weird", "Docker Compose version v2.40.0", true, true, false},
		{"optional compose", "latest", "Docker version 29.1.0", "", "", false, true, false},
		{"missing compose", "latest", "Docker version 29.1.0", "", "", true, false, true},
		{"version collision", "2", "Docker version 29.1.0", "", "", false, false, false},
		{"missing client", "latest", "", "", "", true, false, false},
		{"unknown client version", "latest", "strange", "", "", true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, "linux", dockerTestRunner(test.client, test.modern, test.legacy, "29.1.0"))
			instance := platformTestInstance(t, Docker(), map[string]any{"version": test.requested, "include_compose": test.include})
			result, err := detectDocker(runtime, instance)
			if err != nil || result.OK != test.ok || result.ComposeMissing != test.missing {
				t.Fatalf("%+v %v", result, err)
			}
			if !test.include && len(*commands) != 1 {
				t.Fatal(*commands)
			}
		})
	}
}
func TestDockerDaemonChecksExistingClientAndTimeout(t *testing.T) {
	for _, test := range []struct {
		name, daemon, expected string
		ok                     bool
	}{{"ready", "29.1.0", "daemon 29.1.0", true}, {"missing", "", "daemon is unavailable", false}, {"wrong daemon", "28.5.1", "does not match", false}, {"malformed", "latest", "could not be determined", false}} {
		t.Run(test.name, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, "linux", dockerTestRunner("Docker version 29.1.0", "Docker Compose version v2.40.0", "", test.daemon))
			instance := platformTestInstance(t, Docker(), map[string]any{"version": "29.1"})
			result, err := Docker().Apply(runtime, instance)
			if err != nil || result.OK != test.ok || result.DidChange || !strings.Contains(result.Details, test.expected) {
				t.Fatalf("%+v %v", result, err)
			}
			last := (*commands)[len(*commands)-1]
			if last.Command != "docker" || last.Args[0] != "info" || last.Options.Timeout != 15*time.Second {
				t.Fatal(last)
			}
			validated, err := Docker().Validate(runtime, instance)
			if err != nil || validated.OK != test.ok {
				t.Fatalf("%+v %v", validated, err)
			}
		})
	}
}
func TestDockerExistingSkipsPrerequisites(t *testing.T) {
	runtime, _ := platformTestRuntime(t, "linux", dockerTestRunner("Docker version 29.1.0", "Docker Compose version v2.40.0", "", "29.1.0"))
	instance := platformTestInstance(t, Docker(), nil)
	if err := Docker().Prepare(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if err := Docker().RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if len(runtime.Tasks.IDs()) != 0 {
		t.Fatal(runtime.Tasks.IDs())
	}
}
func TestDockerUnsupportedInstallPreflight(t *testing.T) {
	for _, test := range []struct {
		platform string
		options  map[string]any
		message  string
	}{{"windows", map[string]any{}, "manually"}, {"macos", map[string]any{"version": "29.1"}, "pinning"}, {"macos", map[string]any{"install_desktop": true}, "manually"}} {
		t.Run(test.platform+test.message, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, test.platform, nil)
			instance := platformTestInstance(t, Docker(), test.options)
			if err := Docker().Prepare(runtime, instance); err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatal(err)
			}
			if err := Docker().RegisterTasks(runtime, instance); err != nil || len(runtime.Tasks.IDs()) != 0 {
				t.Fatalf("%v %+v", err, runtime.Tasks.IDs())
			}
			result, err := Docker().Apply(runtime, instance)
			if err != nil || result.OK || result.DidChange || !strings.Contains(result.Details, test.message) {
				t.Fatalf("%+v %v", result, err)
			}
			for _, command := range *commands {
				if command.Command != "docker" {
					t.Fatal(command)
				}
			}
		})
	}
}
func TestDockerColimaStartsAfterPrerequisites(t *testing.T) {
	running := false
	runner := dockerTestRunner("Docker version 29.1.0", "Docker Compose version v2.40.0", "", "")
	runtime, commands := platformTestRuntime(t, "macos", func(command string, args []string, options core.CommandOptions) (core.CommandResult, error) {
		if command == "colima" {
			running = true
			return core.CommandResult{}, nil
		}
		if running {
			return dockerTestRunner("Docker version 29.1.0", "Docker Compose version v2.40.0", "", "29.1.0")(command, args, options)
		}
		return runner(command, args, options)
	})
	instance := platformTestInstance(t, Docker(), nil)
	if err := Docker().RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	ids := strings.Join(runtime.Tasks.IDs(), ",")
	if !strings.Contains(ids, "colima") || strings.Contains(ids, "install:docker") {
		t.Fatal(ids)
	}
	result, err := Docker().Apply(runtime, instance)
	if err != nil || !result.OK || !result.DidChange || !running {
		t.Fatalf("%+v %v", result, err)
	}
	for _, command := range *commands {
		if command.Command == "brew" {
			t.Fatal("apply repeated package installation")
		}
	}
}
func TestDockerColimaFailureIsChanged(t *testing.T) {
	runtime, _ := platformTestRuntime(t, "macos", nil)
	result, err := Docker().Apply(runtime, platformTestInstance(t, Docker(), nil))
	if err != nil || result.OK || !result.DidChange || !strings.Contains(result.Details, "colima failed") {
		t.Fatalf("%+v %v", result, err)
	}
}
