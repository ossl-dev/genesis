// Package extension supports portable executable plugins and Go configurations.
package extension

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ossl-dev/genesis/packages/core"
)

const ProtocolVersion = 1
const maxResponse = 8 << 20

type Request struct {
	Protocol int               `json:"protocol"`
	Method   string            `json:"method"`
	Instance core.Instance     `json:"instance"`
	Cwd      string            `json:"cwd"`
	Env      map[string]string `json:"env,omitempty"`
}
type Command struct {
	ID             string   `json:"id"`
	Description    string   `json:"description"`
	Priority       int      `json:"priority"`
	DependsOn      []string `json:"dependsOn"`
	Command        string   `json:"command"`
	Args           []string `json:"args"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
}
type Response struct {
	Methods   []string             `json:"methods,omitempty"`
	Protocol  int                  `json:"protocol"`
	Error     string               `json:"error,omitempty"`
	DependsOn []string             `json:"dependsOn,omitempty"`
	Options   map[string]any       `json:"options,omitempty"`
	Env       map[string]string    `json:"env,omitempty"`
	Tasks     []Command            `json:"tasks,omitempty"`
	Detect    *core.DetectResult   `json:"detect,omitempty"`
	Apply     *core.ApplyResult    `json:"apply,omitempty"`
	Validate  *core.ValidateResult `json:"validate,omitempty"`
}

func execute(ctx context.Context, command string, args []string, cwd string, env map[string]string, input any) ([]byte, error) {
	options := core.CommandOptions{Cwd: cwd, Env: env, OutputLimit: maxResponse}
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		options.Stdin = bytes.NewReader(append(data, '\n'))
	}
	result, err := core.RunCommand(ctx, command, args, options)
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("%s exited %d: %s", command, result.Code, result.Stderr)
	}
	if result.StdoutTruncated {
		return nil, errors.New("extension output exceeds 8 MiB")
	}
	return []byte(result.Stdout), nil
}

// LoadGoConfig runs an explicitly selected Go program whose stdout is JSON.
func LoadGoConfig(cwd, path string) (core.Config, error) {
	return LoadGoConfigContext(context.Background(), cwd, path)
}

func LoadGoConfigContext(parent context.Context, cwd, path string) (core.Config, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	data, err := execute(ctx, "go", []string{"run", path}, cwd, nil, nil)
	if err != nil {
		return core.Config{}, err
	}
	var value any
	if err = decodeOne(data, &value); err != nil {
		return core.Config{}, fmt.Errorf("Go config stdout must contain one JSON configuration: %w", err)
	}
	return core.NormalizeConfig(value)
}

func decodeOne(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

// Load resolves exec:<path>. The config explicitly authorizes executing that path.
func Load(ctx context.Context, cwd string, instance core.Instance) (core.Plugin, error) {
	if !strings.HasPrefix(instance.Module, "exec:") {
		return core.Plugin{}, fmt.Errorf("plugin %q: unsupported module %q; compile Go plugins into registry or use exec:<path>", instance.ID, instance.Module)
	}
	command := strings.TrimPrefix(instance.Module, "exec:")
	if command == "" {
		return core.Plugin{}, errors.New("executable plugin path is empty")
	}
	if !filepath.IsAbs(command) {
		command = filepath.Join(cwd, command)
	}
	info, err := os.Stat(command)
	if err != nil {
		return core.Plugin{}, err
	}
	if info.IsDir() {
		return core.Plugin{}, errors.New("plugin executable is a directory")
	}
	invoke := func(runtime *core.Runtime, method string, i core.Instance) (Response, error) {
		runtime.Ensure()
		callCtx, cancel := context.WithTimeout(runtime.Context, 30*time.Minute)
		defer cancel()
		data, err := execute(callCtx, command, nil, runtime.Cwd, runtime.Env.Snapshot(), Request{Protocol: ProtocolVersion, Method: method, Instance: i, Cwd: runtime.Cwd, Env: runtime.Env.Snapshot()})
		if err != nil {
			return Response{}, err
		}
		var response Response
		if err = decodeOne(data, &response); err != nil {
			return response, fmt.Errorf("plugin %s %s: %w", i.ID, method, err)
		}
		if response.Protocol != ProtocolVersion {
			return response, fmt.Errorf("plugin %s: unsupported protocol %d", i.ID, response.Protocol)
		}
		if response.Error != "" {
			return response, errors.New(response.Error)
		}
		for key, value := range response.Env {
			if key == "" || strings.ContainsAny(key, "=\x00") || strings.ContainsRune(value, 0) {
				return response, fmt.Errorf("plugin %s: invalid environment entry", i.ID)
			}
		}
		for key, value := range response.Env {
			runtime.Env.Set(key, value)
		}
		return response, nil
	}
	runtime := &core.Runtime{Context: ctx, Cwd: cwd}
	runtime.Ensure()
	metadata, err := invoke(runtime, "describe", instance)
	if err != nil {
		return core.Plugin{}, err
	}
	plugin := core.Plugin{ID: instance.ID, Category: instance.Category, DependsOn: metadata.DependsOn}
	plugin.ParseOptions = func(options map[string]any) (map[string]any, error) {
		i := instance
		i.Options = options
		reply, err := invoke(runtime, "parseOptions", i)
		if err != nil {
			return nil, err
		}
		if reply.Options == nil {
			return options, nil
		}
		return reply.Options, nil
	}
	hook := func(method string) core.Hook {
		return func(r *core.Runtime, i core.Instance) error { _, err := invoke(r, method, i); return err }
	}
	plugin.Prepare = hook("prepare")
	plugin.PreApply = hook("preApply")
	plugin.PostApply = hook("postApply")
	plugin.RegisterTasks = func(r *core.Runtime, i core.Instance) error {
		reply, err := invoke(r, "registerTasks", i)
		if err != nil {
			return err
		}
		for _, spec := range reply.Tasks {
			if spec.Command == "" || spec.ID == "" || spec.TimeoutSeconds < 0 || spec.TimeoutSeconds > 86400 {
				return errors.New("plugin task requires ID, command and timeout within 0..86400")
			}
			spec := spec
			err = r.Tasks.Register(core.Task{ID: spec.ID, Description: spec.Description, Priority: spec.Priority, DependsOn: spec.DependsOn, Execute: func(ctx context.Context) (core.TaskResult, error) {
				options := core.CommandOptions{Cwd: r.Cwd, Env: r.Env.Snapshot(), Timeout: time.Duration(spec.TimeoutSeconds) * time.Second}
				runner := r.Runner
				if runner == nil {
					runner = core.RunCommand
				}
				result, err := runner(ctx, spec.Command, spec.Args, options)
				return core.TaskResult{OK: err == nil && result.Code == 0, Details: result.Stdout, Error: result.Stderr}, err
			}})
			if err != nil {
				return err
			}
		}
		return nil
	}
	plugin.Detect = func(r *core.Runtime, i core.Instance) (core.DetectResult, error) {
		reply, err := invoke(r, "detect", i)
		if err != nil {
			return core.DetectResult{}, err
		}
		if reply.Detect == nil {
			return core.DetectResult{}, errors.New("plugin detect response missing")
		}
		return *reply.Detect, nil
	}
	plugin.Apply = func(r *core.Runtime, i core.Instance) (core.ApplyResult, error) {
		reply, err := invoke(r, "apply", i)
		if reply.Apply != nil {
			return *reply.Apply, err
		}
		if err != nil {
			return core.ApplyResult{}, err
		}
		return core.ApplyResult{}, errors.New("plugin apply response missing")
	}

	plugin.Validate = func(r *core.Runtime, i core.Instance) (core.ValidateResult, error) {
		reply, err := invoke(r, "validate", i)
		if err != nil {
			return core.ValidateResult{}, err
		}
		if reply.Validate == nil {
			return core.ValidateResult{}, errors.New("plugin validate response missing")
		}
		return *reply.Validate, nil
	}
	methods := map[string]bool{}
	for _, method := range metadata.Methods {
		switch method {
		case "parseOptions", "prepare", "registerTasks", "preApply", "postApply", "detect", "apply", "validate":
			methods[method] = true
		default:
			return core.Plugin{}, fmt.Errorf("plugin %s: unknown method %q", instance.ID, method)
		}
	}
	if !methods["parseOptions"] {
		plugin.ParseOptions = nil
	}
	if !methods["prepare"] {
		plugin.Prepare = nil
	}
	if !methods["registerTasks"] {
		plugin.RegisterTasks = nil
	}
	if !methods["preApply"] {
		plugin.PreApply = nil
	}
	if !methods["postApply"] {
		plugin.PostApply = nil
	}
	if !methods["detect"] {
		plugin.Detect = nil
	}
	if !methods["apply"] {
		plugin.Apply = nil
	}
	if !methods["validate"] {
		plugin.Validate = nil
	}

	return plugin, nil
}
