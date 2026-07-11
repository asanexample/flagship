//go:build dashboard

// Package web embeds the built dashboard SPA (Vite output in dist/) so the Flagship binary serves its own
// UI — one image, no separate Node runtime. Built only under `-tags dashboard` (the release build), so a
// plain `go build ./...` works without the frontend having been built (see embed_stub.go).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

func dashboardFS() (fs.FS, bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, false
	}
	return sub, true
}
