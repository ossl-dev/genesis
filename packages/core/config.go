package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type PluginMetadata struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Module      string `json:"module"`
	Description string `json:"description"`
}

var BuiltinPlugins = []PluginMetadata{
	{"node", "tool", "node", "Node.js JavaScript runtime"}, {"bun", "tool", "bun", "Bun JavaScript runtime"}, {"deno", "tool", "deno", "Deno JavaScript runtime"}, {"pnpm", "tool", "pnpm", "pnpm package manager"}, {"yarn", "tool", "yarn", "Yarn Classic and modern package manager"}, {"python", "language", "python", "Python programming language"}, {"go", "language", "go", "Go programming language"}, {"java", "language", "java", "Java development kit"}, {"docker", "tool", "docker", "Docker container runtime"}, {"homebrew", "tool", "homebrew", "Homebrew package manager (macOS)"}, {"git", "tool", "git", "Git version control system"},
}

// GoConfigLoader is supplied by the CLI, which compiles executable Go configurations.
var GoConfigLoader func(cwd, path string) (Config, error)
var envReference = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func LoadConfig(cwd, path string) (Config, error) { return LoadConfigWithEnv(cwd, path, os.LookupEnv) }
func LoadConfigWithEnv(cwd, path string, lookup func(string) (string, bool)) (Config, error) {
	if path == "" {
		for _, name := range []string{"genesis.config.go", "genesis.config.yaml", "genesis.config.yml", "genesis.config.json"} {
			candidate := filepath.Join(cwd, name)
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
		if path == "" {
			return Config{}, fmt.Errorf("no genesis.config.go, genesis.config.yaml, genesis.config.yml or genesis.config.json found")
		}
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	var config Config
	var err error
	extension := filepath.Ext(path)
	if extension != ".go" && extension != ".yaml" && extension != ".yml" && extension != ".json" {
		return config, fmt.Errorf("%s: config must use .go, .yaml, .yml or .json", path)
	}
	if filepath.Ext(path) == ".go" {
		if GoConfigLoader == nil {
			return config, fmt.Errorf("%s: executable Go configuration requires a GoConfigLoader", path)
		}
		config, err = GoConfigLoader(cwd, path)
		if err == nil {
			var value any
			data, _ := json.Marshal(config)
			err = json.Unmarshal(data, &value)
			if err == nil {
				value, err = interpolate(value, lookup, "config")
			}
			if err == nil {
				data, err = json.Marshal(value)
			}
			if err == nil {
				err = json.Unmarshal(data, &config)
			}
		}
		if err == nil {
			err = ValidateConfig(config)
		}
	} else {
		var file *os.File
		file, err = os.Open(path)
		if err == nil {
			defer file.Close()
			if info, statErr := file.Stat(); statErr != nil {
				return config, statErr
			} else if info.Size() > 8<<20 {
				return config, fmt.Errorf("%s: config exceeds 8 MiB", path)
			}
			decoder := yaml.NewDecoder(io.LimitReader(file, 8<<20))
			var raw any
			err = decoder.Decode(&raw)
			if err == nil {
				var extra any
				if more := decoder.Decode(&extra); more != io.EOF {
					if more == nil {
						err = fmt.Errorf("multiple configuration documents are not supported")
					} else {
						err = more
					}
				}
			}
			if err == nil {
				raw, err = interpolate(raw, lookup, "config")
			}
			if err == nil {
				config, err = NormalizeConfig(raw)
			}
		}
	}
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return config, nil
}
func interpolate(value any, lookup func(string) (string, bool), location string) (any, error) {
	switch v := value.(type) {
	case string:
		var missing error
		out := envReference.ReplaceAllStringFunc(v, func(match string) string {
			name := envReference.FindStringSubmatch(match)[1]
			text, ok := lookup(name)
			if !ok {
				missing = fmt.Errorf("%s: environment variable '%s' is not set", location, name)
			}
			return text
		})
		return out, missing
	case []any:
		for i, e := range v {
			out, err := interpolate(e, lookup, fmt.Sprintf("%s[%d]", location, i))
			if err != nil {
				return nil, err
			}
			v[i] = out
		}
	case map[string]any:
		for key, e := range v {
			out, err := interpolate(e, lookup, location+"."+key)
			if err != nil {
				return nil, err
			}
			v[key] = out
		}
	}
	return value, nil
}
func NormalizeConfig(raw any) (Config, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return Config{}, fmt.Errorf("config: expected object")
	}
	if err := validateRawConfig(m); err != nil {
		return Config{}, err
	}
	for _, section := range []string{"tools", "sdks", "languages"} {
		entries, ok := m[section].([]any)
		if !ok {
			continue
		}
		for index, entry := range entries {
			obj, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			kind, has := obj["type"]
			if !has {
				continue
			}
			name, ok := kind.(string)
			if !ok {
				return Config{}, fmt.Errorf("%s[%d].type: expected string", section, index)
			}
			var meta *PluginMetadata
			for i := range BuiltinPlugins {
				if BuiltinPlugins[i].ID == name {
					meta = &BuiltinPlugins[i]
					break
				}
			}
			if meta == nil {
				return Config{}, fmt.Errorf("unknown plugin type '%s' (%s[%d])", name, section, index)
			}
			opts := map[string]any{}
			for k, v := range obj {
				if k != "type" {
					opts[k] = v
				}
			}
			entries[index] = map[string]any{"id": name, "category": meta.Category, "module": meta.Module, "options": opts}
		}
	}
	if entries, ok := m["repositories"].([]any); ok {
		for _, entry := range entries {
			if obj, ok := entry.(map[string]any); ok {
				if _, has := obj["folder"]; !has {
					if path, ok := obj["path"]; ok {
						obj["folder"] = path
						delete(obj, "path")
					}
				}
			}
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err = json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, ValidateConfig(config)
}
func ValidateConfig(config Config) error {
	seen := map[string]bool{}
	for _, section := range []struct {
		name    string
		entries []Instance
	}{{"tools", config.Tools}, {"sdks", config.SDKs}, {"languages", config.Languages}} {
		for i, p := range section.entries {
			prefix := fmt.Sprintf("%s[%d]", section.name, i)
			if strings.TrimSpace(p.ID) == "" {
				return fmt.Errorf("%s.id: required nonempty string", prefix)
			}
			if p.ID != strings.TrimSpace(p.ID) {
				return fmt.Errorf("%s.id: leading/trailing whitespace not allowed", prefix)
			}
			switch p.Category {
			case "tool", "sdk", "language", "library", "framework":
			default:
				return fmt.Errorf("%s.category: unknown category '%s'", prefix, p.Category)
			}
			if strings.TrimSpace(p.Module) == "" {
				return fmt.Errorf("%s.module: required nonempty string", prefix)
			}
			if seen[p.ID] {
				return fmt.Errorf("%s.id: duplicate plugin id '%s'", prefix, p.ID)
			}
			seen[p.ID] = true
		}
	}
	for i, r := range config.Repositories {
		if r.URL == "" || r.Folder == "" {
			return fmt.Errorf("repositories[%d]: url and folder are required", i)
		}
	}
	for i, s := range config.Scripts {
		if s.Command == "" {
			return fmt.Errorf("scripts[%d].command: required nonempty string", i)
		}
		if s.When != "" && s.When != "before" && s.When != "after" {
			return fmt.Errorf("scripts[%d].when: expected before or after", i)
		}
	}
	for k := range config.Env {
		if k == "" || strings.ContainsAny(k, "=\x00") {
			return fmt.Errorf("env: invalid environment variable name %q", k)
		}
	}
	return nil
}
func CollectInstances(config Config) []Instance {
	all := make([]Instance, 0, len(config.Tools)+len(config.SDKs)+len(config.Languages))
	all = append(all, config.Tools...)
	all = append(all, config.SDKs...)
	return append(all, config.Languages...)
}

func validateRawConfig(m map[string]any) error {
	for _, section := range []string{"tools", "sdks", "languages", "repositories", "scripts"} {
		raw, exists := m[section]
		if !exists {
			continue
		}
		entries, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("%s: expected array", section)
		}
		for i, entry := range entries {
			obj, ok := entry.(map[string]any)
			if !ok {
				return fmt.Errorf("%s[%d]: expected object", section, i)
			}
			fields := []string{}
			switch section {
			case "tools", "sdks", "languages":
				if _, ok := obj["type"]; ok {
					fields = []string{"type"}
				} else {
					fields = []string{"id", "category", "module"}
				}
			case "repositories":
				fields = []string{"url"}
				if _, ok := obj["folder"]; ok {
					fields = append(fields, "folder")
				} else {
					fields = append(fields, "path")
				}
			case "scripts":
				fields = []string{"name", "command"}
			}
			for _, field := range fields {
				if _, ok := obj[field].(string); !ok {
					return fmt.Errorf("%s[%d].%s: expected string", section, i, field)
				}
			}
		}
	}
	if raw, exists := m["env"]; exists {
		env, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("env: expected object")
		}
		for k, v := range env {
			if _, ok := v.(string); !ok {
				return fmt.Errorf("env.%s: expected string", k)
			}
		}
	}
	return nil
}
