package plugins

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ossl-dev/genesis/packages/core"
)

func tarFixture(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for name, content := range entries {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
func zipFixture(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range entries {
		header := &zip.FileHeader{Name: name}
		header.SetMode(0755)
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
func fixtureRelease(t *testing.T, data []byte, format string) (ArchiveRelease, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	t.Cleanup(server.Close)
	return ArchiveRelease{URL: server.URL, SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Format: format}, server
}
func testRuntime(t *testing.T) *core.Runtime {
	t.Helper()
	r := &core.Runtime{Context: context.Background(), Cwd: t.TempDir(), Env: core.NewEnvironment(map[string]string{"HOME": t.TempDir(), "PATH": os.Getenv("PATH")})}
	r.Ensure()
	return r
}
func TestArchiveInstallAndRollback(t *testing.T) {
	for _, format := range []string{"tar.gz", "zip"} {
		t.Run(format, func(t *testing.T) {
			data := tarFixture(t, map[string]string{"runtime/version": "new"})
			if format == "zip" {
				data = zipFixture(t, map[string]string{"runtime/version": "new"})
			}
			release, _ := fixtureRelease(t, data, format)
			for _, fail := range []string{"", "stage", "promoted", "selector", "checksum"} {
				t.Run(fail, func(t *testing.T) {
					r := testRuntime(t)
					destination := filepath.Join(r.Cwd, "installation with spaces")
					if err := os.Mkdir(destination, 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(destination, "version"), []byte("old"), 0600); err != nil {
						t.Fatal(err)
					}
					verifyCount := 0
					currentRelease := release
					if fail == "checksum" {
						currentRelease.SHA256 = strings.Repeat("0", 64)
					}
					selector := SingleDirectory
					if fail == "selector" {
						selector = func(string) (string, error) { return destination, nil }
					}
					err := InstallArchive(r, destination, "version", currentRelease, selector, func(root string) error {
						verifyCount++
						if fail == "stage" || fail == "promoted" && root == destination {
							return errors.New("verification failed")
						}
						data, e := os.ReadFile(filepath.Join(root, "version"))
						if e != nil {
							return e
						}
						if string(data) != "new" {
							return errors.New("wrong runtime")
						}
						return nil
					})
					if fail == "" && err != nil {
						t.Fatal(err)
					}
					if fail != "" && err == nil {
						t.Fatal("expected failure")
					}
					want := "new"
					if fail != "" {
						want = "old"
					}
					installed, err := os.ReadFile(filepath.Join(destination, "version"))
					if err != nil || string(installed) != want {
						t.Fatalf("previous runtime not preserved: %q %v", installed, err)
					}
					if fail == "" && verifyCount != 2 {
						t.Fatalf("verify count %d", verifyCount)
					}
					files, _ := os.ReadDir(r.Cwd)
					if len(files) != 1 {
						t.Fatalf("staging or lock leaked: %v", files)
					}
				})
			}
		})
	}
}
func TestArchiveGuardrails(t *testing.T) {
	release, _ := fixtureRelease(t, tarFixture(t, map[string]string{"runtime/version": "new"}), "tar.gz")
	for _, kind := range []string{"lock", "occupied", "symlink", "root"} {
		t.Run(kind, func(t *testing.T) {
			r := testRuntime(t)
			destination := filepath.Join(r.Cwd, "installed")
			switch kind {
			case "lock":
				os.Mkdir(destination+".genesis-lock", 0700)
			case "occupied":
				os.Mkdir(destination, 0755)
				os.WriteFile(filepath.Join(destination, "unrelated"), []byte("x"), 0600)
			case "symlink":
				if runtime.GOOS == "windows" {
					t.Skip("symlink privilege needed")
				}
				target := t.TempDir()
				if err := os.Symlink(target, destination); err != nil {
					t.Fatal(err)
				}
			case "root":
				destination = filepath.VolumeName(r.Cwd) + string(filepath.Separator)
			}
			if err := InstallArchive(r, destination, "bin/tool", release, SingleDirectory, func(string) error { return nil }); err == nil {
				t.Fatal("expected refusal")
			}
		})
	}
}
func TestArchiveUnsafePaths(t *testing.T) {
	for _, name := range []string{"../escaped", "/absolute", "C:\\windows\\bad", "runtime/../../escape", "runtime\\..\\bad"} {
		for _, format := range []string{"tar.gz", "zip"} {
			t.Run(format+name, func(t *testing.T) {
				r := testRuntime(t)
				data := tarFixture(t, map[string]string{name: "bad"})
				if format == "zip" {
					data = zipFixture(t, map[string]string{name: "bad"})
				}
				release, _ := fixtureRelease(t, data, format)
				err := InstallArchive(r, filepath.Join(r.Cwd, "installed"), "", release, func(root string) (string, error) { return root, nil }, func(string) error { return nil })
				if err == nil {
					t.Fatal("unsafe archive accepted")
				}
			})
		}
	}
}
func TestArchiveSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires symlinks")
	}
	for _, target := range []string{"file", "../../escape"} {
		t.Run(target, func(t *testing.T) {
			var b bytes.Buffer
			gz := gzip.NewWriter(&b)
			writer := tar.NewWriter(gz)
			writer.WriteHeader(&tar.Header{Name: "runtime/file", Mode: 0755, Size: 2})
			writer.Write([]byte("ok"))
			writer.WriteHeader(&tar.Header{Name: "runtime/link", Typeflag: tar.TypeSymlink, Linkname: target})
			writer.Close()
			gz.Close()
			release, _ := fixtureRelease(t, b.Bytes(), "tar.gz")
			r := testRuntime(t)
			destination := filepath.Join(r.Cwd, "installed")
			err := InstallArchive(r, destination, "", release, SingleDirectory, func(root string) error {
				data, e := os.ReadFile(filepath.Join(root, "link"))
				if e != nil {
					return e
				}
				if string(data) != "ok" {
					return errors.New("wrong link")
				}
				return nil
			})
			if target == "file" && err != nil {
				t.Fatal(err)
			}
			if target != "file" && err == nil {
				t.Fatal("escaping symlink accepted")
			}
		})
	}
}
func TestFetchErrorsAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	if _, err := FetchText(context.Background(), server.URL); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("bad status: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FetchText(ctx, server.URL); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestInstallationRecoveryRetainsBackup(t *testing.T) {
	r := testRuntime(t)
	destination := filepath.Join(r.Cwd, "installed")
	os.Mkdir(destination, 0755)
	os.WriteFile(filepath.Join(destination, "version"), []byte("old"), 0600)
	err := InstallDirectory(r, destination, "version", func(stage string) (string, error) {
		selected := filepath.Join(stage, "new")
		os.Mkdir(selected, 0755)
		os.WriteFile(filepath.Join(selected, "version"), []byte("new"), 0600)
		os.Mkdir(filepath.Join(stage, "failed"), 0755)
		os.WriteFile(filepath.Join(stage, "failed", "occupied"), []byte("x"), 0600)
		return selected, nil
	}, func(root string) error {
		if root == destination {
			return errors.New("promoted runtime fails")
		}
		return nil
	})
	var recovery *InstallationRecoveryError
	if !errors.As(err, &recovery) {
		t.Fatalf("expected recovery failure: %v", err)
	}
	stages, _ := filepath.Glob(filepath.Join(r.Cwd, ".genesis-*"))
	if len(stages) != 1 {
		t.Fatal("backup stage missing", stages)
	}
	data, err := os.ReadFile(filepath.Join(stages[0], "previous", "version"))
	if err != nil || string(data) != "old" {
		t.Fatal("backup lost", string(data), err)
	}
	if exists(destination + ".genesis-lock") {
		t.Fatal("lock leaked")
	}
}
func TestCorruptArchiveKeepsOldInstall(t *testing.T) {
	r := testRuntime(t)
	destination := filepath.Join(r.Cwd, "installed")
	os.Mkdir(destination, 0755)
	os.WriteFile(filepath.Join(destination, "version"), []byte("old"), 0600)
	release, _ := fixtureRelease(t, []byte("corrupt archive"), "tar.gz")
	if err := InstallArchive(r, destination, "version", release, SingleDirectory, func(string) error { return nil }); err == nil {
		t.Fatal("corrupt archive accepted")
	}
	data, err := os.ReadFile(filepath.Join(destination, "version"))
	if err != nil || string(data) != "old" {
		t.Fatal("old runtime lost")
	}
}
func TestArchiveEntryLimits(t *testing.T) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	writer.WriteHeader(&tar.Header{Name: "runtime/oversized", Mode: 0755, Size: maxArchiveBytes + 1})
	writer.Close()
	gz.Close()
	r := testRuntime(t)
	release, _ := fixtureRelease(t, buffer.Bytes(), "tar.gz")
	if err := InstallArchive(r, filepath.Join(r.Cwd, "installed"), "", release, SingleDirectory, func(string) error { return nil }); err == nil || !strings.Contains(err.Error(), "2 GiB") {
		t.Fatalf("oversized entry accepted: %v", err)
	}
}
func BenchmarkMatchesVersion(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		if !MatchesVersion("22.18.0", "22.18") {
			b.Fatal("mismatch")
		}
	}
}
