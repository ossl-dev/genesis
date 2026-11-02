package plugins

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ossl-dev/genesis/packages/core"
)

type ArchiveRelease struct{ Version, URL, SHA256, Format string }
type InstallationRecoveryError struct{ Err error }

func (e *InstallationRecoveryError) Error() string { return e.Err.Error() }
func (e *InstallationRecoveryError) Unwrap() error { return e.Err }

var checksumPattern = regexp.MustCompile(`(?i)^[a-f0-9]{64}$`)

const maxArchiveBytes int64 = 2 << 30
const maxExtractedBytes int64 = 8 << 30
const maxArchiveEntries = 100000

type contextReader struct {
	context context.Context
	reader  io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.context.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

var HTTPClient = &http.Client{Timeout: 5 * time.Minute}

func FetchText(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	body, err := fetchBody(ctx, url)
	if err != nil {
		return "", err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 32<<20+1))
	if len(data) > 32<<20 {
		return "", errors.New("release metadata exceeds 32 MiB")
	}
	return string(data), err
}
func FetchJSON(ctx context.Context, url string, target any) error {
	text, err := FetchText(ctx, url)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(text), target)
}
func fetchBody(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "genesis")
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed (%d): %s", resp.StatusCode, url)
	}
	return resp.Body, nil
}
func download(ctx context.Context, release ArchiveRelease, target string) error {
	if !checksumPattern.MatchString(release.SHA256) {
		return errors.New("missing or invalid release SHA-256")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	body, err := fetchBody(ctx, release.URL)
	if err != nil {
		return err
	}
	defer body.Close()
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(body, maxArchiveBytes+1))
	err = copyErr
	if err == nil && written > maxArchiveBytes {
		err = errors.New("archive exceeds 2 GiB download limit")
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), release.SHA256) {
		return errors.New("archive SHA-256 mismatch")
	}
	return nil
}
func SingleDirectory(root string) (string, error) {
	items, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	if len(items) != 1 || !items[0].IsDir() {
		return "", errors.New("expected a single runtime directory in archive")
	}
	return filepath.Join(root, items[0].Name()), nil
}
func contained(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func InstallDirectory(r *core.Runtime, destination, executable string, prepare func(string) (string, error), verify func(string) error) (err error) {
	r.Ensure()
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	if filepath.Dir(destination) == destination {
		return errors.New("cannot install into a filesystem root")
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	lock := destination + ".genesis-lock"
	if err = os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("installation locked or unavailable: %s: %w", lock, err)
	}
	defer os.Remove(lock)
	if stat, e := os.Lstat(destination); e == nil {
		if !stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
			return errors.New("installation destination must be a real directory")
		}
		items, e := os.ReadDir(destination)
		if e != nil {
			return e
		}
		if len(items) > 0 && executable != "" && !exists(filepath.Join(destination, executable)) {
			return fmt.Errorf("refusing to replace nonempty directory without %s", executable)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".genesis-")
	if err != nil {
		return err
	}
	preserve := false
	defer func() {
		if !preserve {
			if e := os.RemoveAll(stage); e != nil && r.Logger != nil {
				r.Logger.Printf("could not remove staging %s: %v", stage, e)
			}
		}
	}()
	selected, err := prepare(stage)
	if err != nil {
		return err
	}
	selected, err = filepath.EvalSymlinks(selected)
	if err != nil {
		return err
	}
	realStage, err := filepath.EvalSymlinks(stage)
	if err != nil {
		return err
	}
	if selected == realStage || !contained(realStage, selected) {
		return errors.New("runtime directory escapes staging")
	}
	if err = verify(selected); err != nil {
		return err
	}
	previous := filepath.Join(stage, "previous")
	_, statErr := os.Lstat(destination)
	hadPrevious := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if hadPrevious {
		if err = os.Rename(destination, previous); err != nil {
			return err
		}
	}
	promoted := false
	if err = os.Rename(selected, destination); err == nil {
		promoted = true
		err = verify(destination)
	}
	if err != nil {
		var recovery error
		if promoted {
			recovery = os.Rename(destination, filepath.Join(stage, "failed"))
		}
		if recovery == nil && hadPrevious {
			recovery = os.Rename(previous, destination)
		}
		if recovery != nil {
			preserve = true
			return &InstallationRecoveryError{fmt.Errorf("installation failed: %v; recovery failed: %v; backup retained at %s", err, recovery, previous)}
		}
		return err
	}
	return nil
}
func InstallArchive(r *core.Runtime, destination, executable string, release ArchiveRelease, selectRoot func(string) (string, error), verify func(string) error) error {
	return InstallDirectory(r, destination, executable, func(stage string) (string, error) {
		archive := filepath.Join(stage, "download")
		if err := download(r.Context, release, archive); err != nil {
			return "", err
		}
		unpack := filepath.Join(stage, "unpack")
		if err := os.Mkdir(unpack, 0700); err != nil {
			return "", err
		}
		if err := extractArchive(r.Context, archive, unpack, release.Format); err != nil {
			return "", err
		}
		return selectRoot(unpack)
	}, verify)
}
func archivePath(root, name string) (string, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") || strings.Contains(normalized, ":") {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return "", fmt.Errorf("unsafe archive path %q", name)
		}
	}
	target := filepath.Join(root, filepath.FromSlash(path.Clean(normalized)))
	if !contained(root, target) {
		return "", errors.New("archive path escapes extraction root")
	}
	return target, nil
}
func safeParent(root, target string) error {
	parent := filepath.Dir(target)
	for parent != root && contained(root, parent) {
		if info, err := os.Lstat(parent); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return errors.New("archive entry traverses symlink")
		}
		parent = filepath.Dir(parent)
	}
	return os.MkdirAll(filepath.Dir(target), 0755)
}
func writeArchiveFile(ctx context.Context, root, target string, mode os.FileMode, reader io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeParent(root, target); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm()&0777)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(contextReader{ctx, reader}, maxArchiveBytes+1))
	err = copyErr
	if err == nil && written > maxArchiveBytes {
		err = errors.New("archive entry exceeds 2 GiB limit")
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func extractArchive(ctx context.Context, archive, root, format string) error {
	count := 0
	var extracted int64
	account := func(size int64) error {
		count++
		if count > maxArchiveEntries {
			return errors.New("archive exceeds 100000 entries")
		}
		if size < 0 || size > maxArchiveBytes {
			return errors.New("archive entry exceeds 2 GiB limit")
		}
		extracted += size
		if extracted > maxExtractedBytes {
			return errors.New("archive exceeds 8 GiB extraction limit")
		}
		return nil
	}
	if format == "zip" {
		reader, err := zip.OpenReader(archive)
		if err != nil {
			return err
		}
		defer reader.Close()
		for _, item := range reader.File {
			if err := ctx.Err(); err != nil {
				return err
			}
			target, err := archivePath(root, item.Name)
			if err != nil {
				return err
			}
			if item.UncompressedSize64 > uint64(maxArchiveBytes) {
				return errors.New("archive entry exceeds 2 GiB limit")
			}
			if err := account(int64(item.UncompressedSize64)); err != nil {
				return err
			}
			if item.Mode()&os.ModeSymlink != 0 {
				return errors.New("ZIP symlinks are unsupported")
			}
			if item.FileInfo().IsDir() {
				if err := safeParent(root, target); err != nil {
					return err
				}
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
				continue
			}
			body, err := item.Open()
			if err != nil {
				return err
			}
			err = writeArchiveFile(ctx, root, target, item.Mode(), body)
			body.Close()
			if err != nil {
				return err
			}
		}
	} else if format == "tar.gz" {
		file, err := os.Open(archive)
		if err != nil {
			return err
		}
		defer file.Close()
		gz, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gz.Close()
		reader := tar.NewReader(gz)
		type link struct {
			name, target string
			hard         bool
		}
		var links []link
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			item, err := reader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			target, err := archivePath(root, item.Name)
			if err != nil {
				return err
			}
			if err := account(item.Size); err != nil {
				return err
			}
			switch item.Typeflag {
			case tar.TypeDir:
				if err := safeParent(root, target); err != nil {
					return err
				}
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			case tar.TypeReg, tar.TypeRegA:
				if err := writeArchiveFile(ctx, root, target, os.FileMode(item.Mode), reader); err != nil {
					return err
				}
			case tar.TypeSymlink, tar.TypeLink:
				linkTarget := item.Linkname
				if item.Typeflag == tar.TypeSymlink {
					if filepath.IsAbs(linkTarget) || strings.Contains(linkTarget, ":") {
						return errors.New("unsafe archive symlink")
					}
					linkTarget = filepath.Join(filepath.Dir(target), filepath.FromSlash(strings.ReplaceAll(linkTarget, "\\", "/")))
				} else {
					linkTarget, err = archivePath(root, item.Linkname)
					if err != nil {
						return err
					}
				}
				if !contained(root, linkTarget) {
					return errors.New("archive link escapes extraction root")
				}
				links = append(links, link{target, linkTarget, item.Typeflag == tar.TypeLink})
			default:
				return fmt.Errorf("unsupported archive entry type %d", item.Typeflag)
			}
		}
		var hardLinks []link
		for _, item := range links {
			if item.hard {
				hardLinks = append(hardLinks, item)
				continue
			}
			if err := safeParent(root, item.name); err != nil {
				return err
			}
			relative, err := filepath.Rel(filepath.Dir(item.name), item.target)
			if err != nil {
				return err
			}
			if err = os.Symlink(relative, item.name); err != nil {
				return err
			}
		}
		for len(hardLinks) > 0 {
			pending := hardLinks[:0]
			for _, item := range hardLinks {
				if err := safeParent(root, item.name); err != nil {
					return err
				}
				resolved, err := filepath.EvalSymlinks(item.target)
				if os.IsNotExist(err) {
					pending = append(pending, item)
					continue
				}
				if err != nil || !contained(root, resolved) {
					return errors.New("unsafe archive hard link")
				}
				info, err := os.Stat(resolved)
				if err != nil {
					return err
				}
				if !info.Mode().IsRegular() {
					return errors.New("archive hard link target must be regular file")
				}
				if err = os.Link(resolved, item.name); err != nil {
					return err
				}
			}
			if len(pending) == len(hardLinks) {
				return errors.New("unresolved or cyclic archive hard links")
			}
			hardLinks = pending
		}
	} else {
		return fmt.Errorf("unsupported archive format %q", format)
	}
	if count == 0 {
		return errors.New("empty archive")
	}
	return nil
}
