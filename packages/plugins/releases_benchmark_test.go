package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// Approximate the public manifest's release/file counts and metadata without network timing.
func goManifestFixture(b *testing.B) []byte {
	b.Helper()
	type file struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int64  `json:"size"`
		Kind     string `json:"kind"`
	}
	type release struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
		Files   []file `json:"files"`
	}
	releases := make([]release, 369)
	for i := range releases {
		version := fmt.Sprintf("go1.%d.%d", 27-i/20, 3+i%20)
		files := make([]file, 24)
		for j := range files {
			system, arch := "linux", fmt.Sprintf("arch%d", j)
			if j == 0 {
				system, arch = "darwin", "arm64"
			}
			files[j] = file{fmt.Sprintf("%s.%s-%s.tar.gz", version, system, arch), system, arch, version, testHash, 74000000, "archive"}
		}
		releases[i] = release{version, true, files}
	}
	data, err := json.MarshalIndent(releases, "", " ")
	if err != nil {
		b.Fatal(err)
	}
	return data
}

type benchmarkGoRelease struct {
	Files []struct{ Filename, SHA256 string }
}

func benchmarkBufferedChecksum(ctx context.Context, filename string) (string, error) {
	var releases []benchmarkGoRelease
	if err := FetchJSON(ctx, "https://go.dev/dl/?mode=json&include=all", &releases); err != nil {
		return "", err
	}
	return benchmarkSelectChecksum(releases, filename), nil
}
func benchmarkDecodedChecksum(parent context.Context, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	body, err := fetchBody(ctx, "https://go.dev/dl/?mode=json&include=all")
	if err != nil {
		return "", err
	}
	defer body.Close()
	limited := &io.LimitedReader{R: contextReader{ctx, body}, N: maxReleaseMetadataBytes + 1}
	decoder := json.NewDecoder(limited)
	var releases []benchmarkGoRelease
	if err = decoder.Decode(&releases); err != nil {
		return "", err
	}
	if _, err = decoder.Token(); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing JSON")
		}
		return "", err
	}
	if limited.N == 0 {
		return "", errors.New("metadata limit exceeded")
	}
	return benchmarkSelectChecksum(releases, filename), nil
}
func benchmarkSelectChecksum(releases []benchmarkGoRelease, filename string) string {
	for _, release := range releases {
		for _, file := range release.Files {
			if file.Filename == filename && checksumPattern.MatchString(file.SHA256) {
				return file.SHA256
			}
		}
	}
	return ""
}

func BenchmarkGoReleaseManifest(b *testing.B) {
	data := goManifestFixture(b)
	previous := HTTPClient
	HTTPClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(data)), Request: r}, nil
	})}
	b.Cleanup(func() { HTTPClient = previous })
	for _, variant := range []struct {
		name    string
		resolve func(context.Context, string) (string, error)
	}{{"buffered", benchmarkBufferedChecksum}, {"decoded", benchmarkDecodedChecksum}, {"streamed", goArchiveChecksum}} {
		b.Run(variant.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				checksum, err := variant.resolve(context.Background(), "go1.27.3.darwin-arm64.tar.gz")
				if err != nil || checksum != testHash {
					b.Fatal(checksum, err)
				}
			}
		})
	}
}
