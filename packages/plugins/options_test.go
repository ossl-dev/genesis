package plugins

import (
	"path/filepath"
	"testing"
)

func TestOptionsDefaults(t *testing.T) {
	for _, name := range []string{"node", "bun", "deno", "pnpm", "yarn", "go", "java", "python", "git", "docker", "homebrew"} {
		t.Run(name, func(t *testing.T) {
			raw := map[string]any{}
			if name != "git" && name != "docker" && name != "homebrew" {
				raw["version"] = "22.1.0"
			}
			options, err := ParseOptions(name, raw)
			if err != nil {
				t.Fatal(err)
			}
			if name == "node" && options["use_nvm"] != true {
				t.Fatal(options)
			}
			if name == "pnpm" && options["node_plugin"] != "node" {
				t.Fatal(options)
			}
			if name == "git" && options["install_method"] != "package" {
				t.Fatal(options)
			}
		})
	}
}
func TestOptionsRejectInvalid(t *testing.T) {
	cases := []struct {
		name    string
		options map[string]any
	}{{"node", nil}, {"node", map[string]any{"version": 22}}, {"node", map[string]any{"version": "22;echo hi"}}, {"node", map[string]any{"version": "22", "use_nvm": "false"}}, {"node", map[string]any{"version": "22", "typo": true}}, {"go", map[string]any{"version": "1.22"}}, {"bun", map[string]any{"version": "1.3"}}, {"deno", map[string]any{"version": "2.1.0", "install_dir": "relative"}}, {"yarn", map[string]any{"version": "4.1.0", "install_dir": filepath.VolumeName(filepath.Clean("/")) + string(filepath.Separator)}}, {"homebrew", map[string]any{"global_packages": []string{"--force"}}}, {"git", map[string]any{"install_dir": t.TempDir()}}}
	for _, c := range cases {
		if _, err := ParseOptions(c.name, c.options); err == nil {
			t.Errorf("accepted %s %#v", c.name, c.options)
		}
	}
}
func TestVersionComponents(t *testing.T) {
	for _, c := range []struct {
		actual, want string
		matches      bool
	}{{"20.11.0", "2", false}, {"3.11.9", "3.1", false}, {"20.11.0", "20", true}, {"3.11.9", "3.11.8", false}, {"1.2", "1.2.0", false}} {
		if MatchesVersion(c.actual, c.want) != c.matches {
			t.Errorf("%s vs %s", c.actual, c.want)
		}
	}
}
func TestPackagesAndNullDependency(t *testing.T) {
	options, err := ParseOptions("pnpm", map[string]any{"version": "10.19.0", "node_plugin": nil})
	if err != nil || options["node_plugin"] != nil {
		t.Fatal(options, err)
	}
	options, err = ParseOptions("node", map[string]any{"version": "22", "global_packages": []any{"@scope/tool@1.2.3", "typescript"}})
	if err != nil || len(options["global_packages"].([]string)) != 2 {
		t.Fatal(options, err)
	}
}
