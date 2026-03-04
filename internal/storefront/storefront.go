// Package storefront embeds the SvelteKit storefront frontend static build and
// exposes a handler that serves it at the root path. All non-file requests fall
// back to index.html so that the SPA router handles client-side navigation.
package storefront

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
)

// Files holds the compiled SvelteKit build.
// Run `make storefront-build` to populate internal/storefront/build/ before embedding.
//
//go:embed all:build
var Files embed.FS

// Handler returns an http.Handler that serves the embedded storefront SPA.
// Mount it at /* in the router – the more specific /api and /admin routes
// are registered first and take priority in chi's radix tree.
func Handler() http.Handler {
	sub, err := fs.Sub(Files, "build")
	if err != nil {
		panic("storefront: build directory missing – run `make storefront-build`")
	}

	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("storefront: build/index.html missing – run `make storefront-build`")
	}

	fileServer := http.FileServerFS(sub)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Path
		if rel == "" || rel == "/" {
			serveIndex(w, indexHTML)
			return
		}

		// Strip leading slash for fs.Open.
		assetPath := rel
		if len(assetPath) > 0 && assetPath[0] == '/' {
			assetPath = assetPath[1:]
		}

		if assetPath == "index.html" {
			serveIndex(w, indexHTML)
			return
		}

		// Try to serve exact static asset.
		if f, openErr := sub.Open(assetPath); openErr == nil {
			stat, statErr := f.Stat()
			f.Close()
			if statErr == nil && !stat.IsDir() {
				fileServer.ServeHTTP(w, withPath(r, "/"+assetPath))
				return
			}
		}

		// Unknown path → SPA routing.
		serveIndex(w, indexHTML)
	})
}

func serveIndex(w http.ResponseWriter, html []byte) {
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(html) //nolint:errcheck
}

func withPath(r *http.Request, p string) *http.Request {
	r2 := r.Clone(r.Context())
	u := &url.URL{}
	if r.URL != nil {
		*u = *r.URL
	}
	u.Path = p
	u.RawPath = ""
	r2.URL = u
	return r2
}
