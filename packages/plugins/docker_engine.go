package plugins

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ossl-dev/genesis/packages/core"
)

var dockerCodenamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var dockerPackageValuePattern = regexp.MustCompile(`^[0-9][a-zA-Z0-9.+:~_-]*$`)
var dockerPackageVersionPattern = regexp.MustCompile(`^(\d+\.\d+\.\d+)(?:[-~]|$)`)
var dockerPreviewPattern = regexp.MustCompile(`(?i)(?:^|[.~_-])(?:rc|beta|alpha)`)
var dockerEpochPattern = regexp.MustCompile(`^\d+:`)

func DockerDistribution(runtime *core.Runtime) (core.LinuxDistribution, error) {
	distribution, err := core.GetLinuxDistribution()
	if err != nil {
		return core.LinuxDistribution{}, err
	}
	return validateDockerDistribution(distribution, runtime.Arch)
}
func validateDockerDistribution(distribution core.LinuxDistribution, arch string) (core.LinuxDistribution, error) {
	switch distribution.ID {
	case "ubuntu", "debian":
		if !dockerCodenamePattern.MatchString(distribution.Codename) {
			return distribution, fmt.Errorf("Docker APT installation requires a valid distribution codename")
		}
	case "fedora", "centos", "rhel":
	default:
		return distribution, fmt.Errorf("unsupported Linux distribution for Docker installation: %s", distribution.ID)
	}
	if arch != "amd64" && arch != "arm64" {
		return distribution, fmt.Errorf("unsupported Docker architecture: %s", arch)
	}
	return distribution, nil
}

type DockerPackageVersion struct{ Name, Value string }

func SelectDockerPackages(packages []DockerPackageVersion, requested, separator string) ([]string, error) {
	clients := make(map[string]string)
	for _, pkg := range packages {
		if pkg.Name == "docker-ce-cli" && dockerPackageValuePattern.MatchString(pkg.Value) {
			clients[dockerEpochPattern.ReplaceAllString(pkg.Value, "")] = pkg.Value
		}
	}
	type candidate struct{ version, engine, client string }
	var candidates []candidate
	for _, pkg := range packages {
		if pkg.Name != "docker-ce" || !dockerPackageValuePattern.MatchString(pkg.Value) {
			continue
		}
		key := dockerEpochPattern.ReplaceAllString(pkg.Value, "")
		version := dockerPackageVersionPattern.FindStringSubmatch(key)
		client, ok := clients[key]
		if version != nil && ok && !dockerPreviewPattern.MatchString(key) && (requested == "latest" || MatchesVersion(version[1], requested)) {
			candidates = append(candidates, candidate{version[1], pkg.Value, client})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if comparison := platformVersionCompare(candidates[i].version, candidates[j].version); comparison != 0 {
			return comparison > 0
		}
		return platformNaturalCompare(candidates[i].engine, candidates[j].engine) > 0
	})
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no matching Docker Engine/CLI packages for %s in the configured repository", requested)
	}
	selected := candidates[0]
	return []string{"docker-ce" + separator + selected.engine, "docker-ce-cli" + separator + selected.client}, nil
}
func InstallDockerLinux(runtime *core.Runtime, instance core.Instance, changed func(), composeOnly bool) error {
	distribution, err := DockerDistribution(runtime)
	if err != nil {
		return err
	}
	return installDockerLinuxForDistribution(runtime, instance, changed, composeOnly, distribution, FetchText)
}
func installDockerLinuxForDistribution(runtime *core.Runtime, instance core.Instance, changed func(), composeOnly bool, distribution core.LinuxDistribution, fetch func(context.Context, string) (string, error)) error {
	runtime.Ensure()
	if _, err := validateDockerDistribution(distribution, runtime.Arch); err != nil {
		return err
	}
	sudo := os.Geteuid() != 0
	run := func(command string, args []string, elevated bool) (string, error) {
		actual := command
		if elevated && sudo {
			actual = "sudo"
			args = append([]string{command}, args...)
		}
		result, err := runtime.Run(actual, args...)
		if err != nil || result.Code != 0 {
			return "", fmt.Errorf("%s failed: %s", command, platformFailure(result, err))
		}
		return result.Stdout, nil
	}
	write := func(value, destination string, download bool) error {
		content := value
		if download {
			var err error
			content, err = fetch(runtime.Context, value)
			if err != nil {
				return err
			}
		}
		changed()
		_, err := run("bash", []string{"-c", `set -e; temporary=$(mktemp "$2.genesis.XXXXXX"); trap 'rm -f "$temporary"' EXIT; printf '%s\n' "$1" >"$temporary"; chmod 0644 "$temporary"; mv -f "$temporary" "$2"`, "genesis", content, destination}, true)
		return err
	}
	base := "https://download.docker.com/linux/" + distribution.ID
	requested := optionString(instance.Options, "version")
	includeCompose := optionBool(instance.Options, "include_compose")
	if distribution.ID == "ubuntu" || distribution.ID == "debian" {
		output, err := run("dpkg", []string{"--print-architecture"}, false)
		if err != nil {
			return err
		}
		arch := strings.TrimSpace(output)
		if arch != "amd64" && arch != "arm64" {
			return fmt.Errorf("unsupported Docker APT architecture: %s", arch)
		}
		changed()
		if _, err := run("install", []string{"-m", "0755", "-d", "/etc/apt/keyrings"}, true); err != nil {
			return err
		}
		if err := write(base+"/gpg", "/etc/apt/keyrings/docker.asc", true); err != nil {
			return err
		}
		if _, err := run("chmod", []string{"a+r", "/etc/apt/keyrings/docker.asc"}, true); err != nil {
			return err
		}
		if err := write(fmt.Sprintf("deb [arch=%s signed-by=/etc/apt/keyrings/docker.asc] %s %s stable", arch, base, distribution.Codename), "/etc/apt/sources.list.d/docker.list", false); err != nil {
			return err
		}
		if _, err := run("apt-get", []string{"update"}, true); err != nil {
			return err
		}
		packages := []string{}
		if !composeOnly {
			var available []DockerPackageVersion
			for _, name := range []string{"docker-ce", "docker-ce-cli"} {
				output, err := run("apt-cache", []string{"madison", name}, false)
				if err != nil {
					return err
				}
				for _, line := range strings.Split(output, "\n") {
					fields := strings.Split(line, "|")
					if len(fields) < 2 {
						continue
					}
					fieldName, value := strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])
					if fieldName == name && dockerPackageValuePattern.MatchString(value) {
						available = append(available, DockerPackageVersion{Name: name, Value: value})
					}
				}
			}
			packages, err = SelectDockerPackages(available, requested, "=")
			if err != nil {
				return err
			}
			packages = append(packages, "containerd.io", "docker-buildx-plugin")
		}
		if includeCompose {
			packages = append(packages, "docker-compose-plugin")
		}
		if len(packages) == 0 {
			return fmt.Errorf("Docker Compose-only installation requires include_compose: true")
		}
		args := []string{"install", "-y", "--no-install-recommends"}
		if requested != "latest" {
			args = append(args, "--allow-downgrades")
		}
		if _, err := run("apt-get", append(args, packages...), true); err != nil {
			return err
		}
	} else {
		if err := write(base+"/docker-ce.repo", "/etc/yum.repos.d/docker-ce.repo", true); err != nil {
			return err
		}
		arch := "x86_64"
		if runtime.Arch == "arm64" {
			arch = "aarch64"
		}
		packages := []string{}
		if !composeOnly {
			output, err := run("dnf", []string{"repoquery", "--available", "--queryformat", "%{name}|%{evr}|%{arch}\\n", "docker-ce", "docker-ce-cli"}, true)
			if err != nil {
				return err
			}
			var available []DockerPackageVersion
			for _, line := range strings.Split(output, "\n") {
				fields := strings.Split(strings.TrimSpace(line), "|")
				if len(fields) != 3 {
					continue
				}
				if (fields[0] == "docker-ce" || fields[0] == "docker-ce-cli") && fields[2] == arch && dockerPackageValuePattern.MatchString(fields[1]) {
					available = append(available, DockerPackageVersion{Name: fields[0], Value: fields[1]})
				}
			}
			packages, err = SelectDockerPackages(available, requested, "-")
			if err != nil {
				return err
			}
			packages = append(packages, "containerd.io", "docker-buildx-plugin")
		}
		if includeCompose {
			packages = append(packages, "docker-compose-plugin")
		}
		if len(packages) == 0 {
			return fmt.Errorf("Docker Compose-only installation requires include_compose: true")
		}
		if _, err := run("dnf", append([]string{"install", "-y", "--setopt=install_weak_deps=False"}, packages...), true); err != nil {
			return err
		}
	}
	if !composeOnly {
		_, err := run("systemctl", []string{"enable", "--now", "docker"}, true)
		return err
	}
	return nil
}
