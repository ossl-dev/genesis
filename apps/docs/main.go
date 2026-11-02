// Genesis documentation is built and served without browser scripting.
package main

import (
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

//go:embed *.md api/*.md guide/*.md plugins/*.md
var sources embed.FS

type page struct {
	Path, Title, Text string
	SearchText        string
	HTML              template.HTML
	Outline           template.HTML
}
type view struct {
	Title, Current, Query string
	Source                string
	Search                bool
	Content, Outline      template.HTML
	Pages                 []page
}

var hrefPattern = regexp.MustCompile(`href="([^"]+)"`)
var headingPattern = regexp.MustCompile(`<h([23]) id="([^"]+)">(.*?)</h[23]>`)
var tagPattern = regexp.MustCompile(`<[^>]*>`)
var siteTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="description" content="Genesis developer environment provisioning"><title>{{.Title}} · Genesis</title><style>
:root{color-scheme:light dark;font-family:system-ui,sans-serif;line-height:1.65;color:light-dark(#242424,#dfdfdf);background:light-dark(#fff,#181818)}*{box-sizing:border-box}body{margin:0}a{color:light-dark(#3156ab,#9ab7ff);text-underline-offset:3px}header{border-bottom:1px solid #8885;padding:1rem 2rem;display:flex;align-items:center;gap:1.5rem;flex-wrap:wrap}header strong{font-size:1.2rem;margin-right:auto}header form{display:flex;gap:.4rem}input,button{font:inherit;padding:.3rem .6rem;background:transparent;color:inherit;border:1px solid #8888;border-radius:4px}.layout{max-width:1450px;margin:auto;display:grid;grid-template-columns:230px minmax(0,800px) 180px;gap:2rem;padding:2rem}aside{font-size:.875rem}aside nav{display:grid;gap:.3rem}aside summary{font-weight:600;margin:.6rem 0}aside a[aria-current=page]{font-weight:700}main{min-width:0}h1,h2,h3{line-height:1.3;scroll-margin-top:1rem}h1{font-size:2.2rem}h2{margin-top:2rem;border-top:1px solid #8884;padding-top:1rem}pre{overflow:auto;padding:1rem;background:#8881;border:1px solid #8884;border-radius:6px}code{font-family:ui-monospace,monospace;font-size:.88em}p code,li code,td code{background:#8882;padding:.12rem .25rem}table{display:block;overflow:auto;border-collapse:collapse}td,th{border:1px solid #8885;padding:.4rem .7rem;text-align:left}blockquote{border-left:3px solid #888;padding-left:1rem;margin-left:0}footer{padding:2rem;border-top:1px solid #8884;font-size:.9rem}.skip{position:absolute;top:-100px}.skip:focus{top:0;background:Canvas;padding:1rem}.outline nav{display:grid;gap:.5rem}.outline{position:sticky;top:1rem;align-self:start}small{color:light-dark(#555,#aaa)}@media(max-width:1100px){.layout{grid-template-columns:200px minmax(0,1fr)}.outline{display:none}}@media(max-width:720px){header{padding:1rem;gap:.8rem}.layout{display:block;padding:1rem}aside{margin-bottom:2rem}h1{font-size:1.8rem}}
</style></head><body><a class="skip" href="#content">Skip to content</a><header><strong><a href="/index.html">Genesis</a></strong><a href="/guide/getting-started.html">Guide</a><a href="/plugins/overview.html">Plugins</a><a href="/api/core.html">API</a><a href="https://github.com/ossl-dev/genesis">GitHub</a>{{if .Search}}<form action="/search" method="get" role="search"><label for="q">Search</label><input id="q" name="q" type="search" value="{{.Query}}"><button>Find</button></form>{{else}}<a href="/search.html">Search index</a>{{end}}</header><div class="layout"><aside><details open><summary>Documentation</summary><nav>{{range .Pages}}<a href="/{{.Path}}"{{if eq .Path $.Current}} aria-current="page"{{end}}>{{.Title}}</a>{{end}}</nav></details></aside><main id="content">{{.Content}}<footer>{{if .Source}}<a href="https://github.com/ossl-dev/genesis/edit/main/apps/docs/{{.Source}}">Edit source on GitHub</a>{{end}}<p>Genesis · <a href="https://github.com/ossl-dev/genesis/releases">Changelog</a> · <a href="/guide/contributing.html">Contributing</a></p><small>Released under the MIT License. Copyright © 2024-present Mohammed Aman Khan.</small></footer></main><aside class="outline"><strong>On this page</strong><nav>{{.Outline}}</nav></aside></div></body></html>`))

func loadPages() (pages []page, err error) {
	renderer := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithParserOptions(parser.WithAutoHeadingID()))
	err = fs.WalkDir(sources, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			return nil
		}
		data, err := sources.ReadFile(name)
		if err != nil {
			return err
		}
		text := string(data)
		title := strings.TrimSuffix(path.Base(name), ".md")
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimPrefix(line, "# ")
				break
			}
		}
		var result bytes.Buffer
		if err = renderer.Convert(data, &result); err != nil {
			return err
		}
		rendered := hrefPattern.ReplaceAllStringFunc(result.String(), func(match string) string {
			target := hrefPattern.FindStringSubmatch(match)[1]
			u, err := url.Parse(html.UnescapeString(target))
			if err != nil || u.IsAbs() || u.Host != "" || u.Path == "" {
				return match
			}
			if strings.HasSuffix(u.Path, ".md") {
				u.Path = strings.TrimSuffix(u.Path, ".md") + ".html"
			} else if path.Ext(u.Path) == "" {
				if u.Path == "/" {
					u.Path = "/index.html"
				} else {
					u.Path += ".html"
				}
			}
			return `href="` + template.HTMLEscapeString(u.String()) + `"`
		})
		var outline strings.Builder
		for _, heading := range headingPattern.FindAllStringSubmatch(rendered, -1) {
			fmt.Fprintf(&outline, `<a href="#%s">%s</a>`, template.HTMLEscapeString(heading[2]), template.HTMLEscapeString(html.UnescapeString(tagPattern.ReplaceAllString(heading[3], ""))))
		}
		pages = append(pages, page{Path: strings.TrimSuffix(name, ".md") + ".html", Title: title, Text: text, SearchText: strings.ToLower(text), HTML: template.HTML(rendered), Outline: template.HTML(outline.String())})
		return nil
	})
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages, err
}
func renderPage(p page, pages []page, search bool) ([]byte, error) {
	var output bytes.Buffer
	source := ""
	if p.Text != "" {
		source = strings.TrimSuffix(p.Path, ".html") + ".md"
	}
	err := siteTemplate.Execute(&output, view{Title: p.Title, Current: p.Path, Source: source, Search: search, Content: p.HTML, Outline: p.Outline, Pages: pages})
	return output.Bytes(), err
}
func build(output string, pages []page) error {
	var index strings.Builder
	index.WriteString("<h1>Search index</h1><p>Use your browser's Find command to search the complete documentation below. The Go documentation server provides query search.</p>")
	for _, p := range pages {
		fmt.Fprintf(&index, `<section><h2><a href="/%s">%s</a></h2><pre>%s</pre></section>`, template.HTMLEscapeString(p.Path), template.HTMLEscapeString(p.Title), template.HTMLEscapeString(p.Text))
	}
	pages = append(pages, page{Path: "search.html", Title: "Search index", HTML: template.HTML(index.String())})

	for _, p := range pages {
		data, err := renderPage(p, pages, false)
		if err != nil {
			return err
		}
		dest := filepath.Join(output, filepath.FromSlash(p.Path))
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
func handler(pages []page) http.Handler {
	mux := http.NewServeMux()
	byPath := map[string]page{}
	for _, p := range pages {
		byPath["/"+p.Path] = p
	}
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) > 256 {
			http.Error(w, "query too long", http.StatusBadRequest)
			return
		}
		var content strings.Builder
		content.WriteString("<h1>Search</h1>")
		if query != "" {
			lowerQuery := strings.ToLower(query)
			fmt.Fprintf(&content, "<p>Results for %s</p><ul>", template.HTMLEscapeString(query))
			for _, p := range pages {
				if strings.Contains(p.SearchText, lowerQuery) {
					fmt.Fprintf(&content, `<li><a href="/%s">%s</a></li>`, template.HTMLEscapeString(p.Path), template.HTMLEscapeString(p.Title))
				}
			}
			content.WriteString("</ul>")
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		siteTemplate.Execute(w, view{Title: "Search", Search: true, Query: query, Content: template.HTML(content.String()), Pages: pages})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if key == "/" {
			key = "/index.html"
		}
		if path.Ext(key) == "" {
			key += ".html"
		}
		p, ok := byPath[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		data, err := renderPage(p, pages, true)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})
	return mux
}
func main() {
	output := flag.String("output", "dist/docs", "Static output directory")
	serve := flag.String("serve", "", "Serve documentation at address (e.g. localhost:8080)")
	flag.Parse()
	pages, err := loadPages()
	if err == nil {
		if *serve != "" {
			server := &http.Server{Addr: *serve, Handler: handler(pages), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
			err = server.ListenAndServe()
		} else {
			err = build(*output, pages)
		}
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
