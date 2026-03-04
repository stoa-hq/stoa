// Package admin embeds the SvelteKit admin frontend static build and exposes
// a handler that serves it under /admin/*. All non-file requests fall back to
// index.html so that the SPA router handles client-side navigation.
package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

// Files holds the compiled SvelteKit build.
// Run `make admin-build` to populate admin/build/ before embedding.
//
//go:embed all:build
var Files embed.FS

// Handler returns an http.Handler that serves the embedded admin SPA.
// Mount it at /admin and /admin/* in the router.
func Handler() http.Handler {
	sub, err := fs.Sub(Files, "build")
	if err != nil {
		panic("admin: build directory missing – run `make admin-build`")
	}

	// Cache index.html bytes once at startup.
	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic("admin: build/index.html missing – run `make admin-build`")
	}

	// fileServer serves static assets (JS, CSS, images, …).
	// IMPORTANT: we never let it serve "index.html" directly because Go's
	// http.FileServer redirects any path ending in "/index.html" to "./"
	// which would break the SPA. We serve index.html ourselves via indexHTML.
	fileServer := http.FileServerFS(sub)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the /admin prefix to get the relative asset path.
		rel := strings.TrimPrefix(r.URL.Path, "/admin")
		rel = strings.TrimPrefix(rel, "/")

		// Empty path or explicit "index.html" → serve SPA shell directly.
		if rel == "" || rel == "index.html" {
			// SvelteKit needs inline scripts; relax CSP for the admin SPA.
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(indexHTML) //nolint:errcheck
			return
		}

		// Try to open the exact file (non-directory).
		if f, openErr := sub.Open(rel); openErr == nil {
			stat, statErr := f.Stat()
			f.Close()
			if statErr == nil && !stat.IsDir() {
				// Rewrite URL.Path so the file server resolves the asset
				// relative to the build root (without the /admin prefix).
				fileServer.ServeHTTP(w, withPath(r, "/"+rel))
				return
			}
		}

		// Unknown path → SPA client-side routing handles it.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML) //nolint:errcheck
	})
}

// withPath returns a shallow copy of r with URL.Path replaced by p.
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
