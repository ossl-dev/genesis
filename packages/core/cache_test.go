package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCacheMetadataRoundTrip(t *testing.T) {
	manager, err := NewEnvironmentCacheManager(CacheConfig{})
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Env: map[string]string{"KEY": "value"}}
	entry, err := manager.CacheEnvironment("test", config, "description", []string{"tag"})
	if err != nil {
		t.Fatal(err)
	}
	config.Env["KEY"] = "mutated"
	data, err := manager.ExportEnvironment(entry.ID, "json")
	if err != nil || !strings.Contains(data, "value") || strings.Contains(data, "mutated") {
		t.Fatal(data, err)
	}
	other, _ := NewEnvironmentCacheManager(CacheConfig{})
	imported, err := other.ImportEnvironment(data, "json")
	if err != nil || imported.Checksum != entry.Checksum {
		t.Fatal(imported, err)
	}
	for _, level := range []int{1, 6, 9} {
		compressed, err := CompressMetadata(imported, level)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := DecompressMetadata(compressed)
		if err != nil || restored.Checksum != entry.Checksum {
			t.Fatal(restored, err)
		}
	}
	snapshot, err := manager.CreateSnapshot(entry.ID, "before", "")
	if err != nil || snapshot.Restorable || snapshot.EnvironmentID != entry.ID {
		t.Fatal(snapshot, err)
	}
	if manager.RestoreSnapshot(snapshot.ID) == nil {
		t.Fatal("fake restore success")
	}
	if _, err := manager.RestoreEnvironment(entry.ID); err == nil {
		t.Fatal("fake artifact restore success")
	}
	if manager.SwitchEnvironment(entry.ID) == nil {
		t.Fatal("fake switch success")
	}
	stats := manager.GetStats()
	if stats.TotalEnvironments != 1 || stats.TotalSnapshots != 1 || stats.OldestEntry == nil {
		t.Fatal(stats)
	}
}
func TestCacheRejectsUnsupportedAndCorrupt(t *testing.T) {
	for _, config := range []CacheConfig{{EncryptionEnabled: true}, {RemoteSync: true}, {MaxSize: -1}, {MaxAge: -1}, {CompressionLevel: 10}} {
		if _, err := NewEnvironmentCacheManager(config); err == nil {
			t.Fatal("invalid cache accepted", config)
		}
	}
	manager, _ := NewEnvironmentCacheManager(CacheConfig{})
	entry, _ := manager.CacheEnvironment("test", Config{}, "", nil)
	for _, format := range []string{"tarball", "docker", "unknown"} {
		if _, err := manager.ExportEnvironment(entry.ID, format); err == nil {
			t.Fatal("unsupported export accepted")
		}
		if _, err := manager.ImportEnvironment("{}", format); err == nil {
			t.Fatal("unsupported import accepted")
		}
	}
	entry.Config.Env = map[string]string{"CHANGED": "yes"}
	data, _ := json.Marshal(entry)
	if _, err := manager.ImportEnvironment(string(data), "json"); err == nil {
		t.Fatal("bad checksum accepted")
	}
	for _, data := range []string{"{}", "invalid", strings.Repeat("x", (8<<20)+1)} {
		if _, err := manager.ImportEnvironment(data, "json"); err == nil {
			t.Fatal("corrupt input accepted")
		}
	}
	if _, err := DecompressMetadata([]byte("invalid")); err == nil {
		t.Fatal("corrupt compressed input accepted")
	}
	if _, err := manager.ExportEnvironment("missing", "json"); err == nil {
		t.Fatal("missing export succeeded")
	}
	if _, err := manager.RestoreEnvironment("missing"); err == nil {
		t.Fatal("missing restore succeeded")
	}
	if _, err := manager.CreateSnapshot("missing", "", ""); err == nil {
		t.Fatal("missing snapshot succeeded")
	}
	if manager.RestoreSnapshot("missing") == nil {
		t.Fatal("missing snapshot restore succeeded")
	}
}
func TestCacheCleanup(t *testing.T) {
	manager, _ := NewEnvironmentCacheManager(CacheConfig{MaxAge: time.Nanosecond})
	_, _ = manager.CacheEnvironment("test", Config{}, "", nil)
	time.Sleep(time.Millisecond)
	manager.Cleanup()
	if manager.GetStats().TotalEnvironments != 0 {
		t.Fatal("expired cache retained")
	}
	manager, _ = NewEnvironmentCacheManager(CacheConfig{MaxSize: 1})
	entry, _ := manager.CacheEnvironment("test", Config{}, "", nil)
	manager.cache[entry.ID] = func() CachedEnvironment { entry.Size = 2; return entry }()
	manager.Cleanup()
	if manager.GetStats().TotalEnvironments != 0 {
		t.Fatal("oversized cache retained")
	}
}
func TestWritePrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "auth.json")
	if err := WritePrivateFile(path, []byte("token")); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateFile(path, []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "replacement" {
		t.Fatal(string(data), err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal(info, err)
		}
	}
}
func TestSnapshotReturnedMapsIndependent(t *testing.T) {
	manager, _ := NewEnvironmentCacheManager(CacheConfig{})
	entry, _ := manager.CacheEnvironment("test", Config{Env: map[string]string{"KEY": "value"}}, "", nil)
	snapshot, err := manager.CreateSnapshot(entry.ID, "snapshot", "")
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Config.Env["KEY"] = "mutated"
	snapshot.State.Environment["KEY"] = "mutated"
	snapshot.State.Filesystem.Permissions["KEY"] = "mutated"
	if manager.cache[entry.ID].Config.Env["KEY"] != "value" || manager.snapshots[snapshot.ID].Config.Env["KEY"] != "value" || manager.snapshots[snapshot.ID].State.Environment["KEY"] == "mutated" {
		t.Fatal("snapshot aliases internal cache")
	}
}
func TestSnapshotGenesisHomeOverride(t *testing.T) {
	manager, _ := NewEnvironmentCacheManager(CacheConfig{})
	entry, err := manager.CacheEnvironment("test", Config{}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(t.TempDir(), "snapshot-home")
	t.Setenv("GENESIS_HOME", override)
	key := "HOME"
	if runtime.GOOS == "windows" {
		key = "USERPROFILE"
	}
	t.Setenv(key, "")
	snapshot, err := manager.CreateSnapshot(entry.ID, "snapshot", "")
	if err != nil || snapshot.State.Filesystem.Directories[0] != override {
		t.Fatal(snapshot, err)
	}
	if _, err := ResolveGenesisHome(); err == nil {
		t.Fatal("snapshot override changed native home resolution")
	}
	t.Setenv("GENESIS_HOME", "relative")
	if _, err := manager.CreateSnapshot(entry.ID, "snapshot", ""); err == nil {
		t.Fatal("relative snapshot home accepted")
	}
}
