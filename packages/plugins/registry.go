package plugins

import (
	"fmt"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

type Definition struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Module      string `json:"module"`
	Description string `json:"description"`
}

func Builtins() []Definition {
	catalog := make([]Definition, 0, len(core.BuiltinPlugins))
	for _, entry := range core.BuiltinPlugins {
		catalog = append(catalog, Definition{entry.ID, entry.Category, entry.Module, entry.Description})
	}
	return catalog
}

func Builtin(name string) (core.Plugin, bool) {
	switch name {
	case "node":
		return Node(), true
	case "bun":
		return Bun(), true
	case "deno":
		return Deno(), true
	case "pnpm":
		return Pnpm(), true
	case "yarn":
		return Yarn(), true
	case "python":
		return Python(), true
	case "go":
		return Go(), true
	case "java":
		return Java(), true
	case "docker":
		return Docker(), true
	case "homebrew":
		return Homebrew(), true
	case "git":
		return Git(), true
	}
	return core.Plugin{}, false
}
func Load(config core.Config) ([]core.Node, error) { return LoadWithResolver(config, nil) }
func LoadWithResolver(config core.Config, resolver func(core.Instance) (core.Plugin, error)) ([]core.Node, error) {
	instances := core.CollectInstances(config)
	nodes := make([]core.Node, 0, len(instances))
	for _, instance := range instances {
		name := instance.Module
		if name == "" {
			name = instance.ID
		}
		name = strings.TrimPrefix(strings.TrimPrefix(name, "builtin:"), "@ossl/genesis-plugins/")
		plugin, ok := Builtin(name)
		if !ok {
			if resolver == nil {
				return nil, fmt.Errorf("plugin %s: unknown module %s", instance.ID, instance.Module)
			}
			var err error
			plugin, err = resolver(instance)
			if err != nil {
				return nil, fmt.Errorf("plugin %s (%s): %w", instance.ID, instance.Module, err)
			}
			if plugin.ID != instance.ID || plugin.Category != instance.Category {
				return nil, fmt.Errorf("plugin %s returned invalid identity from %s", instance.ID, instance.Module)
			}
		}
		if plugin.ParseOptions != nil {
			options, err := plugin.ParseOptions(instance.Options)
			if err != nil {
				return nil, fmt.Errorf("plugin %s (%s): %w", instance.ID, instance.Module, err)
			}
			instance.Options = options
		}
		plugin.ID = instance.ID
		if instance.Category == "" {
			instance.Category = plugin.Category
		}
		plugin.Category = instance.Category
		if name == "pnpm" || name == "yarn" {
			if instance.Options["node_plugin"] == nil {
				plugin.DependsOn = nil
			} else {
				plugin.DependsOn = []string{optionString(instance.Options, "node_plugin")}
			}
		}
		nodes = append(nodes, core.Node{Instance: instance, Plugin: plugin})
	}
	return core.BuildPluginGraph(nodes)
}
