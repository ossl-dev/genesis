package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ossl-dev/genesis/packages/core"
	"github.com/ossl-dev/genesis/packages/extension"
	"github.com/ossl-dev/genesis/packages/plugins"
)

const version = "0.2.0"

type cliOptions struct {
	config                   string
	dry, json, cloud, local  bool
	format, token, url, name string
	workers                  int
}

func Run(ctx context.Context, args []string, cwd string, in io.Reader, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprintln(out, "Genesis — rebuild developer environments\n\nUsage: genesis <command> [options]\nCommands: create init apply doctor diff validate list-plugins login list\n\nUse genesis <command> --help for options.")
		return err
	}
	if args[0] == "--version" || args[0] == "version" {
		_, err := fmt.Fprintln(out, version)
		return err
	}
	command := args[0]
	opts := cliOptions{}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errOut)
	switch command {
	case "apply", "doctor", "diff", "validate":
		fs.StringVar(&opts.config, "config", "", "Path to configuration")
		if command == "apply" {
			fs.BoolVar(&opts.dry, "dry-run", false, "Plan without provisioning")
			fs.BoolVar(&opts.cloud, "cloud", false, "Cloud environment (unavailable)")
			fs.IntVar(&opts.workers, "parallel", 1, "Maximum concurrent plugins")
		}
		fs.BoolVar(&opts.json, "json", false, "JSON output")
	case "list-plugins":
		fs.BoolVar(&opts.json, "json", false, "JSON output")
	case "list":
		fs.BoolVar(&opts.cloud, "cloud", false, "Include cloud metadata (unavailable)")
		fs.BoolVar(&opts.local, "local", true, "Include local environments")
		fs.StringVar(&opts.format, "format", "table", "table or json")
	case "login":
		fs.StringVar(&opts.token, "token", "", "Token metadata to save")
		fs.StringVar(&opts.token, "t", "", "Token metadata to save")
		fs.StringVar(&opts.url, "url", "https://cloud.genesis-docs.vercel.app", "Cloud URL")
		fs.StringVar(&opts.url, "u", "https://cloud.genesis-docs.vercel.app", "Cloud URL")
	case "init", "create":
		fs.StringVar(&opts.format, "format", "", "yaml or go")
		fs.StringVar(&opts.name, "name", "", "Environment name")
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	// Accept flags after positional arguments, as the previous CLI did.
	flags, positionals, err := interspersed(fs, args[1:])
	if err != nil {
		return err
	}
	if err = fs.Parse(flags); errors.Is(err, flag.ErrHelp) {
		return nil
	} else if err != nil {
		return err
	}
	if len(positionals) > 0 {
		if command == "apply" {
			return errors.New("cloud environment apply is not available; use a local config")
		}
		return fmt.Errorf("unexpected argument %q", positionals[0])
	}
	switch command {
	case "init", "create":
		return scaffold(cwd, command, opts, in, out)
	case "login":
		return login(opts, out)
	case "list":
		return list(cwd, opts, out, errOut)
	case "list-plugins":
		if opts.json {
			return writeJSON(out, core.BuiltinPlugins)
		}
		fmt.Fprintf(out, "Available plugins (%d):\n", len(core.BuiltinPlugins))
		for _, p := range core.BuiltinPlugins {
			fmt.Fprintf(out, "  %-12s %s\n", p.ID, p.Description)
		}
		return nil
	}
	if command == "apply" && opts.cloud {
		return errors.New("cloud environment apply is not available; use a local config")
	}
	if command == "apply" && opts.json && !opts.dry {
		return errors.New("--json requires --dry-run")
	}
	if command == "apply" && opts.workers < 1 {
		return errors.New("--parallel must be a positive integer")
	}
	if os.Getenv("GENESIS_DEBUG") != "" && !opts.json {
		fmt.Fprintf(errOut, "genesis: loading config from %s\n", cwd)
	}
	config, err := core.LoadConfigWithOptions(cwd, opts.config, core.ConfigLoadOptions{GoConfigLoader: func(dir, path string) (core.Config, error) { return extension.LoadGoConfigContext(ctx, dir, path) }})
	if err != nil {
		return err
	}
	nodes, err := plugins.LoadWithResolver(config, func(i core.Instance) (core.Plugin, error) { return extension.Load(ctx, cwd, i) })
	if err != nil {
		return err
	}
	nodes, err = core.BuildPluginGraph(nodes)
	if err != nil {
		return err
	}
	logger := log.New(errOut, "genesis: ", 0)
	if opts.json {
		logger = log.New(io.Discard, "", 0)
	}
	if os.Getenv("GENESIS_DEBUG") != "" && !opts.json {
		logger.Printf("Loaded %d plugins in dependency order", len(nodes))
	}
	runtime := &core.Runtime{Context: ctx, Cwd: cwd, Logger: logger}
	runtime.Ensure()
	for key, value := range config.Env {
		runtime.Env.Set(key, value)
	}
	switch command {
	case "apply":
		if opts.dry {
			plan, err := core.CreateEnvironmentPlan(config, nodes)
			if err != nil {
				return err
			}
			if opts.json {
				return writeJSON(out, plan)
			}
			fmt.Fprintln(out, "Dry run: no provisioning commands executed")
			return writeJSON(out, plan)
		}
		var summaries []core.ApplySummary
		if opts.workers == 1 {
			summaries, err = core.ApplyEnvironment(config, nodes, runtime)
		} else {
			summaries, err = core.ApplyEnvironmentParallel(config, nodes, runtime, opts.workers)
		}
		if err != nil {
			return err
		}
		table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(table, "Plugin\tStatus\tDetails")
		failed := false
		for _, s := range summaries {
			status := "ok"
			if !s.OK {
				failed = true
				status = "error"
			}
			fmt.Fprintf(table, "%s\t%s\t%s\n", s.ID, status, s.Details)
		}
		table.Flush()
		if failed {
			return errors.New("environment apply failed")
		}
		fmt.Fprintln(out, "Environment applied successfully")
		return nil
	case "diff":
		summaries, err := core.RunDiff(nodes, runtime)
		if err != nil {
			return err
		}
		if opts.json {
			return writeJSON(out, summaries)
		}
		for _, s := range summaries {
			fmt.Fprintf(out, "%s\t%s\t%s\n", s.ID, s.Status, s.Details)
		}
		return nil
	case "validate", "doctor":
		var detected []core.DetectSummary
		if command == "doctor" {
			detected, err = core.RunDetect(nodes, runtime)
			if err != nil {
				return err
			}
		}
		summaries, err := core.RunValidate(nodes, runtime)
		if err != nil {
			return err
		}
		failed := false
		for _, s := range summaries {
			if !s.OK {
				failed = true
			}
		}
		if command == "doctor" {
			detect := detected
			for _, d := range detect {
				if d.Status == "missing" {
					failed = true
				}
			}
			if opts.json {
				if err = writeJSON(out, map[string]any{"detect": detect, "validate": summaries}); err != nil {
					return err
				}
			} else {
				for _, d := range detect {
					status, details := d.Status, d.Details
					for _, validated := range summaries {
						if validated.ID == d.ID {
							if validated.Message != "" {
								details = validated.Message
							}
							if !validated.OK {
								status = "error"
							} else if d.Status == "present" {
								status = "ok"
							}
							break
						}
					}
					fmt.Fprintf(out, "%s\t%s\t%s\n", d.ID, status, details)
				}
			}
		} else if opts.json {
			if err = writeJSON(out, summaries); err != nil {
				return err
			}
		} else {
			for _, s := range summaries {
				fmt.Fprintf(out, "%s\t%t\t%s\n", s.ID, s.OK, s.Message)
			}
		}
		if failed {
			return fmt.Errorf("environment %s failed", command)
		}
		return nil
	}
	return nil
}

func interspersed(fs *flag.FlagSet, args []string) ([]string, []string, error) {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if strings.Contains(name, "=") {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		boolean, ok := f.Value.(interface{ IsBoolFlag() bool })
		if ok && boolean.IsBoolFlag() {
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("flag needs an argument: %s", arg)
		}
		i++
		flags = append(flags, args[i])
	}
	return flags, positionals, nil
}
func writeJSON(out io.Writer, value any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func scaffold(cwd, command string, opts cliOptions, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	if opts.name == "" {
		opts.name = "genesis-env"
		if command == "create" {
			fmt.Fprint(out, "Environment name [genesis-env]: ")
			name, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return err
			}
			if strings.TrimSpace(name) != "" {
				opts.name = strings.TrimSpace(name)
			}
		}
	}
	if opts.format == "" {
		fmt.Fprint(out, "Config format [yaml/go] (yaml): ")
		format, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		opts.format = strings.TrimSpace(format)
		if opts.format == "" {
			opts.format = "yaml"
		}
	}
	if opts.format != "yaml" && opts.format != "go" {
		return errors.New("config format must be yaml or go")
	}
	source := "tools: []\nsdks: []\nlanguages: []\nrepositories: []\nscripts: []\nenv: {}\n"
	if opts.format == "go" {
		source = "//go:build ignore\n\npackage main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(`{\"tools\":[],\"languages\":[],\"repositories\":[],\"scripts\":[],\"env\":{}}`)\n}\n"
	}
	if err := writeNew(filepath.Join(cwd, "genesis.config."+opts.format), []byte(source), 0644); err != nil {
		return err
	}
	if err := writeNew(filepath.Join(cwd, "README.md"), []byte("# "+opts.name+"\n\nManaged by Genesis.\n"), 0644); err != nil {
		return err
	}
	fmt.Fprintln(out, "Configuration ready. Preview: genesis apply --dry-run")
	return nil
}
func writeNew(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

type tokenMetadata struct {
	Token   string `json:"token"`
	URL     string `json:"url"`
	SavedAt string `json:"savedAt"`
}

func genesisHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".genesis"), nil
}
func login(opts cliOptions, out io.Writer) error {
	home, err := genesisHome()
	if err != nil {
		return err
	}
	path := filepath.Join(home, "auth.json")
	if opts.token != "" {
		if err = os.MkdirAll(home, 0700); err != nil {
			return err
		}
		data, err := json.Marshal(tokenMetadata{opts.token, opts.url, time.Now().UTC().Format(time.RFC3339)})
		if err != nil {
			return err
		}
		if err = core.WritePrivateFile(path, data); err != nil {
			return err
		}
		fmt.Fprintf(out, "Token saved for %s\nToken saved to %s\n", opts.url, path)
		return nil
	}
	data, err := os.ReadFile(path)
	var saved tokenMetadata
	if err == nil && json.Unmarshal(data, &saved) == nil && saved.Token != "" && saved.URL != "" {
		fmt.Fprintf(out, "A token is saved for %s\nSaved: %s\nReplace: genesis login --token <new-token>\n", saved.URL, saved.SavedAt)
		return nil
	}
	fmt.Fprintln(out, "Cloud OAuth is not implemented. Use genesis login --token <token> to save metadata locally.")
	return nil
}

type environmentEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

func list(cwd string, opts cliOptions, out, errOut io.Writer) error {
	if opts.format != "table" && opts.format != "json" {
		return errors.New("format must be table or json")
	}
	entries := []environmentEntry{}
	if opts.local {
		for _, name := range []string{"genesis.config.go", "genesis.config.yaml", "genesis.config.yml", "genesis.config.json"} {
			if _, err := os.Stat(filepath.Join(cwd, name)); err == nil {
				entries = append(entries, environmentEntry{ID: "current", Name: "Current Project", Description: "Loaded from " + name, Source: "local"})
				break
			}
		}
		home, err := genesisHome()
		if err != nil {
			return err
		}
		files, _ := os.ReadDir(filepath.Join(home, "cache"))
		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}
			path := filepath.Join(home, "cache", file.Name())
			info, err := os.Stat(path)
			if err != nil || info.Size() > 8<<20 {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var entry environmentEntry
			if json.Unmarshal(data, &entry) != nil {
				continue
			}
			if entry.ID == "" {
				entry.ID = strings.TrimSuffix(file.Name(), ".json")
			}
			if entry.Name == "" {
				entry.Name = entry.ID
			}
			entry.Source = "local"
			entries = append(entries, entry)
		}
	}
	if opts.cloud {
		fmt.Fprintln(errOut, "Cloud API is not implemented; listing local metadata only.")
	}
	if opts.format == "json" {
		return writeJSON(out, entries)
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "No environments found. Run genesis init to create a configuration.")
		return nil
	}
	for _, entry := range entries {
		fmt.Fprintf(out, "%-20s %s\n", entry.ID, entry.Name)
	}
	return nil
}
