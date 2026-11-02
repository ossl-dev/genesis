package plugins

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var gitSourceChecksumPattern = regexp.MustCompile(`(?i)^([a-f0-9]{64})\s+git-(\d+\.\d+\.\d+)\.tar\.gz$`)

func GitSourceRelease(ctx context.Context, requested string) (ArchiveRelease, error) {
	const base = "https://www.kernel.org/pub/software/scm/git"
	sums, err := FetchText(ctx, base+"/sha256sums.asc")
	if err != nil {
		return ArchiveRelease{}, err
	}
	return selectGitSourceRelease(sums, requested, base)
}
func selectGitSourceRelease(sums, requested, base string) (ArchiveRelease, error) {
	var releases []ArchiveRelease
	for _, line := range strings.Split(sums, "\n") {
		match := gitSourceChecksumPattern.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if match == nil || (requested != "latest" && !MatchesVersion(match[2], requested)) {
			continue
		}
		releases = append(releases, ArchiveRelease{Version: match[2], URL: base + "/git-" + match[2] + ".tar.gz", SHA256: match[1], Format: "tar.gz"})
	}
	sort.SliceStable(releases, func(i, j int) bool { return platformVersionCompare(releases[i].Version, releases[j].Version) > 0 })
	if len(releases) == 0 {
		return ArchiveRelease{}, fmt.Errorf("no stable Git source release matches %s", requested)
	}
	return releases[0], nil
}

type gitWindowsAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}
type gitWindowsReleaseInfo struct {
	Tag        string            `json:"tag_name"`
	Draft      bool              `json:"draft"`
	Prerelease bool              `json:"prerelease"`
	Assets     []gitWindowsAsset `json:"assets"`
}

var gitWindowsTagPattern = regexp.MustCompile(`^v(\d+\.\d+\.\d+)\.windows\.\d+$`)
var gitWindowsDigestPattern = regexp.MustCompile(`(?i)^sha256:([a-f0-9]{64})$`)

func GitWindowsRelease(ctx context.Context, requested, arch string) (ArchiveRelease, error) {
	if arch != "amd64" && arch != "arm64" {
		return ArchiveRelease{}, fmt.Errorf("unsupported Git for Windows architecture: %s", arch)
	}
	const base = "https://api.github.com/repos/git-for-windows/git/releases"
	for page := 1; page <= 5; page++ {
		var releases []gitWindowsReleaseInfo
		if requested == "latest" {
			var release gitWindowsReleaseInfo
			if err := FetchJSON(ctx, base+"/latest", &release); err != nil {
				return ArchiveRelease{}, err
			}
			releases = []gitWindowsReleaseInfo{release}
		} else if err := FetchJSON(ctx, fmt.Sprintf("%s?per_page=100&page=%d", base, page), &releases); err != nil {
			return ArchiveRelease{}, err
		}
		for _, release := range releases {
			selected, matched, err := selectGitWindowsRelease(release, requested, arch)
			if err != nil {
				return ArchiveRelease{}, err
			}
			if matched {
				return selected, nil
			}
		}
		if requested == "latest" || len(releases) < 100 {
			break
		}
	}
	return ArchiveRelease{}, fmt.Errorf("no stable Git for Windows release matches %s in the release catalog", requested)
}
func selectGitWindowsRelease(release gitWindowsReleaseInfo, requested, arch string) (ArchiveRelease, bool, error) {
	match := gitWindowsTagPattern.FindStringSubmatch(release.Tag)
	if match == nil || release.Draft || release.Prerelease || (requested != "latest" && !MatchesVersion(match[1], requested)) {
		return ArchiveRelease{}, false, nil
	}
	suffix := "64-bit"
	if arch == "arm64" {
		suffix = "arm64"
	}
	assetPattern := regexp.MustCompile(`^MinGit-[0-9.]+-` + suffix + `\.zip$`)
	for _, asset := range release.Assets {
		if !assetPattern.MatchString(asset.Name) {
			continue
		}
		digest := gitWindowsDigestPattern.FindStringSubmatch(asset.Digest)
		if digest == nil {
			return ArchiveRelease{}, false, fmt.Errorf("no SHA-256 published for %s; choose a newer Git for Windows release", asset.Name)
		}
		return ArchiveRelease{Version: match[1], URL: asset.URL, SHA256: digest[1], Format: "zip"}, true, nil
	}
	return ArchiveRelease{}, false, fmt.Errorf("Git %s has no MinGit archive for Windows/%s", match[1], arch)
}
