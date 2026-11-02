package plugins

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

var npmVersion = regexp.MustCompile(`^(\d+\.\d+\.\d+)$`)

func npmPlugin(name string) core.Plugin {
	bin := func(r *core.Runtime, root string) string {
		if r.Platform == "windows" {
			return root
		}
		return filepath.Join(root, "bin")
	}
	executable := func(r *core.Runtime, root string) string {
		filename := name
		if r.Platform == "windows" {
			filename += ".cmd"
		}
		return filepath.Join(bin(r, root), filename)
	}
	check := func(r *core.Runtime, i core.Instance, command string) (core.DetectResult, error) {
		env := r.Env.Snapshot()
		env["YARN_IGNORE_PATH"] = "1"
		env["COREPACK_ENABLE_PROJECT_SPEC"] = "0"
		result, err := r.RunWithOptions(command, []string{"--version"}, core.CommandOptions{Env: env})
		if err != nil {
			return core.DetectResult{}, err
		}
		version := npmVersion.FindStringSubmatch(strings.TrimSpace(result.Stdout))
		if result.Code != 0 || len(version) < 2 {
			return core.DetectResult{Details: name + " is unavailable: " + result.Stderr}, nil
		}
		return core.DetectResult{OK: MatchesVersion(version[1], optionString(i.Options, "version")), Details: fmt.Sprintf("Detected %s %s; requested %s", name, version[1], optionString(i.Options, "version"))}, nil
	}
	detect := func(r *core.Runtime, i core.Instance) (core.DetectResult, error) {
		r.Ensure()
		root, locationErr := managedDirectory(r, i, name)
		if locationErr != nil {
			return check(r, i, name)
		}
		command := name
		managed := exists(executable(r, root))
		if managed || optionString(i.Options, "install_dir") != "" {
			command = executable(r, root)
		}
		found, err := check(r, i, command)
		if err == nil && found.OK && managed {
			r.Env.PrependPath(bin(r, root))
		}
		return found, err
	}
	return core.Plugin{ID: name, Category: "tool", Prepare: func(r *core.Runtime, i core.Instance) error {
		r.Ensure()
		if _, err := managedDirectory(r, i, name); err != nil {
			detected, detectErr := check(r, i, name)
			if detectErr != nil {
				return detectErr
			}
			if !detected.OK {
				return err
			}
		}
		return nil
	}, DependsOn: []string{"node"}, ParseOptions: func(o map[string]any) (map[string]any, error) { return ParseOptions(name, o) }, Detect: detect, Apply: func(r *core.Runtime, i core.Instance) (core.ApplyResult, error) {
		r.Ensure()
		found, err := detect(r, i)
		if err != nil {
			return core.ApplyResult{}, err
		}
		if found.OK {
			return core.ApplyResult{OK: true, Details: found.Details}, nil
		}
		destination, err := managedDirectory(r, i, name)
		if err != nil {
			return core.ApplyResult{}, err
		}
		result, err := r.Run("npm", "--version")
		if err != nil {
			return core.ApplyResult{}, err
		}
		if result.Code != 0 {
			return core.ApplyResult{Details: "npm is required; configure a Node plugin or provide Node/npm on PATH"}, nil
		}
		packageName := name
		major, _ := strconv.Atoi(strings.Split(optionString(i.Options, "version"), ".")[0])
		if name == "yarn" && major >= 2 {
			packageName = "@yarnpkg/cli-dist"
		}
		relative, _ := filepath.Rel(destination, executable(r, destination))
		err = InstallDirectory(r, destination, relative, func(stage string) (string, error) {
			prefix := filepath.Join(stage, "prefix")
			result, err := r.Run("npm", "install", "--global", "--prefix", prefix, "--engine-strict", "--ignore-scripts", "--no-audit", "--no-fund", packageName+"@"+optionString(i.Options, "version"))
			if err != nil {
				return "", err
			}
			if result.Code != 0 {
				return "", fmt.Errorf("npm installation failed: %s %s", result.Stderr, result.Stdout)
			}
			return prefix, nil
		}, func(root string) error {
			verified, err := check(r, i, executable(r, root))
			if err != nil {
				return err
			}
			if !verified.OK {
				return errors.New(verified.Details)
			}
			return nil
		})
		if err != nil {
			var recovery *InstallationRecoveryError
			return failedApply(r, name+" installation failed: "+err.Error(), errors.As(err, &recovery), err)
		}
		r.Env.PrependPath(bin(r, destination))
		return core.ApplyResult{OK: true, DidChange: true, Details: fmt.Sprintf("%s %s installed to %s", name, optionString(i.Options, "version"), destination)}, nil
	}, Validate: func(r *core.Runtime, i core.Instance) (core.ValidateResult, error) {
		found, err := detect(r, i)
		return core.ValidateResult{OK: found.OK, Message: found.Details}, err
	}}
}
func Pnpm() core.Plugin { return npmPlugin("pnpm") }
func Yarn() core.Plugin { return npmPlugin("yarn") }
