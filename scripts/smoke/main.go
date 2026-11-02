// Smoke runners deliberately change only temporary prefixes or explicitly disposable hosts.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ossl-dev/genesis/packages/core"
	"github.com/ossl-dev/genesis/packages/plugins"
)

func main() {
	mode := "installers"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	ctx := context.Background()
	var err error
	switch mode {
	case "installers":
		err = installers(ctx)
	case "packages":
		err = packages(ctx)
	case "docker":
		err = dockerPackages(ctx)
	default:
		err = fmt.Errorf("unknown smoke suite %q", mode)
	}
	if err != nil {
		log.Fatal(err)
	}
}
func tempRuntime(ctx context.Context) (*core.Runtime, func(), error) {
	dir, err := os.MkdirTemp("", "genesis smoke-")
	if err != nil {
		return nil, nil, err
	}
	r := &core.Runtime{Context: ctx, Cwd: dir, Logger: log.New(os.Stdout, "", 0)}
	r.Ensure()
	return r, func() { os.RemoveAll(dir) }, nil
}
func installers(ctx context.Context) error {
	r, cleanup, err := tempRuntime(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	base := r.Env.Snapshot()
	base["npm_config_cache"] = filepath.Join(r.Cwd, "npm-cache")
	cases := []struct {
		name    string
		plugin  core.Plugin
		options map[string]any
	}{{"node", plugins.Node(), map[string]any{"version": "22.18.0", "use_nvm": false}}, {"go", plugins.Go(), map[string]any{"version": "1.22.5"}}, {"java", plugins.Java(), map[string]any{"version": "17.0.9"}}, {"bun", plugins.Bun(), map[string]any{"version": "1.3.2"}}, {"deno", plugins.Deno(), map[string]any{"version": "2.5.4"}}, {"pnpm", plugins.Pnpm(), map[string]any{"version": "10.19.0", "node_plugin": nil}}, {"yarn", plugins.Yarn(), map[string]any{"version": "4.9.4", "node_plugin": nil}}, {"git", plugins.Git(), map[string]any{"version": "2.53.0", "install_method": "source"}}, {"yarn-classic", plugins.Yarn(), map[string]any{"version": "1.22.22", "node_plugin": nil}}}
	for _, check := range cases {
		options := check.options
		options["install_dir"] = filepath.Join(r.Cwd, check.name)
		if check.name == "git" && runtime.GOOS == "windows" {
			options["install_method"] = "binary"
		}
		options, err = check.plugin.ParseOptions(options)
		if err != nil {
			return err
		}
		i := core.Instance{ID: check.name, Category: check.plugin.Category, Module: check.plugin.ID, Options: options}
		current := *r
		current.Env = core.NewEnvironment(base)
		current.Tasks = core.NewTaskRegistry()
		applied, err := check.plugin.Apply(&current, i)
		if err != nil || !applied.OK || !applied.DidChange {
			return fmt.Errorf("%s fresh apply: %+v %v", check.name, applied, err)
		}
		verified, err := check.plugin.Validate(&current, i)
		if err != nil || !verified.OK {
			return fmt.Errorf("%s validate: %+v %v", check.name, verified, err)
		}
		fresh := *r
		fresh.Env = core.NewEnvironment(base)
		repeated, err := check.plugin.Apply(&fresh, i)
		if err != nil || !repeated.OK || repeated.DidChange {
			return fmt.Errorf("%s rediscovery: %+v %v", check.name, repeated, err)
		}
		if check.name == "node" {
			base = current.Env.Snapshot()
		}
		fmt.Printf("%s installed, validated, rediscovered on %s/%s\n", check.name, runtime.GOOS, runtime.GOARCH)
	}
	return nil
}
func disposable() error {
	if runtime.GOOS != "linux" || os.Getuid() != 0 || os.Getenv("GENESIS_DISPOSABLE_HOST") != "1" {
		return errors.New("requires disposable Linux root host and GENESIS_DISPOSABLE_HOST=1")
	}
	return nil
}
func packages(ctx context.Context) error {
	if err := disposable(); err != nil {
		return err
	}
	r, cleanup, err := tempRuntime(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	if err = core.RegisterPackageTasks(r, "git", "build-essential", "libcurl4-openssl-dev", "zlib1g-dev", "libexpat1-dev"); err != nil {
		return err
	}
	results, err := r.Tasks.ExecuteAll(ctx)
	if err != nil {
		return err
	}
	for id, result := range results {
		if !result.OK {
			return fmt.Errorf("%s: %+v", id, result)
		}
	}
	source := filepath.Join(r.Cwd, "probe.c")
	program := `#include <curl/curl.h>
#include <expat.h>
#include <zlib.h>
int main(void) {return !(curl_version() && XML_ExpatVersion() && zlibVersion());}
`
	if err = os.WriteFile(source, []byte(program), 0600); err != nil {
		return err
	}
	for _, command := range [][]string{{"git", "--version"}, {"cc", source, "-lcurl", "-lexpat", "-lz", "-o", filepath.Join(r.Cwd, "probe")}, {filepath.Join(r.Cwd, "probe")}} {
		result, err := r.Run(command[0], command[1:]...)
		if err != nil || result.Code != 0 {
			return fmt.Errorf("%v: %+v %v", command, result, err)
		}
	}
	fmt.Println("Git, compiler, mapped build libraries verified")
	return nil
}
func dockerPackages(ctx context.Context) error {
	if err := disposable(); err != nil {
		return err
	}
	r, cleanup, err := tempRuntime(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	bin := filepath.Join(r.Cwd, "bin")
	if err = os.Mkdir(bin, 0755); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(bin, "systemctl"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		return err
	}
	r.Env.PrependPath(bin)
	plugin := plugins.Docker()
	options, err := plugin.ParseOptions(map[string]any{"version": "29", "include_compose": false})
	if err != nil {
		return err
	}
	i := core.Instance{ID: "docker", Category: "tool", Module: "docker", Options: options}
	if err = plugin.Prepare(r, i); err != nil {
		return err
	}
	if err = plugin.RegisterTasks(r, i); err != nil {
		return err
	}
	results, err := r.Tasks.ExecuteAll(ctx)
	if err != nil {
		return err
	}
	for id, result := range results {
		if !result.OK {
			return fmt.Errorf("%s: %+v", id, result)
		}
	}
	if err = plugins.InstallDockerLinux(r, i, func() {}, false); err != nil {
		return err
	}
	found, err := plugin.Detect(r, i)
	if err != nil || !found.OK {
		return fmt.Errorf("Docker pin not detected: %+v %v", found, err)
	}
	compose, err := r.Run("docker", "compose", "version")
	if err != nil {
		return err
	}
	if compose.Code == 0 {
		return errors.New("Compose installed despite include_compose:false")
	}
	i.Options["include_compose"] = true
	if err = plugins.InstallDockerLinux(r, i, func() {}, true); err != nil {
		return err
	}
	found, err = plugin.Detect(r, i)
	if err != nil || !found.OK {
		return fmt.Errorf("Compose not detected: %+v %v", found, err)
	}
	applied, err := plugin.Apply(r, i)
	if err != nil {
		return err
	}
	if applied.OK || applied.DidChange {
		return fmt.Errorf("missing daemon accepted: %+v", applied)
	}
	fmt.Println("Docker pins, optional Compose, missing-daemon errors verified; systemctl was stubbed")
	return nil
}
