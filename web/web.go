package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Handler serves the embedded dashboard SPA with client-side-routing fallback (any non-asset path serves
// index.html). Returns nil when the dashboard was not embedded — the caller then runs API + sync only.
func Handler() http.Handler {
	fsys, ok := dashboardFS()
	if !ok {
		return nil
	}
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(fsys, p); err != nil {
			// Not a real file → hand the SPA its entry so the client router can take over.
			r.URL.Path = "/"
		} else if strings.HasPrefix(p, "assets/") {
			// Vite fingerprints asset filenames, so they're safe to cache immutably.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	})
}
