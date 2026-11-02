package plugins

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

func TestSelectDockerPackages(t *testing.T) {
	packages := []DockerPackageVersion{{"docker-ce", "3:29.1.0-1.fc43"}, {"docker-ce-cli", "1:29.1.0-1.fc43"}, {"docker-ce", "3:29.1.1-1.fc43"}, {"docker-ce-cli", "1:29.1.1-1.fc43"}, {"docker-ce", "3:29.1.2-1.fc43"}}
	actual, err := SelectDockerPackages(packages, "29.1", "-")
	expected := []string{"docker-ce-3:29.1.1-1.fc43", "docker-ce-cli-1:29.1.1-1.fc43"}
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%+v %v", actual, err)
	}
	for _, version := range []string{"5:29.10.0-1", "5:29.1.0~rc1-1", "5:29.1.0-beta1-1", "5:29.1.0-alpha1-1"} {
		if _, err := SelectDockerPackages([]DockerPackageVersion{{"docker-ce", version}, {"docker-ce-cli", version}}, "29.1", "="); err == nil {
			t.Fatal(version)
		}
	}
	packages = []DockerPackageVersion{}
	for _, version := range []string{"5:29.1.0-9", "5:29.1.0-10"} {
		packages = append(packages, DockerPackageVersion{"docker-ce", version}, DockerPackageVersion{"docker-ce-cli", version})
	}
	actual, err = SelectDockerPackages(packages, "29.1", "=")
	if err != nil || !strings.Contains(actual[0], "-10") {
		t.Fatalf("%+v %v", actual, err)
	}
}
func TestDockerDistributionValidation(t *testing.T) {
	for _, id := range []string{"arch", "alpine", "notubuntu"} {
		if _, err := validateDockerDistribution(core.LinuxDistribution{ID: id, Like: []string{"ubuntu"}, Codename: "noble"}, "amd64"); err == nil {
			t.Fatal(id)
		}
	}
	for _, codename := range []string{"", "noble;bad", "noble\nbad", "$(id)"} {
		if _, err := validateDockerDistribution(core.LinuxDistribution{ID: "ubuntu", Codename: codename}, "amd64"); err == nil {
			t.Fatal(codename)
		}
	}
	if _, err := validateDockerDistribution(core.LinuxDistribution{ID: "fedora"}, "386"); err == nil {
		t.Fatal("unsupported architecture accepted")
	}
	for _, id := range []string{"debian", "ubuntu", "fedora", "centos", "rhel"} {
		if _, err := validateDockerDistribution(core.LinuxDistribution{ID: id, Codename: "bookworm"}, "arm64"); err != nil {
			t.Fatal(err)
		}
	}
}
func platformUnwrapSudo(command string, args []string) (string, []string) {
	if command == "sudo" && len(args) > 0 {
		return args[0], args[1:]
	}
	return command, args
}
func TestDockerAptExactPinsAndAtomicRepositoryWrites(t *testing.T) {
	runtime, commands := platformTestRuntime(t, "linux", func(command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
		command, args = platformUnwrapSudo(command, args)
		output := ""
		if command == "dpkg" {
			output = "amd64"
		}
		if command == "apt-cache" {
			output = args[1] + " | 5:29.1.0-1~debian.12~bookworm | repo\n"
		}
		return core.CommandResult{Stdout: output}, nil
	})
	runtime.Arch = "amd64"
	changed := false
	var downloads []string
	err := installDockerLinuxForDistribution(runtime, platformTestInstance(t, Docker(), map[string]any{"version": "29.1", "include_compose": false}), func() { changed = true }, false, core.LinuxDistribution{ID: "debian", Codename: "bookworm"}, func(_ context.Context, url string) (string, error) {
		downloads = append(downloads, url)
		return "repository contents", nil
	})
	if err != nil || !changed {
		t.Fatalf("%v %v", changed, err)
	}
	if !reflect.DeepEqual(downloads, []string{"https://download.docker.com/linux/debian/gpg"}) {
		t.Fatal(downloads)
	}
	installed := false
	started := false
	writes := 0
	for _, call := range *commands {
		command, args := platformUnwrapSudo(call.Command, call.Args)
		if call.Options.Cwd != runtime.Cwd || call.Options.Env["PATH"] != "/host/bin" {
			t.Fatal(call)
		}
		switch command {
		case "bash":
			writes++
			if !strings.Contains(args[1], "mktemp") || !strings.Contains(args[1], "mv -f") || strings.Contains(args[1], "bookworm") {
				t.Fatal(args)
			}
		case "apt-get":
			if args[0] == "install" {
				installed = true
				expected := []string{"install", "-y", "--no-install-recommends", "--allow-downgrades", "docker-ce=5:29.1.0-1~debian.12~bookworm", "docker-ce-cli=5:29.1.0-1~debian.12~bookworm", "containerd.io", "docker-buildx-plugin"}
				if !reflect.DeepEqual(args, expected) {
					t.Fatal(args)
				}
			}
		case "systemctl":
			started = true
			if !reflect.DeepEqual(args, []string{"enable", "--now", "docker"}) {
				t.Fatal(args)
			}
		}
		if strings.Contains(strings.Join(args, " "), "usermod") {
			t.Fatal(call)
		}
	}
	if !installed || !started || writes != 2 {
		t.Fatalf("installed=%v started=%v writes=%d", installed, started, writes)
	}
}
func TestDockerRPMRepositoriesAndEpochs(t *testing.T) {
	for _, id := range []string{"fedora", "centos", "rhel"} {
		t.Run(id, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, "linux", func(command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				command, args = platformUnwrapSudo(command, args)
				if command == "dnf" && args[0] == "repoquery" {
					return core.CommandResult{Stdout: "docker-ce|3:29.1.0-1.fc43|x86_64\ndocker-ce-cli|1:29.1.0-1.fc43|x86_64\ndocker-ce|3:30.0.0-1.fc43|aarch64\n"}, nil
				}
				return core.CommandResult{}, nil
			})
			runtime.Arch = "amd64"
			var url string
			err := installDockerLinuxForDistribution(runtime, platformTestInstance(t, Docker(), map[string]any{"version": "29.1"}), func() {}, false, core.LinuxDistribution{ID: id}, func(_ context.Context, source string) (string, error) { url = source; return "contents", nil })
			if err != nil {
				t.Fatal(err)
			}
			if url != "https://download.docker.com/linux/"+id+"/docker-ce.repo" {
				t.Fatal(url)
			}
			installed := false
			for _, call := range *commands {
				command, args := platformUnwrapSudo(call.Command, call.Args)
				if command == "dnf" {
					if args[0] == "repoquery" && args[3] != "%{name}|%{evr}|%{arch}\\n" {
						t.Fatal(args)
					}
					if args[0] == "install" {
						installed = true
						expected := []string{"install", "-y", "--setopt=install_weak_deps=False", "docker-ce-3:29.1.0-1.fc43", "docker-ce-cli-1:29.1.0-1.fc43", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"}
						if !reflect.DeepEqual(args, expected) {
							t.Fatal(args)
						}
					}
					if args[0] == "update" {
						t.Fatal("host-wide update")
					}
				}
			}
			if !installed {
				t.Fatal("missing install")
			}
		})
	}
}
func TestDockerComposeOnlyPreservesEngineAndService(t *testing.T) {
	for _, id := range []string{"debian", "fedora"} {
		t.Run(id, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, "linux", func(command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				command, _ = platformUnwrapSudo(command, args)
				if command == "dpkg" {
					return core.CommandResult{Stdout: "arm64"}, nil
				}
				return core.CommandResult{}, nil
			})
			err := installDockerLinuxForDistribution(runtime, platformTestInstance(t, Docker(), nil), func() {}, true, core.LinuxDistribution{ID: id, Codename: "bookworm"}, func(context.Context, string) (string, error) { return "contents", nil })
			if err != nil {
				t.Fatal(err)
			}
			for _, call := range *commands {
				command, args := platformUnwrapSudo(call.Command, call.Args)
				if command == "systemctl" || command == "apt-cache" || (command == "dnf" && args[0] == "repoquery") {
					t.Fatal(call)
				}
				for _, arg := range args {
					if strings.HasPrefix(arg, "docker-ce=") || strings.HasPrefix(arg, "docker-ce-") || arg == "containerd.io" {
						t.Fatal(call)
					}
				}
			}
		})
	}
}
func TestDockerInstallationFailures(t *testing.T) {
	for _, failure := range []string{"download", "write", "unmatched"} {
		t.Run(failure, func(t *testing.T) {
			runtime, commands := platformTestRuntime(t, "linux", func(command string, args []string, _ core.CommandOptions) (core.CommandResult, error) {
				command, args = platformUnwrapSudo(command, args)
				if command == "dpkg" {
					return core.CommandResult{Stdout: "amd64"}, nil
				}
				if command == "bash" && failure == "write" {
					return core.CommandResult{Code: 1, Stderr: "write failed"}, nil
				}
				if command == "apt-cache" {
					return core.CommandResult{Stdout: args[1] + " | 5:28.0.0-1 | repo"}, nil
				}
				return core.CommandResult{}, nil
			})
			runtime.Arch = "amd64"
			err := installDockerLinuxForDistribution(runtime, platformTestInstance(t, Docker(), map[string]any{"version": "29.1"}), func() {}, false, core.LinuxDistribution{ID: "debian", Codename: "bookworm"}, func(context.Context, string) (string, error) {
				if failure == "download" {
					return "", fmt.Errorf("download failed")
				}
				return "contents", nil
			})
			if err == nil {
				t.Fatal("failure ignored")
			}
			for _, call := range *commands {
				command, args := platformUnwrapSudo(call.Command, call.Args)
				if command == "apt-get" && args[0] == "install" {
					t.Fatal("installed on failure")
				}
			}
		})
	}
}
