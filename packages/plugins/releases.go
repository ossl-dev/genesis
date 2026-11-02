package plugins

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

func releaseArchitecture(arch string) (string, error) {
	switch arch {
	case "amd64", "x64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	}
	return "", fmt.Errorf("unsupported architecture: %s", arch)
}
func IsMusl() bool {
	if exists("/etc/alpine-release") {
		return true
	}
	matches, _ := filepath.Glob("/lib/ld-musl-*.so.1")
	return len(matches) > 0
}
func FileChecksum(text, filename string) (string, error) {
	pattern := regexp.MustCompile(`(?i)^([a-f0-9]{64})(?:\s+\*?(.+))?$`)
	for _, line := range strings.Split(text, "\n") {
		m := pattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) > 0 && (m[2] == "" || m[2] == filename) {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("no SHA-256 published for %s", filename)
}
func GoRelease(ctx context.Context, version, platform, architecture string) (ArchiveRelease, error) {
	arch, err := releaseArchitecture(architecture)
	if err != nil {
		return ArchiveRelease{}, err
	}
	if arch == "x64" {
		arch = "amd64"
	}
	system := platform
	if system == "macos" {
		system = "darwin"
	}
	format := "tar.gz"
	if platform == "windows" {
		format = "zip"
	}
	filename := fmt.Sprintf("go%s.%s-%s.%s", version, system, arch, format)
	var releases []struct {
		Files []struct{ Filename, SHA256 string }
	}
	if err := FetchJSON(ctx, "https://go.dev/dl/?mode=json&include=all", &releases); err != nil {
		return ArchiveRelease{}, err
	}
	for _, r := range releases {
		for _, f := range r.Files {
			if f.Filename == filename && checksumPattern.MatchString(f.SHA256) {
				return ArchiveRelease{version, "https://go.dev/dl/" + filename, f.SHA256, format}, nil
			}
		}
	}
	return ArchiveRelease{}, fmt.Errorf("Go %s has no archive for %s/%s", version, platform, arch)
}
func NormalizeJavaVersion(version string) string {
	return strings.Replace(strings.TrimPrefix(version, "1."), "_", ".", 1)
}
func JavaRelease(ctx context.Context, version, platform, architecture string) (ArchiveRelease, error) {
	arch, err := releaseArchitecture(architecture)
	if err != nil {
		return ArchiveRelease{}, err
	}
	if arch == "arm64" {
		arch = "aarch64"
	}
	system := platform
	if system == "macos" {
		system = "mac"
	}
	normalized := NormalizeJavaVersion(version)
	parts := strings.Split(normalized, ".")
	var last int
	fmt.Sscanf(parts[len(parts)-1], "%d", &last)
	parts[len(parts)-1] = fmt.Sprint(last + 1)
	q := url.Values{"architecture": {arch}, "os": {system}, "image_type": {"jdk"}, "jvm_impl": {"hotspot"}, "vendor": {"eclipse"}, "release_type": {"ga"}, "project": {"jdk"}, "sort_method": {"DATE"}, "sort_order": {"DESC"}, "page_size": {"1"}}
	var releases []struct {
		Binaries []struct {
			Package struct{ Link, Checksum string }
		}
	}
	uri := "https://api.adoptium.net/v3/assets/version/" + url.PathEscape("["+normalized+","+strings.Join(parts, ".")+")") + "?" + q.Encode()
	if err := FetchJSON(ctx, uri, &releases); err != nil {
		return ArchiveRelease{}, err
	}
	if len(releases) == 0 || len(releases[0].Binaries) == 0 {
		return ArchiveRelease{}, fmt.Errorf("Temurin Java %s has no archive for %s/%s", version, platform, arch)
	}
	asset := releases[0].Binaries[0].Package
	if !checksumPattern.MatchString(asset.Checksum) {
		return ArchiveRelease{}, fmt.Errorf("missing Java checksum")
	}
	format := "tar.gz"
	if platform == "windows" {
		format = "zip"
	}
	return ArchiveRelease{version, asset.Link, asset.Checksum, format}, nil
}
func BunRelease(ctx context.Context, version, platform, architecture, libc string) (ArchiveRelease, error) {
	arch, err := releaseArchitecture(architecture)
	if err != nil {
		return ArchiveRelease{}, err
	}
	system := platform
	if system == "macos" {
		system = "darwin"
	}
	assetArch := arch
	if arch == "arm64" {
		assetArch = "aarch64"
	}
	suffix := ""
	if platform == "linux" && (libc == "musl" || libc == "auto" && IsMusl()) {
		suffix = "-musl"
	}
	if arch == "x64" {
		suffix += "-baseline"
	}
	filename := "bun-" + system + "-" + assetArch + suffix + ".zip"
	var release struct {
		Assets []struct {
			Name   string
			URL    string `json:"browser_download_url"`
			Digest string
		}
	}
	if err := FetchJSON(ctx, "https://api.github.com/repos/oven-sh/bun/releases/tags/bun-v"+version, &release); err != nil {
		return ArchiveRelease{}, err
	}
	var assetURL, digest, sumsURL string
	for _, a := range release.Assets {
		if a.Name == filename {
			assetURL = a.URL
			if len(a.Digest) >= 7 && strings.EqualFold(a.Digest[:7], "sha256:") {
				digest = a.Digest[7:]
			}
		}
		if a.Name == "SHASUMS256.txt" {
			sumsURL = a.URL
		}
	}
	if assetURL == "" {
		return ArchiveRelease{}, fmt.Errorf("Bun %s has no archive for %s/%s", version, platform, arch)
	}
	if !checksumPattern.MatchString(digest) {
		if sumsURL == "" {
			return ArchiveRelease{}, fmt.Errorf("no SHA-256 published for %s", filename)
		}
		text, err := FetchText(ctx, sumsURL)
		if err != nil {
			return ArchiveRelease{}, err
		}
		digest, err = FileChecksum(text, filename)
		if err != nil {
			return ArchiveRelease{}, err
		}
	}
	return ArchiveRelease{version, assetURL, digest, "zip"}, nil
}
func DenoRelease(ctx context.Context, version, platform, architecture string) (ArchiveRelease, error) {
	arch, err := releaseArchitecture(architecture)
	if err != nil {
		return ArchiveRelease{}, err
	}
	if arch == "arm64" {
		arch = "aarch64"
	} else {
		arch = "x86_64"
	}
	if platform == "linux" && IsMusl() {
		return ArchiveRelease{}, fmt.Errorf("Deno's official Linux archives require glibc; musl is unsupported")
	}
	target := "unknown-linux-gnu"
	if platform == "macos" {
		target = "apple-darwin"
	}
	if platform == "windows" {
		target = "pc-windows-msvc"
	}
	filename := "deno-" + arch + "-" + target + ".zip"
	uri := "https://github.com/denoland/deno/releases/download/v" + version + "/" + filename
	text, err := FetchText(ctx, uri+".sha256sum")
	if err != nil {
		return ArchiveRelease{}, err
	}
	checksum, err := FileChecksum(text, filename)
	if err != nil {
		return ArchiveRelease{}, err
	}
	return ArchiveRelease{version, uri, checksum, "zip"}, nil
}
func NodeRelease(ctx context.Context, requested, platform, architecture string) (ArchiveRelease, error) {
	arch, err := releaseArchitecture(architecture)
	if err != nil {
		return ArchiveRelease{}, err
	}
	if platform == "linux" && IsMusl() {
		return ArchiveRelease{}, fmt.Errorf("Node's official Linux archives require glibc; use a compatible runtime on musl")
	}
	var releases []struct{ Version string }
	if err := FetchJSON(ctx, "https://nodejs.org/dist/index.json", &releases); err != nil {
		return ArchiveRelease{}, err
	}
	var version string
	for _, release := range releases {
		candidate := strings.TrimPrefix(release.Version, "v")
		if fullVersion.MatchString(candidate) && MatchesVersion(candidate, requested) {
			version = release.Version
			break
		}
	}
	if version == "" {
		return ArchiveRelease{}, fmt.Errorf("no Node release matches %s", requested)
	}
	system := platform
	if system == "macos" {
		system = "darwin"
	}
	format := "tar.gz"
	if system == "windows" {
		system = "win"
		format = "zip"
	}
	filename := "node-" + version + "-" + system + "-" + arch + "." + format
	base := "https://nodejs.org/dist/" + version
	text, err := FetchText(ctx, base+"/SHASUMS256.txt")
	if err != nil {
		return ArchiveRelease{}, err
	}
	checksum, err := FileChecksum(text, filename)
	if err != nil {
		return ArchiveRelease{}, err
	}
	return ArchiveRelease{strings.TrimPrefix(version, "v"), base + "/" + filename, checksum, format}, nil
}
