package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

type Artifact struct {
	Path       string `json:"path"`
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Compressed bool   `json:"compressed"`
	URL        string `json:"url,omitempty"`
}
type EnvironmentArtifacts struct {
	Binaries []Artifact `json:"binaries"`
	Configs  []Artifact `json:"configs"`
	Caches   []Artifact `json:"caches"`
	Logs     []Artifact `json:"logs"`
}
type EnvironmentMetadata struct {
	Platform     string            `json:"platform"`
	Arch         string            `json:"arch"`
	Versions     map[string]string `json:"versions"`
	Dependencies []string          `json:"dependencies"`
	SetupTime    int64             `json:"setupTime"`
	LastUsed     time.Time         `json:"lastUsed"`
	UsageCount   int               `json:"usageCount"`
}
type CachedEnvironment struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Config      Config               `json:"config"`
	Metadata    EnvironmentMetadata  `json:"metadata"`
	Artifacts   EnvironmentArtifacts `json:"artifacts"`
	Checksum    string               `json:"checksum"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
	Size        int64                `json:"size"`
	Public      bool                 `json:"public"`
	Tags        []string             `json:"tags"`
}
type FileState struct {
	Path, Hash  string
	Size        int64
	Modified    time.Time
	Permissions string
}
type FilesystemState struct {
	Files       []FileState
	Directories []string
	Permissions map[string]string
}
type ProcessState struct {
	PID     int
	Command string
	Args    []string
	Status  string
	CPU     float64
	Memory  uint64
}
type PortState struct {
	Port             int
	Protocol, Status string
	Process          int
}
type NetworkAddress struct {
	Address string
	Port    int
}
type ConnectionState struct {
	Local, Remote NetworkAddress
	Status        string
}
type NetworkState struct {
	Ports       []PortState
	Connections []ConnectionState
}
type SnapshotState struct {
	Filesystem  FilesystemState
	Environment map[string]string
	Processes   []ProcessState
	Network     NetworkState
}
type EnvironmentSnapshot struct {
	ID, EnvironmentID, Name, Description string
	Config                               Config
	State                                SnapshotState
	CreatedAt                            time.Time
	Restorable                           bool
}
type CacheConfig struct {
	MaxSize                       int64
	MaxAge                        time.Duration
	CompressionLevel              int
	EncryptionEnabled, RemoteSync bool
	RemoteURL                     string
}
type CacheStats struct {
	TotalEnvironments        int
	TotalSize                int64
	TotalSnapshots           int
	HitRate                  float64
	OldestEntry, NewestEntry *time.Time
}
type EnvironmentCacheManager struct {
	mu        sync.Mutex
	config    CacheConfig
	cache     map[string]CachedEnvironment
	snapshots map[string]EnvironmentSnapshot
}

func NewEnvironmentCacheManager(config CacheConfig) (*EnvironmentCacheManager, error) {
	if config.EncryptionEnabled {
		return nil, fmt.Errorf("cache encryption is not implemented")
	}
	if config.RemoteSync {
		return nil, fmt.Errorf("remote cache synchronization is not implemented")
	}
	if config.MaxSize == 0 {
		config.MaxSize = 1 << 30
	}
	if config.MaxAge == 0 {
		config.MaxAge = 7 * 24 * time.Hour
	}
	if config.MaxSize < 0 || config.MaxAge < 0 || config.CompressionLevel < 0 || config.CompressionLevel > 9 {
		return nil, fmt.Errorf("invalid cache limits")
	}
	return &EnvironmentCacheManager{config: config, cache: map[string]CachedEnvironment{}, snapshots: map[string]EnvironmentSnapshot{}}, nil
}
func cacheID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func configChecksum(config Config) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
func cloneCached(e CachedEnvironment) (CachedEnvironment, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return CachedEnvironment{}, err
	}
	var result CachedEnvironment
	err = json.Unmarshal(data, &result)
	return result, err
}
func (m *EnvironmentCacheManager) CacheEnvironment(name string, config Config, description string, tags []string) (CachedEnvironment, error) {
	if err := ValidateConfig(config); err != nil {
		return CachedEnvironment{}, err
	}
	id, err := cacheID()
	if err != nil {
		return CachedEnvironment{}, err
	}
	checksum, err := configChecksum(config)
	if err != nil {
		return CachedEnvironment{}, err
	}
	now := time.Now()
	versions := map[string]string{}
	if result, detectErr := RunCommand(context.Background(), "node", []string{"--version"}, CommandOptions{Timeout: 5 * time.Second}); detectErr == nil && result.Code == 0 {
		versions["node"] = result.Stdout
	}
	entry := CachedEnvironment{ID: id, Name: name, Description: description, Config: config, Metadata: EnvironmentMetadata{Platform: GetPlatform(), Arch: runtime.GOARCH, Versions: versions, Dependencies: []string{}, LastUsed: now}, Checksum: checksum, CreatedAt: now, UpdatedAt: now, Tags: append([]string{}, tags...)}
	copy, err := cloneCached(entry)
	if err != nil {
		return entry, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache[id] = copy
	return entry, nil
}
func (m *EnvironmentCacheManager) RestoreEnvironment(id string) (CachedEnvironment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cache[id]; !ok {
		return CachedEnvironment{}, fmt.Errorf("environment %s not found in cache", id)
	}
	return CachedEnvironment{}, fmt.Errorf("artifact restoration is not implemented; cached metadata is not an environment backup")
}
func (m *EnvironmentCacheManager) CreateSnapshot(environmentID, name, description string) (EnvironmentSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.cache[environmentID]
	if !ok {
		return EnvironmentSnapshot{}, fmt.Errorf("environment %s not found", environmentID)
	}
	id, err := cacheID()
	if err != nil {
		return EnvironmentSnapshot{}, err
	}
	home, err := ResolveGenesisHome()
	if err != nil {
		return EnvironmentSnapshot{}, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return EnvironmentSnapshot{}, err
	}
	snapshot := EnvironmentSnapshot{ID: id, EnvironmentID: environmentID, Name: name, Description: description, Config: entry.Config, CreatedAt: time.Now(), Restorable: false, State: SnapshotState{Filesystem: FilesystemState{Files: []FileState{}, Directories: []string{home, cwd}, Permissions: map[string]string{}}, Environment: NewEnvironment(nil).Snapshot(), Processes: []ProcessState{}, Network: NetworkState{Ports: []PortState{}, Connections: []ConnectionState{}}}}
	stored, err := cloneSnapshot(snapshot)
	if err != nil {
		return snapshot, err
	}
	m.snapshots[id] = stored
	return cloneSnapshot(snapshot)
}
func (m *EnvironmentCacheManager) RestoreSnapshot(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.snapshots[id]; !ok {
		return fmt.Errorf("snapshot %s not found", id)
	}
	return fmt.Errorf("filesystem and network snapshot restoration is not implemented")
}
func (m *EnvironmentCacheManager) SwitchEnvironment(id string) error {
	_, err := m.RestoreEnvironment(id)
	return err
}
func (m *EnvironmentCacheManager) ExportEnvironment(id, format string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.cache[id]
	if !ok {
		return "", fmt.Errorf("environment %s not found", id)
	}
	if format == "" || format == "json" {
		data, err := json.MarshalIndent(entry, "", "  ")
		return string(data), err
	}
	return "", fmt.Errorf("%s export is not implemented", format)
}
func (m *EnvironmentCacheManager) ImportEnvironment(data, format string) (CachedEnvironment, error) {
	if format != "" && format != "json" {
		return CachedEnvironment{}, fmt.Errorf("%s import is not implemented", format)
	}
	if len(data) > 8<<20 {
		return CachedEnvironment{}, fmt.Errorf("cache metadata exceeds 8 MiB")
	}
	var e CachedEnvironment
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return e, err
	}
	if e.ID == "" || e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() || e.Size < 0 {
		return e, fmt.Errorf("invalid environment metadata")
	}
	if err := ValidateConfig(e.Config); err != nil {
		return e, err
	}
	sum, err := configChecksum(e.Config)
	if err != nil {
		return e, err
	}
	if sum != e.Checksum {
		return e, fmt.Errorf("environment config checksum mismatch")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	copy, err := cloneCached(e)
	if err != nil {
		return e, err
	}
	m.cache[e.ID] = copy
	return e, nil
}
func (m *EnvironmentCacheManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	entries := []CachedEnvironment{}
	var size int64
	for id, e := range m.cache {
		if now.Sub(e.UpdatedAt) > m.config.MaxAge {
			delete(m.cache, id)
		} else {
			entries = append(entries, e)
			size += e.Size
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].UpdatedAt.Before(entries[j].UpdatedAt) })
	for _, e := range entries {
		if size <= m.config.MaxSize {
			break
		}
		delete(m.cache, e.ID)
		size -= e.Size
	}
}
func (m *EnvironmentCacheManager) GetStats() CacheStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats := CacheStats{TotalEnvironments: len(m.cache), TotalSnapshots: len(m.snapshots)}
	for _, e := range m.cache {
		stats.TotalSize += e.Size
		if stats.OldestEntry == nil || e.UpdatedAt.Before(*stats.OldestEntry) {
			time := e.UpdatedAt
			stats.OldestEntry = &time
		}
		if stats.NewestEntry == nil || e.UpdatedAt.After(*stats.NewestEntry) {
			time := e.UpdatedAt
			stats.NewestEntry = &time
		}
	}
	return stats
}

// CompressMetadata is a bounded transport helper; it never represents captured binaries.
func CompressMetadata(e CachedEnvironment, level int) ([]byte, error) {
	var out bytes.Buffer
	writer, err := gzip.NewWriterLevel(&out, level)
	if err != nil {
		return nil, err
	}
	if err = json.NewEncoder(writer).Encode(e); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func DecompressMetadata(data []byte) (CachedEnvironment, error) {
	var e CachedEnvironment
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return e, err
	}
	defer reader.Close()
	decoded, err := io.ReadAll(io.LimitReader(reader, (8<<20)+1))
	if err != nil {
		return e, err
	}
	if len(decoded) > 8<<20 {
		return e, fmt.Errorf("decompressed metadata exceeds 8 MiB")
	}
	err = json.Unmarshal(decoded, &e)
	return e, err
}
func WritePrivateFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".genesis-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}

func cloneSnapshot(snapshot EnvironmentSnapshot) (EnvironmentSnapshot, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return EnvironmentSnapshot{}, err
	}
	var result EnvironmentSnapshot
	err = json.Unmarshal(data, &result)
	return result, err
}
