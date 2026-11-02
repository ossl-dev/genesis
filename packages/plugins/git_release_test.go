package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type platformHTTPTransport func(*http.Request) (*http.Response, error)

func (transport platformHTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}
func platformHTTP(t *testing.T, respond func(*http.Request) ([]byte, error)) {
	t.Helper()
	previous := HTTPClient
	HTTPClient = &http.Client{Transport: platformHTTPTransport(func(request *http.Request) (*http.Response, error) {
		body, err := respond(request)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header), Request: request}, nil
	})}
	t.Cleanup(func() { HTTPClient = previous })
}
func TestGitSourceReleaseStableSelection(t *testing.T) {
	hash := strings.Repeat("a", 64)
	sums := fmt.Sprintf("%s  git-2.9.9.tar.gz\n%s  git-2.53.1.tar.gz\n%s  git-2.53.0.tar.gz\n%s  git-2.54.0.rc0.tar.gz\nbad git-2.99.0.tar.gz", hash, hash, hash, hash)
	for _, test := range []struct{ requested, expected string }{{"latest", "2.53.1"}, {"2.53.0", "2.53.0"}, {"2.53", "2.53.1"}} {
		release, err := selectGitSourceRelease(sums, test.requested, "https://example.com/git")
		if err != nil || release.Version != test.expected || release.SHA256 != hash || release.URL != "https://example.com/git/git-"+test.expected+".tar.gz" {
			t.Fatalf("%+v %v", release, err)
		}
	}
	if _, err := selectGitSourceRelease(sums, "2.5", "https://example.com"); err == nil {
		t.Fatal("matched wrong minor")
	}
	if _, err := selectGitSourceRelease("bad  git-2.53.0.tar.gz", "latest", "https://example.com"); err == nil {
		t.Fatal("malformed checksum accepted")
	}
	platformHTTP(t, func(request *http.Request) ([]byte, error) {
		if request.URL.String() != "https://www.kernel.org/pub/software/scm/git/sha256sums.asc" {
			t.Fatal(request.URL)
		}
		return []byte(sums), nil
	})
	release, err := GitSourceRelease(context.Background(), "2.53.0")
	if err != nil || release.Version != "2.53.0" {
		t.Fatalf("%+v %v", release, err)
	}
}
func TestGitWindowsAssetSelection(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			suffix := "64-bit"
			if arch == "arm64" {
				suffix = "arm64"
			}
			release := gitWindowsReleaseInfo{Tag: "v2.53.0.windows.2", Assets: []gitWindowsAsset{{Name: "MinGit-2.53.0.2-busybox-" + suffix + ".zip", URL: "https://example.com/busybox", Digest: "sha256:" + strings.Repeat("a", 64)}, {Name: "MinGit-2.53.0.2-" + suffix + ".zip", URL: "https://example.com/git.zip", Digest: "sha256:" + strings.Repeat("a", 64)}}}
			selected, matched, err := selectGitWindowsRelease(release, "2.53", arch)
			if err != nil || !matched || selected.Version != "2.53.0" || selected.URL != "https://example.com/git.zip" {
				t.Fatalf("%+v %v %v", selected, matched, err)
			}
			release.Assets = release.Assets[:1]
			if _, _, err := selectGitWindowsRelease(release, "latest", arch); err == nil {
				t.Fatal("busybox-only accepted")
			}
			release.Assets = []gitWindowsAsset{{Name: "MinGit-2.53.0.2-" + suffix + ".zip", URL: "https://example.com/git.zip"}}
			if _, _, err := selectGitWindowsRelease(release, "latest", arch); err == nil || !strings.Contains(err.Error(), "SHA-256") {
				t.Fatal(err)
			}
		})
	}
	for _, release := range []gitWindowsReleaseInfo{{Tag: "v2.53.0.windows.2", Prerelease: true}, {Tag: "v2.53.0.windows.2", Draft: true}, {Tag: "v2.5.0.windows.1"}, {Tag: "v2.53.0.rc0.windows.1"}} {
		if _, matched, err := selectGitWindowsRelease(release, "2.53", "amd64"); matched || err != nil {
			t.Fatalf("%+v %v %v", release, matched, err)
		}
	}
}
func TestGitWindowsCatalogPagination(t *testing.T) {
	pages := 0
	platformHTTP(t, func(request *http.Request) ([]byte, error) {
		pages++
		if pages == 1 {
			if request.URL.Query().Get("page") != "1" {
				t.Fatal(request.URL)
			}
			releases := make([]gitWindowsReleaseInfo, 100)
			for i := range releases {
				releases[i] = gitWindowsReleaseInfo{Tag: "v2.53.0.windows.2", Prerelease: true}
			}
			return json.Marshal(releases)
		}
		if request.URL.Query().Get("page") != "2" {
			t.Fatal(request.URL)
		}
		return json.Marshal([]gitWindowsReleaseInfo{{Tag: "v2.53.0.windows.2", Assets: []gitWindowsAsset{{Name: "MinGit-2.53.0.2-64-bit.zip", URL: "https://example.com/git.zip", Digest: "sha256:" + strings.Repeat("a", 64)}}}})
	})
	release, err := GitWindowsRelease(context.Background(), "2.53", "amd64")
	if err != nil || release.Version != "2.53.0" || pages != 2 {
		t.Fatalf("%+v %v pages=%d", release, err, pages)
	}
}
func TestGitWindowsLatestAndUnsupportedArch(t *testing.T) {
	platformHTTP(t, func(request *http.Request) ([]byte, error) {
		if !strings.HasSuffix(request.URL.Path, "/latest") {
			t.Fatal(request.URL)
		}
		return json.Marshal(gitWindowsReleaseInfo{Tag: "v2.53.0.windows.2", Assets: []gitWindowsAsset{{Name: "MinGit-2.53.0.2-arm64.zip", URL: "https://example.com/git.zip", Digest: "sha256:" + strings.Repeat("a", 64)}}})
	})
	release, err := GitWindowsRelease(context.Background(), "latest", "arm64")
	if err != nil || release.Format != "zip" {
		t.Fatalf("%+v %v", release, err)
	}
	if _, err := GitWindowsRelease(context.Background(), "latest", "386"); err == nil {
		t.Fatal("unsupported architecture accepted")
	}
}
