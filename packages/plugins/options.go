package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

var numericVersion = regexp.MustCompile(`^\d+(?:\.\d+){0,2}$`)
var fullVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var packagePattern = regexp.MustCompile(`^[^-\s][^\s]*$`)

func ParseOptions(name string, raw map[string]any) (map[string]any, error) {
	options := map[string]any{}
	allowed := map[string]string{"version": "version"}
	defaults := map[string]any{}
	switch name {
	case "bun":
		allowed["install_dir"] = "directory"
		allowed["libc"] = "libc"
		defaults["libc"] = "auto"
	case "deno", "go":
		allowed["install_dir"] = "directory"
	case "pnpm", "yarn":
		allowed["install_dir"] = "directory"
		allowed["node_plugin"] = "dependency"
		defaults["node_plugin"] = "node"
	case "node":
		allowed["install_dir"] = "directory"
		allowed["use_nvm"] = "bool"
		allowed["global_packages"] = "packages"
		defaults["use_nvm"] = true
	case "java":
		allowed["install_dir"] = "directory"
		allowed["distribution"] = "distribution"
		defaults["distribution"] = "openjdk"
	case "python":
	case "git":
		allowed["install_dir"] = "directory"
		allowed["install_method"] = "method"
		defaults["version"] = "latest"
		defaults["install_method"] = "package"
	case "docker":
		allowed["include_compose"] = "bool"
		allowed["install_desktop"] = "bool"
		defaults["version"] = "latest"
		defaults["include_compose"] = true
		defaults["install_desktop"] = false
	case "homebrew":
		delete(allowed, "version")
		allowed["update_packages"] = "bool"
		allowed["install_cask"] = "bool"
		allowed["add_to_path"] = "bool"
		allowed["global_packages"] = "packages"
		defaults["update_packages"] = true
		defaults["install_cask"] = true
		defaults["add_to_path"] = true
	default:
		return nil, fmt.Errorf("unknown plugin %q", name)
	}
	for k, v := range defaults {
		options[k] = v
	}
	for key, value := range raw {
		kind, ok := allowed[key]
		if !ok {
			return nil, fmt.Errorf("%s.%s: unknown option", name, key)
		}
		valid := false
		switch kind {
		case "bool":
			_, valid = value.(bool)
		case "directory":
			s, ok := value.(string)
			valid = ok && filepath.IsAbs(s) && filepath.Clean(s) != filepath.VolumeName(s)+string(filepath.Separator)
		case "version":
			s, ok := value.(string)
			valid = ok && numericVersion.MatchString(s)
			if s == "latest" && (name == "git" || name == "docker") {
				valid = true
			}
			if name == "go" || name == "bun" || name == "deno" || name == "pnpm" || name == "yarn" {
				valid = ok && fullVersion.MatchString(s)
			}
		case "dependency":
			s, ok := value.(string)
			valid = value == nil || (ok && strings.TrimSpace(s) != "")
		case "packages":
			list, ok := stringSlice(value)
			valid = ok
			for _, s := range list {
				valid = valid && packagePattern.MatchString(s)
			}
			if valid {
				value = list
			}
		case "libc":
			s, ok := value.(string)
			valid = ok && (s == "auto" || s == "glibc" || s == "musl")
		case "distribution":
			s, ok := value.(string)
			valid = ok && (s == "openjdk" || s == "oracle")
		case "method":
			s, ok := value.(string)
			valid = ok && (s == "package" || s == "source" || s == "binary")
		}
		if !valid {
			return nil, fmt.Errorf("%s.%s: invalid %s value", name, key, kind)
		}
		options[key] = value
	}
	if _, requires := allowed["version"]; requires && options["version"] == nil {
		return nil, fmt.Errorf("%s.version: required", name)
	}
	if name == "git" && options["install_dir"] != nil && options["install_method"] == "package" {
		return nil, fmt.Errorf("git.install_dir requires source or binary installation")
	}
	return options, nil
}
func MatchesVersion(installed, requested string) bool {
	for {
		actual, restActual, actualMore := strings.Cut(installed, ".")
		desired, restDesired, desiredMore := strings.Cut(requested, ".")
		if actual != desired {
			return false
		}
		if !desiredMore {
			return true
		}
		if !actualMore {
			return false
		}
		installed, requested = restActual, restDesired
	}
}

func optionString(options map[string]any, key string) string { s, _ := options[key].(string); return s }
func optionBool(options map[string]any, key string) bool     { v, _ := options[key].(bool); return v }
func stringSlice(value any) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		return v, true
	case []any:
		r := make([]string, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			r[i] = s
		}
		return r, true
	}
	return nil, false
}
func exists(path string) bool { _, err := os.Stat(path); return err == nil }
func runtimeHome(r *core.Runtime) string {
	key := "HOME"
	if r.Platform == "windows" {
		key = "USERPROFILE"
	}
	if v := r.Env.Get(key); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return home
}
func executableName(r *core.Runtime, name string) string {
	if r.Platform == "windows" {
		return name + ".exe"
	}
	return name
}
func managedDirectory(r *core.Runtime, i core.Instance, name string) string {
	if s := optionString(i.Options, "install_dir"); s != "" {
		return s
	}
	return filepath.Join(runtimeHome(r), ".genesis", name)
}
