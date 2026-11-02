package core

import (
	"context"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Options = map[string]any
type Instance struct {
	ID       string  `json:"id" yaml:"id"`
	Category string  `json:"category" yaml:"category"`
	Module   string  `json:"module" yaml:"module"`
	Options  Options `json:"options,omitempty" yaml:"options,omitempty"`
}
type RepoSpec struct {
	URL    string `json:"url" yaml:"url"`
	Folder string `json:"folder" yaml:"folder"`
	Branch string `json:"branch,omitempty" yaml:"branch,omitempty"`
}
type ScriptSpec struct {
	Name        string `json:"name" yaml:"name"`
	Command     string `json:"command" yaml:"command"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	When        string `json:"when,omitempty" yaml:"when,omitempty"`
}
type Config struct {
	Tools        []Instance        `json:"tools,omitempty" yaml:"tools,omitempty"`
	SDKs         []Instance        `json:"sdks,omitempty" yaml:"sdks,omitempty"`
	Languages    []Instance        `json:"languages,omitempty" yaml:"languages,omitempty"`
	Repositories []RepoSpec        `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Scripts      []ScriptSpec      `json:"scripts,omitempty" yaml:"scripts,omitempty"`
	Env          map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}
type DetectResult struct {
	OK      bool   `json:"ok"`
	Details string `json:"details,omitempty"`
}
type ApplyResult struct {
	OK        bool   `json:"ok"`
	DidChange bool   `json:"didChange"`
	Details   string `json:"details,omitempty"`
}
type ValidateResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}
type Hook func(*Runtime, Instance) error
type Plugin struct {
	ID, Category                                string
	DependsOn                                   []string
	ParseOptions                                func(Options) (Options, error)
	Prepare, RegisterTasks, PreApply, PostApply Hook
	Detect                                      func(*Runtime, Instance) (DetectResult, error)
	Apply                                       func(*Runtime, Instance) (ApplyResult, error)
	Validate                                    func(*Runtime, Instance) (ValidateResult, error)
}
type Node struct {
	Instance Instance
	Plugin   Plugin
}
type Runner func(context.Context, string, []string, CommandOptions) (CommandResult, error)
type CommandOptions struct {
	Cwd         string
	Env         map[string]string
	Timeout     time.Duration
	Stdin       io.Reader
	OutputLimit int
}
type CommandResult struct {
	Code                             int
	Stdout, Stderr                   string
	StdoutTruncated, StderrTruncated bool
}
type Runtime struct {
	Context        context.Context
	Cwd            string
	Env            *Environment
	Logger         *log.Logger
	Tasks          *TaskRegistry
	Runner         Runner
	Platform, Arch string
}

func (r *Runtime) Ensure() {
	if r.Context == nil {
		r.Context = context.Background()
	}
	if r.Cwd == "" {
		r.Cwd, _ = os.Getwd()
	}
	if r.Env == nil {
		r.Env = NewEnvironment(nil)
	}
	if r.Logger == nil {
		r.Logger = log.New(io.Discard, "", 0)
	}
	if r.Tasks == nil {
		r.Tasks = NewTaskRegistry()
	}
	if r.Runner == nil {
		r.Runner = RunCommand
	}
	if r.Platform == "" {
		r.Platform = GetPlatform()
	}
	if r.Arch == "" {
		r.Arch = runtime.GOARCH
	}
}
func (r *Runtime) Run(command string, args ...string) (CommandResult, error) {
	return r.RunWithOptions(command, args, CommandOptions{})
}
func (r *Runtime) RunWithOptions(command string, args []string, options CommandOptions) (CommandResult, error) {
	if options.Cwd == "" {
		options.Cwd = r.Cwd
	}
	if options.Env == nil && r.Env != nil {
		options.Env = r.Env.Snapshot()
	}
	runner := r.Runner
	if runner == nil {
		runner = RunCommand
	}
	ctx := r.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return runner(ctx, command, args, options)
}
func GetPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

type Environment struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewEnvironment(values map[string]string) *Environment {
	e := &Environment{values: map[string]string{}}
	if values == nil {
		for _, s := range os.Environ() {
			k, v, ok := strings.Cut(s, "=")
			if ok {
				e.values[environmentKey(k)] = v
			}
		}
	} else {
		for k, v := range values {
			e.values[environmentKey(k)] = v
		}
	}
	return e
}
func (e *Environment) Get(key string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.values[environmentKey(key)]
}
func (e *Environment) Set(key, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.values[environmentKey(key)] = value
}
func (e *Environment) Snapshot() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	m := make(map[string]string, len(e.values))
	for k, v := range e.values {
		m[k] = v
	}
	return m
}
func (e *Environment) PrependPath(path string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := "PATH"
	if runtime.GOOS == "windows" {
		for k := range e.values {
			if strings.EqualFold(k, "PATH") {
				key = k
				break
			}
		}
	}
	parts := strings.Split(e.values[key], string(os.PathListSeparator))
	result := []string{path}
	for _, v := range parts {
		if v != "" && v != path {
			result = append(result, v)
		}
	}
	e.values[key] = strings.Join(result, string(os.PathListSeparator))
}

func environmentKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}
