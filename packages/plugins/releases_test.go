package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func mockHTTP(t *testing.T, respond func(*http.Request) (string, error)) {
	t.Helper()
	previous := HTTPClient
	HTTPClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		text, err := respond(r)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(text)), Request: r}, nil
	})}
	t.Cleanup(func() { HTTPClient = previous })
}

const testHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestGoReleasePlatforms(t *testing.T) {
	for _, c := range []struct{ platform, arch, system, fileArch, format string }{{"macos", "arm64", "darwin", "arm64", "tar.gz"}, {"linux", "amd64", "linux", "amd64", "tar.gz"}, {"windows", "arm64", "windows", "arm64", "zip"}} {
		t.Run(c.platform, func(t *testing.T) {
			filename := fmt.Sprintf("go1.22.5.%s-%s.%s", c.system, c.fileArch, c.format)
			mockHTTP(t, func(r *http.Request) (string, error) {
				return fmt.Sprintf(`[{"files":[{"filename":%q,"sha256":%q}]}]`, filename, testHash), nil
			})
			release, err := GoRelease(context.Background(), "1.22.5", c.platform, c.arch)
			if err != nil || release.SHA256 != testHash || !strings.HasSuffix(release.URL, filename) || release.Format != c.format {
				t.Fatal(release, err)
			}
		})
	}
}
func TestJavaReleaseRange(t *testing.T) {
	for _, c := range []struct{ version, normalized, upper string }{{"17", "17", "18"}, {"17.0", "17.0", "17.1"}, {"17.0.9", "17.0.9", "17.0.10"}, {"1.8", "8", "9"}} {
		t.Run(c.version, func(t *testing.T) {
			mockHTTP(t, func(r *http.Request) (string, error) {
				if !strings.Contains(r.URL.Path, "["+c.normalized+","+c.upper+")") {
					t.Errorf("bad range %s", r.URL.Path)
				}
				if r.URL.Query().Get("architecture") != "aarch64" || r.URL.Query().Get("os") != "mac" {
					t.Error(r.URL)
				}
				return fmt.Sprintf(`[{"binaries":[{"package":{"link":"https://example.com/jdk.tar.gz","checksum":%q}}]}]`, testHash), nil
			})
			release, err := JavaRelease(context.Background(), c.version, "macos", "arm64")
			if err != nil || release.Format != "tar.gz" || release.SHA256 != testHash {
				t.Fatal(release, err)
			}
		})
	}
}
func TestBunReleasePlatforms(t *testing.T) {
	for _, c := range []struct{ platform, arch, libc, filename string }{{"macos", "arm64", "auto", "bun-darwin-aarch64.zip"}, {"linux", "amd64", "glibc", "bun-linux-x64-baseline.zip"}, {"linux", "arm64", "musl", "bun-linux-aarch64-musl.zip"}, {"windows", "amd64", "auto", "bun-windows-x64-baseline.zip"}, {"windows", "arm64", "auto", "bun-windows-aarch64.zip"}} {
		t.Run(c.platform+c.arch+c.libc, func(t *testing.T) {
			mockHTTP(t, func(r *http.Request) (string, error) {
				return fmt.Sprintf(`{"assets":[{"name":%q,"browser_download_url":"https://example.com/archive.zip","digest":"sha256:%s"}]}`, c.filename, testHash), nil
			})
			release, err := BunRelease(context.Background(), "1.3.2", c.platform, c.arch, c.libc)
			if err != nil || release.SHA256 != testHash || release.Format != "zip" {
				t.Fatal(release, err)
			}
		})
	}
}
func TestBunChecksumFallback(t *testing.T) {
	mockHTTP(t, func(r *http.Request) (string, error) {
		if r.URL.Host == "example.com" {
			return testHash + " *bun-darwin-aarch64.zip\n", nil
		}
		return `{"assets":[{"name":"bun-darwin-aarch64.zip","browser_download_url":"https://example.com/archive.zip"},{"name":"SHASUMS256.txt","browser_download_url":"https://example.com/sums"}]}`, nil
	})
	release, err := BunRelease(context.Background(), "1.3.2", "macos", "arm64", "auto")
	if err != nil || release.SHA256 != testHash {
		t.Fatal(release, err)
	}
}
func TestDenoReleasePlatforms(t *testing.T) {
	for _, c := range []struct{ platform, arch, target string }{{"macos", "arm64", "aarch64-apple-darwin"}, {"windows", "amd64", "x86_64-pc-windows-msvc"}, {"windows", "arm64", "aarch64-pc-windows-msvc"}} {
		t.Run(c.platform+c.arch, func(t *testing.T) {
			mockHTTP(t, func(r *http.Request) (string, error) {
				if !strings.HasSuffix(r.URL.Path, "deno-"+c.target+".zip.sha256sum") {
					t.Error(r.URL)
				}
				return testHash + "  deno-" + c.target + ".zip", nil
			})
			release, err := DenoRelease(context.Background(), "2.5.4", c.platform, c.arch)
			if err != nil || release.SHA256 != testHash {
				t.Fatal(release, err)
			}
		})
	}
}
func TestNodeStableReleaseSelection(t *testing.T) {
	for _, c := range []struct{ platform, system, format string }{{"macos", "darwin", "tar.gz"}, {"windows", "win", "zip"}} {
		t.Run(c.platform, func(t *testing.T) {
			filename := "node-v22.18.0-" + c.system + "-arm64." + c.format
			mockHTTP(t, func(r *http.Request) (string, error) {
				if strings.HasSuffix(r.URL.Path, "index.json") {
					return `[{"version":"v22.19.0-rc.1"},{"version":"v24.0.0"},{"version":"v22.18.0"},{"version":"v22.17.0"}]`, nil
				}
				return testHash + "  " + filename, nil
			})
			release, err := NodeRelease(context.Background(), "22", c.platform, "arm64")
			if err != nil || !strings.HasSuffix(release.URL, filename) {
				t.Fatal(release, err)
			}
		})
	}
}
func TestUnavailableReleases(t *testing.T) {
	mockHTTP(t, func(r *http.Request) (string, error) {
		if strings.Contains(r.URL.Path, "bun") {
			return `{"assets":[]}`, nil
		}
		return `[]`, nil
	})
	calls := []func() error{func() error { _, err := GoRelease(context.Background(), "1.22.999", "macos", "arm64"); return err }, func() error { _, err := JavaRelease(context.Background(), "17.0.999", "windows", "amd64"); return err }, func() error { _, err := NodeRelease(context.Background(), "2", "macos", "arm64"); return err }, func() error {
		_, err := BunRelease(context.Background(), "1.3.2", "windows", "amd64", "auto")
		return err
	}, func() error { _, err := DenoRelease(context.Background(), "2.5.4", "macos", "arm64"); return err }, func() error { _, err := GoRelease(context.Background(), "1.22.5", "linux", "ppc64"); return err }}
	for index, call := range calls {
		if err := call(); err == nil {
			t.Errorf("case %d unexpectedly succeeds", index)
		}
	}
}
func TestChecksumManifest(t *testing.T) {
	checksum, err := FileChecksum(strings.Repeat("b", 64)+" other.zip\n"+testHash+" *correct.zip\n", "correct.zip")
	if err != nil || checksum != testHash {
		t.Fatal(checksum, err)
	}
	if _, err = FileChecksum("bad checksum", "correct.zip"); err == nil {
		t.Fatal("missing checksum accepted")
	}
}
