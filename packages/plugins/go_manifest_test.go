package plugins

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGoManifestSelection(t *testing.T) {
	const filename = "go1.27.3.darwin-arm64.tar.gz"
	for _, c := range []struct{ name, input, want string }{
		{"selected", `[{"files":[{"filename":"` + filename + `","sha256":"` + testHash + `"}]}]`, testHash},
		{"empty", `[]`, ""}, {"null", `null`, ""}, {"absent files", `[{},null,{"files":null}]`, ""},
		{"invalid checksum then valid", `[{"files":[{"filename":"` + filename + `","sha256":"bad"}]},{"files":[{"filename":"` + filename + `","sha256":"` + testHash + `"}]}]`, testHash},
		{"omitted fields after reuse", `[{"files":[{"filename":"one","sha256":"` + testHash + `"},{"filename":"two","sha256":"` + testHash + `"}]},{"files":[{"filename":"three"}]},{"files":[{"filename":"four"},{"filename":"` + filename + `"}]}]`, ""},
		{"earliest matching archive", `[{"files":[{"filename":"` + filename + `","sha256":"` + testHash + `"}]},{"files":[{"filename":"` + filename + `","sha256":"` + strings.Repeat("b", 64) + `"}]}]`, testHash},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := readGoArchiveChecksum(strings.NewReader(c.input), filename)
			if err != nil || got != c.want {
				t.Fatalf("checksum=%q error=%v", got, err)
			}
		})
	}
}
func TestGoManifestRejectsMalformedTail(t *testing.T) {
	prefix := `[{"files":[{"filename":"target","sha256":"` + testHash + `"}]}`
	for _, input := range []string{prefix, prefix + `,`, prefix + `,{"files":[{"filename":17}]}]`, prefix + `,{"files":[{"sha256":true}]}]`, prefix + `] false`, prefix + `] garbage`, prefix + `]{}`, `{}`, `"array"`} {
		if checksum, err := readGoArchiveChecksum(strings.NewReader(input), "target"); err == nil || checksum != "" {
			t.Fatalf("accepted malformed manifest %q: %q %v", input, checksum, err)
		}
	}
}

type repeatingSpaceReader struct{}

func (repeatingSpaceReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }
func TestGoManifestBoundsAndBodyClosure(t *testing.T) {
	prefix := `[{"files":[{"filename":"target","sha256":"` + testHash + `"}]}]`
	body := &trackedBody{Reader: io.MultiReader(strings.NewReader(prefix), io.LimitReader(repeatingSpaceReader{}, maxReleaseMetadataBytes))}
	previous := HTTPClient
	HTTPClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Request: r}, nil
	})}
	t.Cleanup(func() { HTTPClient = previous })
	if _, err := goArchiveChecksum(context.Background(), "target"); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatal("metadata limit ignored", err)
	}
	if !body.closed {
		t.Fatal("metadata body leaked")
	}
}

type canceledManifestReader struct {
	prefix string
	cancel context.CancelFunc
	offset int
}

func (r *canceledManifestReader) Read(p []byte) (int, error) {
	if r.offset == len(r.prefix) {
		return 0, io.EOF
	}
	n := copy(p, r.prefix[r.offset:])
	r.offset += n
	if r.offset == len(r.prefix) {
		r.cancel()
	}
	return n, nil
}
func TestGoManifestCancellationAfterMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := &trackedBody{Reader: &canceledManifestReader{prefix: `[{"files":[{"filename":"target","sha256":"` + testHash + `"}]},`, cancel: cancel}}
	previous := HTTPClient
	HTTPClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Request: r}, nil
	})}
	t.Cleanup(func() { HTTPClient = previous })
	if checksum, err := goArchiveChecksum(ctx, "target"); !errors.Is(err, context.Canceled) || checksum != "" {
		t.Fatalf("cancellation lost after match: %q %v", checksum, err)
	}
	if !body.closed {
		t.Fatal("canceled metadata body leaked")
	}
}

type failingManifestReader struct{}

func (failingManifestReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestGoManifestReadFailureAfterMatch(t *testing.T) {
	prefix := `[{"files":[{"filename":"target","sha256":"` + testHash + `"}]}]`
	checksum, err := readGoArchiveChecksum(io.MultiReader(strings.NewReader(prefix), failingManifestReader{}), "target")
	if !errors.Is(err, io.ErrUnexpectedEOF) || checksum != "" {
		t.Fatalf("truncated HTTP body accepted: %q %v", checksum, err)
	}
}
func TestGoManifestHTTPFailureClosesBody(t *testing.T) {
	body := &trackedBody{Reader: strings.NewReader("unavailable")}
	previous := HTTPClient
	HTTPClient = &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: body, Request: r}, nil
	})}
	t.Cleanup(func() { HTTPClient = previous })
	if _, err := goArchiveChecksum(context.Background(), "target"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("HTTP error body leaked")
	}
}

type unexpectedTrailerReader struct {
	prefix         string
	offset         int
	readPastPrefix bool
}

func (r *unexpectedTrailerReader) Read(p []byte) (int, error) {
	if r.offset == len(r.prefix) {
		r.readPastPrefix = true
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, r.prefix[r.offset:])
	r.offset += n
	return n, nil
}
func TestGoManifestRejectsTrailerWithoutMaterializingIt(t *testing.T) {
	reader := &unexpectedTrailerReader{prefix: `[{"files":[{"filename":"target","sha256":"` + testHash + `"}]}]{`}
	if _, err := readGoArchiveChecksum(reader, "target"); err == nil || !strings.Contains(err.Error(), "unexpected data") {
		t.Fatal(err)
	}
	if reader.readPastPrefix {
		t.Fatal("decoder attempted to materialize a rejected trailing object")
	}
}
