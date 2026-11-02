package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type EnvironmentAction struct {
	Type      string   `json:"type"`
	Name      string   `json:"name,omitempty"`
	Command   string   `json:"command,omitempty"`
	When      string   `json:"when,omitempty"`
	ID        string   `json:"id,omitempty"`
	Module    string   `json:"module,omitempty"`
	DependsOn []string `json:"dependsOn,omitempty"`
	URL       string   `json:"url,omitempty"`
	Folder    string   `json:"folder,omitempty"`
	Branch    string   `json:"branch,omitempty"`
}
type Plan struct {
	Environment []string            `json:"environment"`
	Actions     []EnvironmentAction `json:"actions"`
}

func CreateEnvironmentPlan(config Config, nodes []Node) (Plan, error) {
	ordered, err := BuildPluginGraph(nodes)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Environment: []string{}, Actions: []EnvironmentAction{}}
	for k := range config.Env {
		p.Environment = append(p.Environment, k)
	}
	sort.Strings(p.Environment)
	script := func(s ScriptSpec) EnvironmentAction {
		when := s.When
		if when == "" {
			when = "after"
		}
		return EnvironmentAction{Type: "script", Name: s.Name, Command: s.Command, When: when}
	}
	for _, s := range config.Scripts {
		if s.When == "before" {
			p.Actions = append(p.Actions, script(s))
		}
	}
	for _, n := range ordered {
		p.Actions = append(p.Actions, EnvironmentAction{Type: "plugin", ID: n.Instance.ID, Module: n.Instance.Module, DependsOn: append([]string{}, n.Plugin.DependsOn...)})
	}
	for _, repo := range config.Repositories {
		p.Actions = append(p.Actions, EnvironmentAction{Type: "repository", URL: repo.URL, Folder: repo.Folder, Branch: repo.Branch})
	}
	for _, s := range config.Scripts {
		if s.When != "before" {
			p.Actions = append(p.Actions, script(s))
		}
	}
	return p, nil
}
func runScript(s ScriptSpec, r *Runtime) error {
	r.Logger.Printf("Running script: %s", s.Name)
	command := "sh"
	args := []string{"-c", s.Command}
	if r.Platform == "windows" {
		command = "cmd.exe"
		args = []string{"/d", "/s", "/c", s.Command}
	}
	result, err := r.Run(command, args...)
	if result.Stdout != "" {
		r.Logger.Print(result.Stdout)
	}
	if err != nil {
		return fmt.Errorf("script '%s': %w", s.Name, err)
	}
	if result.Code != 0 {
		return fmt.Errorf("script '%s' failed (exit %d): %s", s.Name, result.Code, result.Stderr)
	}
	return nil
}
func EnsureRepository(repo RepoSpec, r *Runtime) error {
	folder := repo.Folder
	if !filepath.IsAbs(folder) {
		folder = filepath.Join(r.Cwd, folder)
	}
	if _, err := os.Lstat(folder); err == nil {
		root, err := r.Run("git", "-C", folder, "rev-parse", "--show-toplevel")
		if err != nil {
			return err
		}
		actual, e1 := filepath.EvalSymlinks(folder)
		expected, e2 := filepath.EvalSymlinks(strings.TrimSpace(root.Stdout))
		if root.Code != 0 || e1 != nil || e2 != nil || actual != expected {
			return fmt.Errorf("repository destination '%s' exists but is not a repository root", repo.Folder)
		}
		remote, err := r.Run("git", "-C", folder, "remote", "get-url", "origin")
		if err != nil {
			return err
		}
		if remote.Code != 0 || strings.TrimSpace(remote.Stdout) != repo.URL {
			return fmt.Errorf("repository '%s' has a different origin; refusing to overwrite it", repo.Folder)
		}
		if repo.Branch != "" {
			branch, err := r.Run("git", "-C", folder, "symbolic-ref", "--short", "HEAD")
			if err != nil {
				return err
			}
			if branch.Code != 0 || strings.TrimSpace(branch.Stdout) != repo.Branch {
				return fmt.Errorf("repository '%s' is not on requested branch '%s'", repo.Folder, repo.Branch)
			}
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(folder), 0755); err != nil {
		return err
	}
	args := []string{"clone"}
	if repo.Branch != "" {
		args = append(args, "--branch", repo.Branch)
	}
	args = append(args, "--", repo.URL, folder)
	result, err := r.Run("git", args...)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("failed to clone '%s': %s", repo.URL, result.Stderr)
	}
	return nil
}
func ApplyEnvironment(config Config, nodes []Node, r *Runtime) ([]ApplySummary, error) {
	return applyEnvironment(config, nodes, r, 1)
}
func ApplyEnvironmentParallel(config Config, nodes []Node, r *Runtime, workers int) ([]ApplySummary, error) {
	if workers < 1 {
		return nil, fmt.Errorf("workers must be a positive integer")
	}
	return applyEnvironment(config, nodes, r, workers)
}
func applyEnvironment(config Config, nodes []Node, r *Runtime, workers int) ([]ApplySummary, error) {
	r.Ensure()
	if _, err := BuildPluginGraph(nodes); err != nil {
		return nil, err
	}
	copy := *r
	copy.Env = NewEnvironment(r.Env.Snapshot())
	for k, v := range config.Env {
		copy.Env.Set(k, v)
	}
	for _, script := range config.Scripts {
		if script.When == "before" {
			if err := runScript(script, &copy); err != nil {
				return nil, err
			}
		}
	}
	var results []ApplySummary
	var err error
	if workers > 1 {
		engine, createErr := NewParallelExecutionEngine(workers)
		if createErr != nil {
			return nil, createErr
		}
		plan, planErr := engine.CreateExecutionPlan(nodes)
		if planErr != nil {
			return nil, planErr
		}
		parallel, runErr := engine.ExecuteParallel(plan, &copy)
		err = runErr
		categories := map[string]string{}
		for _, n := range nodes {
			categories[n.Instance.ID] = n.Instance.Category
		}
		for _, result := range parallel {
			results = append(results, ApplySummary{ID: result.PluginID, Category: categories[result.PluginID], OK: result.Success, DidChange: result.DidChange, Details: result.Error})
		}
	} else {
		results, err = RunApply(nodes, &copy)
	}
	if err != nil {
		return results, err
	}
	for _, result := range results {
		if !result.OK {
			return results, nil
		}
	}
	for _, repo := range config.Repositories {
		if err := EnsureRepository(repo, &copy); err != nil {
			return results, err
		}
	}
	for _, script := range config.Scripts {
		if script.When != "before" {
			if err := runScript(script, &copy); err != nil {
				return results, err
			}
		}
	}
	return results, nil
}
func GetGenesisHome() string { home, _ := os.UserHomeDir(); return filepath.Join(home, ".genesis") }
func EnsureGenesisHome() (string, error) {
	home := GetGenesisHome()
	return home, os.MkdirAll(home, 0700)
}
func EnsureDir(path string) error        { return os.MkdirAll(path, 0755) }
func JoinPath(segments ...string) string { return filepath.Join(segments...) }
