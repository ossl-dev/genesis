package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ossl-dev/genesis/packages/core"
)

func TestMain(m *testing.M) {
	if os.Getenv("GENESIS_EXTENSION_TEST_HELPER") == "1" {
		if os.Getenv("GENESIS_EXTENSION_TEST_MODE") == "descendant-child" {
			time.Sleep(time.Minute)
			os.Exit(0)
		}
		var request Request
		if json.NewDecoder(os.Stdin).Decode(&request) != nil {
			os.Exit(3)
		}
		reply := Response{Protocol: ProtocolVersion}
		switch os.Getenv("GENESIS_EXTENSION_TEST_MODE") {
		case "bad-version":
			reply.Protocol = 999
		case "error":
			reply.Error = "plugin rejected request"
		case "hang":
			time.Sleep(time.Minute)
		case "descendant":
			child := exec.Command(os.Args[0])
			child.Env = append(os.Environ(), "GENESIS_EXTENSION_TEST_MODE=descendant-child")
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			if err := child.Start(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(4)
			}
			if err := os.WriteFile(os.Getenv("GENESIS_EXTENSION_TEST_PIDFILE"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
				os.Exit(5)
			}
			time.Sleep(time.Minute)
		case "overflow":
			fmt.Print(strings.Repeat("x", maxResponse+1))
			os.Exit(0)
		}
		switch request.Method {
		case "describe":
			reply.DependsOn = []string{}
			reply.Methods = []string{"parseOptions", "prepare", "registerTasks", "preApply", "postApply", "detect", "apply", "validate"}
		case "parseOptions":
			reply.Options = request.Instance.Options
		case "prepare":
			reply.Env = map[string]string{"EXTENSION_READY": "yes"}
			if os.Getenv("GENESIS_EXTENSION_TEST_MODE") == "invalid-env" {
				reply.Env["BAD=KEY"] = "invalid"
			}
		case "registerTasks":
			reply.Tasks = []Command{{ID: "extension:task", Command: os.Args[0], Args: []string{"--version"}}}
		case "detect":
			reply.Detect = &core.DetectResult{OK: true, Details: "external present"}
		case "apply":
			reply.Apply = &core.ApplyResult{OK: true, DidChange: true}
		case "validate":
			reply.Validate = &core.ValidateResult{OK: true, Message: "external verified"}
		}
		if request.Method == "describe" {
			switch os.Getenv("GENESIS_EXTENSION_TEST_MODE") {
			case "apply-only":
				reply.Methods = []string{"apply"}
			case "unknown-method":
				reply.Methods = []string{"typo"}
			case "no-methods":
				reply.Methods = nil
			}
		}
		json.NewEncoder(os.Stdout).Encode(reply)
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func testInstance(t *testing.T) core.Instance {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return core.Instance{ID: "custom", Category: "tool", Module: "exec:" + executable, Options: map[string]any{"version": "1"}}
}
func TestExecutableLifecycle(t *testing.T) {
	t.Setenv("GENESIS_EXTENSION_TEST_HELPER", "1")
	instance := testInstance(t)
	plugin, err := Load(context.Background(), t.TempDir(), instance)
	if err != nil {
		t.Fatal(err)
	}
	runtime := &core.Runtime{Context: context.Background(), Cwd: t.TempDir()}
	runtime.Ensure()
	options, err := plugin.ParseOptions(instance.Options)
	if err != nil || options["version"] != "1" {
		t.Fatalf("%v %v", options, err)
	}
	if err = plugin.Prepare(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if runtime.Env.Get("EXTENSION_READY") != "yes" {
		t.Fatal("missing env change")
	}
	if err = plugin.RegisterTasks(runtime, instance); err != nil {
		t.Fatal(err)
	}
	detected, err := plugin.Detect(runtime, instance)
	if err != nil || !detected.OK {
		t.Fatalf("%v %v", detected, err)
	}
	applied, err := plugin.Apply(runtime, instance)
	if err != nil || !applied.OK || !applied.DidChange {
		t.Fatalf("%v %v", applied, err)
	}
	validated, err := plugin.Validate(runtime, instance)
	if err != nil || !validated.OK {
		t.Fatalf("%v %v", validated, err)
	}
	if err = plugin.PreApply(runtime, instance); err != nil {
		t.Fatal(err)
	}
	if err = plugin.PostApply(runtime, instance); err != nil {
		t.Fatal(err)
	}
}
func TestExecutableFailures(t *testing.T) {
	t.Setenv("GENESIS_EXTENSION_TEST_HELPER", "1")
	for _, mode := range []string{"bad-version", "error", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("GENESIS_EXTENSION_TEST_MODE", mode)
			if _, err := Load(context.Background(), t.TempDir(), testInstance(t)); err == nil {
				t.Fatal("accepted malformed plugin")
			}
		})
	}
	t.Setenv("GENESIS_EXTENSION_TEST_MODE", "hang")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := Load(ctx, t.TempDir(), testInstance(t)); err == nil {
		t.Fatal("expected cancellation")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancellation slow")
	}
}
func TestUnsupportedModules(t *testing.T) {
	for _, module := range []string{"unknown", "exec:", "exec:/missing/plugin"} {
		i := testInstance(t)
		i.Module = module
		if _, err := Load(context.Background(), t.TempDir(), i); err == nil {
			t.Fatal(module)
		}
	}
}
func TestGoConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes Go compiler")
	}
	cwd := t.TempDir()
	path := filepath.Join(cwd, "genesis.config.go")
	source := "//go:build ignore\n\npackage main\nimport \"fmt\"\nfunc main(){fmt.Println(`{\"scripts\":[{\"name\":\"ok\",\"command\":\"echo ok\"}]}`)}\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	config, err := LoadGoConfig(cwd, path)
	if err != nil || len(config.Scripts) != 1 {
		t.Fatalf("%+v %v", config, err)
	}
}
func TestDecodeOne(t *testing.T) {
	for _, input := range []string{"{}{}", "[] oops", "null extra"} {
		var value any
		if decodeOne([]byte(input), &value) == nil {
			t.Fatal(input)
		}
	}
}

func TestOptionalExecutableHooks(t *testing.T) {
	t.Setenv("GENESIS_EXTENSION_TEST_HELPER", "1")
	for _, mode := range []string{"apply-only", "no-methods"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("GENESIS_EXTENSION_TEST_MODE", mode)
			instance := testInstance(t)
			plugin, err := Load(context.Background(), t.TempDir(), instance)
			if err != nil {
				t.Fatal(err)
			}
			if plugin.Detect != nil || plugin.Validate != nil || plugin.Prepare != nil || plugin.RegisterTasks != nil || plugin.PreApply != nil || plugin.PostApply != nil || plugin.ParseOptions != nil {
				t.Fatal("unadvertised hook registered")
			}
			if (plugin.Apply != nil) != (mode == "apply-only") {
				t.Fatal("apply capability lost")
			}
			r := &core.Runtime{Context: context.Background(), Cwd: t.TempDir()}
			nodes := []core.Node{{Instance: instance, Plugin: plugin}}
			detected, err := core.RunDetect(nodes, r)
			if err != nil || detected[0].Status != "unknown" {
				t.Fatal(detected, err)
			}
			validated, err := core.RunValidate(nodes, r)
			if err != nil || !validated[0].OK {
				t.Fatal(validated, err)
			}
			applied, err := core.RunApply(nodes, r)
			if err != nil || !applied[0].OK || applied[0].DidChange != (mode == "apply-only") {
				t.Fatal(applied, err)
			}
		})
	}
	t.Setenv("GENESIS_EXTENSION_TEST_MODE", "unknown-method")
	if _, err := Load(context.Background(), t.TempDir(), testInstance(t)); err == nil {
		t.Fatal("unknown capability accepted")
	}
}
func TestEnvironmentResponseValidationIsAtomic(t *testing.T) {
	t.Setenv("GENESIS_EXTENSION_TEST_HELPER", "1")
	t.Setenv("GENESIS_EXTENSION_TEST_MODE", "invalid-env")
	instance := testInstance(t)
	plugin, err := Load(context.Background(), t.TempDir(), instance)
	if err != nil {
		t.Fatal(err)
	}
	r := &core.Runtime{Context: context.Background(), Cwd: t.TempDir(), Env: core.NewEnvironment(map[string]string{})}
	r.Ensure()
	if err = plugin.Prepare(r, instance); err == nil {
		t.Fatal("malformed environment accepted")
	}
	if r.Env.Get("EXTENSION_READY") != "" {
		t.Fatal("partial environment mutation on malformed response")
	}
}
func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("helper did not start: %s", path)
}
func processRunning(pid int) bool {
	if runtime.GOOS == "windows" {
		data, err := exec.Command("tasklist.exe", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").Output()
		return err == nil && strings.Contains(string(data), `","`+strconv.Itoa(pid)+`","`)
	}
	data, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	state := strings.TrimSpace(string(data))
	return err == nil && state != "" && !strings.HasPrefix(state, "Z")
}
func TestCancellationStopsExecutableDescendants(t *testing.T) {
	t.Setenv("GENESIS_EXTENSION_TEST_HELPER", "1")
	t.Setenv("GENESIS_EXTENSION_TEST_MODE", "descendant")
	pidfile := filepath.Join(t.TempDir(), "child.pid")
	t.Setenv("GENESIS_EXTENSION_TEST_PIDFILE", pidfile)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	cwd, instance := t.TempDir(), testInstance(t)
	go func() { _, err := Load(ctx, cwd, instance); result <- err }()
	waitForFile(t, pidfile)
	data, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if processRunning(pid) {
			if process, e := os.FindProcess(pid); e == nil {
				_ = process.Kill()
			}
		}
	}()
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("plugin cancellation hung")
	}
	deadline := time.Now().Add(2 * time.Second)
	for processRunning(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if processRunning(pid) {
		t.Fatal("plugin descendant survived cancellation")
	}
}
func TestGoConfigHonorsCallerCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("invokes Go compiler")
	}
	cwd := t.TempDir()
	marker := filepath.Join(cwd, "started")
	path := filepath.Join(cwd, "genesis.config.go")
	source := fmt.Sprintf("//go:build ignore\n\npackage main\nimport(\"os\";\"time\")\nfunc main(){os.WriteFile(%q,[]byte(\"ready\"),0600);time.Sleep(time.Minute)}\n", marker)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := LoadGoConfigContext(ctx, cwd, path); result <- err }()
	waitForFile(t, marker)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("Go configuration ignored caller cancellation")
	}
}
