package core

import (
	"context"
	"strings"
	"testing"
)

func TestLinuxDistribution(t *testing.T) {
	for _, test := range []struct {
		input, id, codename string
		like                []string
	}{{"ID=debian\nVERSION_CODENAME=bookworm", "debian", "bookworm", nil}, {"ID=\"ubuntu\"\nID_LIKE='debian'\nUBUNTU_CODENAME=noble", "ubuntu", "noble", []string{"debian"}}, {"ID=derived\nID_LIKE=\"fedora rhel\"", "derived", "", []string{"fedora", "rhel"}}} {
		got, err := ParseLinuxDistribution([]byte(test.input))
		if err != nil || got.ID != test.id || got.Codename != test.codename || strings.Join(got.Like, " ") != strings.Join(test.like, " ") {
			t.Fatal(got, err)
		}
	}
	for _, input := range []string{"", "ID=\"$(evil)\"", "ID=bad id", "ID=debian\nID_LIKE='evil;exec'"} {
		if _, err := ParseLinuxDistribution([]byte(input)); err == nil {
			t.Fatal("unsafe distro accepted", input)
		}
	}
}
func TestPackageManagerSelection(t *testing.T) {
	for id, expected := range map[string]string{"debian": "apt", "ubuntu": "apt", "fedora": "dnf", "rhel": "dnf", "centos": "dnf", "rocky": "dnf", "almalinux": "dnf", "arch": "pacman", "alpine": "apk"} {
		got, err := PackageManager("linux", LinuxDistribution{ID: id})
		if err != nil || got != expected {
			t.Fatal(id, got, err)
		}
	}
	got, err := PackageManager("linux", LinuxDistribution{ID: "derived", Like: []string{"missing", "debian"}})
	if err != nil || got != "apt" {
		t.Fatal(got, err)
	}
	for _, platform := range []string{"macos", "windows"} {
		if _, err := PackageManager(platform, LinuxDistribution{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := PackageManager("linux", LinuxDistribution{ID: "missing"}); err == nil {
		t.Fatal("unknown manager accepted")
	}
	if _, err := PackageManager("freebsd", LinuxDistribution{}); err == nil {
		t.Fatal("unknown platform accepted")
	}
}
func TestPackageTaskSafety(t *testing.T) {
	r := &Runtime{Platform: "windows"}
	r.Ensure()
	task, err := CreatePackageManagerUpdateTask(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Runner = func(context.Context, string, []string, CommandOptions) (CommandResult, error) {
		t.Fatal("choco update mutates host")
		return CommandResult{}, nil
	}
	if result, err := task.Execute(context.Background()); err != nil || !result.OK {
		t.Fatal(result, err)
	}
	for _, name := range []string{"--flag", "name;bad", "name space", ""} {
		if _, err := CreatePackageInstallTask(r, name); err == nil {
			t.Fatal("unsafe package accepted", name)
		}
	}
	r.Platform = "macos"
	calls := []string{}
	r.Runner = func(_ context.Context, command string, args []string, _ CommandOptions) (CommandResult, error) {
		calls = append(calls, command+":"+args[0])
		return CommandResult{}, nil
	}
	if err := RegisterPackageTasks(r, "git", "git", "curl"); err != nil {
		t.Fatal(err)
	}
	results, err := r.Tasks.ExecuteAll(context.Background())
	if err != nil || len(results) != 3 || len(calls) != 3 || calls[0] != "brew:update" {
		t.Fatal(results, calls, err)
	}
}
