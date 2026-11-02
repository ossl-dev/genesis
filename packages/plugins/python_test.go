package plugins

import (
	"context"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

func TestPythonVersionDetectionAndValidation(t *testing.T) {
	for _, test := range []struct {
		name, requested, stdout, stderr string
		code                            int
		ok                              bool
	}{{"minor", "3.12", "Python 3.12.3", "", 0, true}, {"stderr", "3.12", "", "Python 3.12.3", 0, true}, {"major collision", "3.1", "Python 3.12.3", "", 0, false}, {"patch mismatch", "3.12.5", "Python 3.12.3", "", 0, false}, {"missing", "3.12", "", "missing", 1, false}, {"malformed", "3.12", "strange", "", 0, false}} {
		t.Run(test.name, func(t *testing.T) {
			runtime, _ := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
				return core.CommandResult{Code: test.code, Stdout: test.stdout, Stderr: test.stderr}, nil
			})
			instance := platformTestInstance(t, Python(), map[string]any{"version": test.requested})
			detected, err := Python().Detect(runtime, instance)
			if err != nil || detected.OK != test.ok {
				t.Fatalf("%+v %v", detected, err)
			}
			validated, err := Python().Validate(runtime, instance)
			if err != nil || validated.OK != test.ok {
				t.Fatalf("%+v %v", validated, err)
			}
		})
	}
}
func TestPythonMissingAndPresentApply(t *testing.T) {
	for _, platform := range []string{"macos", "linux", "windows"} {
		t.Run(platform, func(t *testing.T) {
			runtime, _ := platformTestRuntime(t, platform, nil)
			instance := platformTestInstance(t, Python(), map[string]any{"version": "3.12"})
			result, err := Python().Apply(runtime, instance)
			if err != nil || result.OK || result.DidChange {
				t.Fatalf("%+v %v", result, err)
			}
			expected := "still unavailable"
			if platform == "windows" {
				expected = "Manual installation required"
			}
			if !strings.Contains(result.Details, expected) {
				t.Fatal(result.Details)
			}
		})
	}
	runtime, _ := platformTestRuntime(t, "macos", func(_ string, _ []string, _ core.CommandOptions) (core.CommandResult, error) {
		return core.CommandResult{Stdout: "Python 3.12.3"}, nil
	})
	instance := platformTestInstance(t, Python(), map[string]any{"version": "3.12"})
	if err := Python().RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if len(runtime.Tasks.IDs()) != 0 {
		t.Fatal(runtime.Tasks.IDs())
	}
	result, err := Python().Apply(runtime, instance)
	if err != nil || !result.OK || result.DidChange {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestPythonPrerequisitesAndCancellation(t *testing.T) {
	runtime, _ := platformTestRuntime(t, "macos", nil)
	instance := platformTestInstance(t, Python(), map[string]any{"version": "3.12.5"})
	if err := Python().RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(runtime.Tasks.IDs(), ","), "python@3.12") {
		t.Fatal(runtime.Tasks.IDs())
	}
	runtime.Platform = "windows"
	runtime.Tasks = core.NewTaskRegistry()
	if err := Python().RegisterTasks(runtime, instance); err != nil || len(runtime.Tasks.IDs()) != 0 {
		t.Fatalf("%v %+v", err, runtime.Tasks.IDs())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runtime.Context = ctx
	if _, err := Python().Detect(runtime, instance); err != context.Canceled {
		t.Fatal(err)
	}
}
