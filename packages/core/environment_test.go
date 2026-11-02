package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestEnvironmentPlan(t *testing.T) {
	config := Config{Env: map[string]string{"SECRET": "value"}, Scripts: []ScriptSpec{{Name: "after", Command: "after"}, {Name: "before", Command: "before", When: "before"}}, Repositories: []RepoSpec{{URL: "url", Folder: "repo", Branch: "main"}}}
	plan, err := CreateEnvironmentPlan(config, []Node{testNode("node")})
	if err != nil {
		t.Fatal(err)
	}
	types := []string{}
	for _, action := range plan.Actions {
		types = append(types, action.Type)
	}
	if !reflect.DeepEqual(types, []string{"script", "plugin", "repository", "script"}) || plan.Environment[0] != "SECRET" {
		t.Fatal(plan)
	}
}
func TestEnvironmentScriptsAndFailures(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		t.Run(map[bool]string{false: "sequential", true: "parallel"}[parallel], func(t *testing.T) {
			dir := t.TempDir()
			events := []string{}
			r := &Runtime{Cwd: dir, Env: NewEnvironment(map[string]string{"PATH": os.Getenv("PATH")}), Runner: func(_ context.Context, _ string, args []string, opts CommandOptions) (CommandResult, error) {
				events = append(events, args[len(args)-1])
				if opts.Env["TOKEN"] != "secret" || opts.Cwd != dir {
					t.Fatal("runtime context lost")
				}
				return CommandResult{}, nil
			}}
			node := testNode("plugin")
			node.Plugin.Apply = func(r *Runtime, _ Instance) (ApplyResult, error) {
				events = append(events, "plugin")
				if r.Env.Get("TOKEN") != "secret" {
					t.Error("config env missing")
				}
				return ApplyResult{OK: true}, nil
			}
			config := Config{Env: map[string]string{"TOKEN": "secret"}, Scripts: []ScriptSpec{{Name: "before", Command: "before", When: "before"}, {Name: "after", Command: "after"}}}
			var err error
			if parallel {
				_, err = ApplyEnvironmentParallel(config, []Node{node}, r, 2)
			} else {
				_, err = ApplyEnvironment(config, []Node{node}, r)
			}
			if err != nil || !reflect.DeepEqual(events, []string{"before", "plugin", "after"}) {
				t.Fatal(events, err)
			}
			if r.Env.Get("TOKEN") != "" {
				t.Fatal("config env leaked into caller")
			}
		})
	}
	dir := t.TempDir()
	r := &Runtime{Cwd: dir, Runner: func(context.Context, string, []string, CommandOptions) (CommandResult, error) {
		return CommandResult{Code: 42, Stderr: "failure"}, nil
	}}
	if _, err := ApplyEnvironment(Config{Scripts: []ScriptSpec{{Name: "broken", Command: "fail", When: "before"}}}, nil, r); err == nil {
		t.Fatal("failed script ignored")
	}
	node := testNode("broken")
	node.Plugin.Apply = func(*Runtime, Instance) (ApplyResult, error) { return ApplyResult{Details: "bad"}, nil }
	r.Runner = func(context.Context, string, []string, CommandOptions) (CommandResult, error) {
		t.Fatal("after script ran after failed plugin")
		return CommandResult{}, nil
	}
	results, err := ApplyEnvironment(Config{Scripts: []ScriptSpec{{Name: "after", Command: "echo"}}}, []Node{node}, r)
	if err != nil || results[0].OK {
		t.Fatal(results, err)
	}
}
func TestRepositoryRealClonePreservesWork(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0755); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+filepath.Join(root, "empty-config"), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, data, err)
		}
		return strings.TrimSpace(string(data))
	}
	git(source, "init", "-b", "main")
	writeTestFile(t, filepath.Join(source, "file"), "initial")
	git(source, "add", "file")
	git(source, "commit", "-m", "initial")
	r := &Runtime{Cwd: root}
	r.Ensure()
	repo := RepoSpec{URL: source, Folder: "nested/clone", Branch: "main"}
	if err := EnsureRepository(repo, r); err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(root, repo.Folder)
	writeTestFile(t, filepath.Join(clone, "file"), "uncommitted")
	if err := EnsureRepository(repo, r); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(clone, "file"))
	if string(data) != "uncommitted" {
		t.Fatal("worktree overwritten")
	}
	for _, bad := range []RepoSpec{{URL: "different", Folder: repo.Folder}, {URL: source, Folder: repo.Folder, Branch: "other"}, {URL: source, Folder: "source/file"}, {URL: source, Folder: "nested"}} {
		if err := EnsureRepository(bad, r); err == nil {
			t.Fatalf("unsafe destination accepted %#v", bad)
		}
	}
	if runtime.GOOS != "windows" {
		alias := filepath.Join(root, "alias")
		if err := os.Symlink(clone, alias); err != nil {
			t.Fatal(err)
		}
		if err := EnsureRepository(RepoSpec{URL: source, Folder: alias, Branch: "main"}, r); err != nil {
			t.Fatal(err)
		}
	}
}
func TestRepositoryCloneError(t *testing.T) {
	r := &Runtime{Cwd: t.TempDir(), Runner: func(context.Context, string, []string, CommandOptions) (CommandResult, error) {
		return CommandResult{Code: 1, Stderr: "bad remote"}, nil
	}}
	r.Ensure()
	if err := EnsureRepository(RepoSpec{URL: "missing", Folder: "clone"}, r); err == nil || !strings.Contains(err.Error(), "bad remote") {
		t.Fatal(err)
	}
}
func TestGenesisHomeRejectsMissingAndRelativeHome(t *testing.T) {
	key := "HOME"
	if runtime.GOOS == "windows" {
		key = "USERPROFILE"
	}
	for _, value := range []string{"", "relative/home"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(key, value)
			if home, err := ResolveGenesisHome(); err == nil || home != "" {
				t.Fatal(home, err)
			}
			if home := GetGenesisHome(); home != "" {
				t.Fatal("unsafe home fallback", home)
			}
			if home, err := EnsureGenesisHome(); err == nil || home != "" {
				t.Fatal(home, err)
			}
			manager, _ := NewEnvironmentCacheManager(CacheConfig{})
			entry, err := manager.CacheEnvironment("test", Config{}, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.CreateSnapshot(entry.ID, "snapshot", ""); err == nil {
				t.Fatal("snapshot accepted unresolved home")
			}
		})
	}
	home := t.TempDir()
	t.Setenv(key, home)
	resolved, err := ResolveGenesisHome()
	if err != nil || resolved != filepath.Join(home, ".genesis") {
		t.Fatal(resolved, err)
	}
}
