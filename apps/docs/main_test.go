package main

import (
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsBuildAndLinks(t *testing.T) {
	pages, err := loadPages()
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 37 {
		t.Fatalf("missing pages: %d", len(pages))
	}
	out := t.TempDir()
	if err = build(out, pages); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "search.html")); err != nil {
		t.Fatal("static search index missing", err)
	}
	known := map[string]bool{}
	for _, p := range pages {
		known["/"+p.Path] = true
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(p.Path)))
		if err != nil || !strings.Contains(string(data), "<main") {
			t.Fatalf("%s %v", p.Path, err)
		}
		if !strings.Contains(string(data), "apps/docs/"+strings.TrimSuffix(p.Path, ".html")+".md") {
			t.Errorf("wrong edit URL for %s", p.Path)
		}
		if !strings.Contains(string(data), `href="/search.html"`) {
			t.Errorf("missing static search link")
		}
		if strings.Contains(string(data), "<script") {
			t.Fatal("browser scripting dependency")
		}
	}
	for _, p := range pages {
		for _, match := range hrefPattern.FindAllStringSubmatch(string(p.HTML), -1) {
			target := strings.Split(match[1], "#")[0]
			if target == "" || strings.Contains(target, ":") || strings.HasPrefix(target, "//") {
				continue
			}
			if !strings.HasPrefix(target, "/") {
				target = path.Join("/", path.Dir(p.Path), target)
			}
			if strings.HasSuffix(target, ".html") && !known[target] {
				t.Errorf("%s links to missing %s", p.Path, target)
			}
		}
	}
}
func TestDocsServeSearchAndEscaping(t *testing.T) {
	pages, err := loadPages()
	if err != nil {
		t.Fatal(err)
	}
	server := handler(pages)
	for _, url := range []string{"/", "/guide/configuration", "/search?q=plugins", "/search?q=%3Cscript%3E"} {
		req := httptest.NewRequest("GET", url, nil)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s: %d", url, rec.Code)
		}
		if strings.Contains(rec.Body.String(), "<script>") {
			t.Fatal("unescaped search")
		}
	}
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest("GET", "/missing", nil))
	if rec.Code != 404 {
		t.Fatal(rec.Code)
	}
}
