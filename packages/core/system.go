package core

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type LinuxDistribution struct {
	ID       string
	Like     []string
	Codename string
}

var distroToken = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func ParseLinuxDistribution(data []byte) (LinuxDistribution, error) {
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	d := LinuxDistribution{ID: values["ID"], Like: strings.Fields(values["ID_LIKE"]), Codename: values["VERSION_CODENAME"]}
	if d.Codename == "" {
		d.Codename = values["UBUNTU_CODENAME"]
	}
	if !distroToken.MatchString(d.ID) {
		return d, fmt.Errorf("invalid or missing Linux distribution ID")
	}
	for _, id := range d.Like {
		if !distroToken.MatchString(id) {
			return d, fmt.Errorf("invalid ID_LIKE token %q", id)
		}
	}
	return d, nil
}
func GetLinuxDistribution() (LinuxDistribution, error) {
	data, err := os.ReadFile("/etc/os-release")
	if os.IsNotExist(err) {
		data, err = os.ReadFile("/usr/lib/os-release")
	}
	if err != nil {
		return LinuxDistribution{}, err
	}
	return ParseLinuxDistribution(data)
}
func PackageManager(platform string, distro LinuxDistribution) (string, error) {
	switch platform {
	case "macos":
		return "brew", nil
	case "windows":
		return "choco", nil
	case "linux":
		for _, id := range append([]string{distro.ID}, distro.Like...) {
			switch id {
			case "debian", "ubuntu":
				return "apt", nil
			case "fedora", "rhel", "centos", "rocky", "almalinux":
				return "dnf", nil
			case "arch":
				return "pacman", nil
			case "alpine":
				return "apk", nil
			}
		}
		return "", fmt.Errorf("unsupported Linux package manager for %s", distro.ID)
	default:
		return "", fmt.Errorf("unsupported platform %s", platform)
	}
}
func runtimePackageManager(r *Runtime) (string, error) {
	platform := r.Platform
	if platform == "" {
		platform = GetPlatform()
	}
	var d LinuxDistribution
	var err error
	if platform == "linux" {
		d, err = GetLinuxDistribution()
		if err != nil {
			return "", err
		}
	}
	return PackageManager(platform, d)
}
func PackageManagerTaskID(platform, manager, operation, packageName string) string {
	id := platform + ":package-manager:" + manager + "-" + operation
	if packageName != "" {
		id += ":" + packageName
	}
	return id
}

var packageToken = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9@+_.:/-]*$`)
var linuxPackages = map[string]map[string][]string{
	"build-essential":      {"dnf": {"gcc", "gcc-c++", "make"}, "pacman": {"base-devel"}, "apk": {"build-base"}},
	"libssl-dev":           {"dnf": {"openssl-devel"}, "pacman": {"openssl"}, "apk": {"openssl-dev"}},
	"libcurl4-openssl-dev": {"dnf": {"libcurl-devel"}, "pacman": {"curl"}, "apk": {"curl-dev"}},
	"zlib1g-dev":           {"dnf": {"zlib-devel"}, "pacman": {"zlib"}, "apk": {"zlib-dev"}},
	"libexpat1-dev":        {"dnf": {"expat-devel"}, "pacman": {"expat"}, "apk": {"expat-dev"}},
	"gnupg":                {"dnf": {"gnupg2"}},
}

func packageCommandTask(r *Runtime, id, description, command string, args, dependencies []string) Task {
	return Task{ID: id, Description: description, Priority: 50, DependsOn: dependencies, Execute: func(ctx context.Context) (TaskResult, error) {
		actualCommand, actualArgs := command, args
		if r.Platform == "linux" && os.Geteuid() != 0 {
			actualArgs = append([]string{command}, args...)
			actualCommand = "sudo"
		}
		copy := *r
		copy.Context = ctx
		result, err := copy.Run(actualCommand, actualArgs...)
		if err != nil {
			return TaskResult{}, err
		}
		if result.Code != 0 {
			return TaskResult{Error: result.Stderr, Details: description}, nil
		}
		return TaskResult{OK: true, Details: description}, nil
	}}
}
func CreatePackageManagerUpdateTask(r *Runtime) (Task, error) {
	manager, err := runtimePackageManager(r)
	if err != nil {
		return Task{}, err
	}
	id := PackageManagerTaskID(r.Platform, manager, "update", "")
	if manager == "pacman" || manager == "choco" {
		return Task{ID: id, Description: "Use existing " + manager + " package index", Priority: 100, Execute: func(context.Context) (TaskResult, error) {
			return TaskResult{OK: true, Details: "No host-wide upgrade; package index maintained externally"}, nil
		}}, nil
	}
	command := manager
	if manager == "apt" {
		command = "apt-get"
	}
	args := []string{"update"}
	if manager == "dnf" {
		args = []string{"makecache"}
	}
	task := packageCommandTask(r, id, "Update "+manager+" package index", command, args, nil)
	task.Priority = 100
	return task, nil
}
func CreatePackageInstallTask(r *Runtime, name string) (Task, error) {
	if !packageToken.MatchString(name) {
		return Task{}, fmt.Errorf("invalid system package name: %s", name)
	}
	manager, err := runtimePackageManager(r)
	if err != nil {
		return Task{}, err
	}
	packages := linuxPackages[name][manager]
	if len(packages) == 0 {
		packages = []string{name}
	}
	var args []string
	switch manager {
	case "pacman":
		args = append([]string{"-S", "--needed", "--noconfirm"}, packages...)
	case "apk":
		args = append([]string{"add"}, packages...)
	case "brew":
		args = append([]string{"install"}, packages...)
	case "choco":
		args = append(append([]string{"install"}, packages...), "-y")
	default:
		args = append([]string{"install", "-y"}, packages...)
	}
	command := manager
	if command == "apt" {
		command = "apt-get"
	}
	return packageCommandTask(r, PackageManagerTaskID(r.Platform, manager, "install", name), "Install "+strings.Join(packages, ", ")+" via "+manager, command, args, []string{PackageManagerTaskID(r.Platform, manager, "update", "")}), nil
}
func RegisterPackageTasks(r *Runtime, packages ...string) error {
	r.Ensure()
	update, err := CreatePackageManagerUpdateTask(r)
	if err != nil {
		return err
	}
	tasks := []Task{update}
	for _, name := range packages {
		task, err := CreatePackageInstallTask(r, name)
		if err != nil {
			return err
		}
		tasks = append(tasks, task)
	}
	for _, task := range tasks {
		if err := r.Tasks.Register(task); err != nil {
			return err
		}
	}
	return nil
}
func CreateCommandCheckTask(r *Runtime, name string) Task {
	return Task{ID: r.Platform + ":command-check:" + name, Description: "Check " + name, Priority: 200, Execute: func(ctx context.Context) (TaskResult, error) {
		command := "which"
		if r.Platform == "windows" {
			command = "where"
		}
		copy := *r
		copy.Context = ctx
		result, err := copy.Run(command, name)
		return TaskResult{OK: err == nil && result.Code == 0, Details: result.Stdout}, err
	}}
}
func CreateCustomTask(platform, id, description string, executor func(context.Context) (TaskResult, error), priority int, dependencies ...string) Task {
	return Task{ID: platform + ":custom:" + id, Description: description, Execute: executor, Priority: priority, DependsOn: dependencies}
}
