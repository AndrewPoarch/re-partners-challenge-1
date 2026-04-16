// Package webui exposes the embedded static UI (HTML/CSS/JS) as an http.FS.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed web
var assets embed.FS

// FS returns the subtree rooted at web/ for http.FileServer.
func FS() fs.FS {
	sub, err := fs.Sub(assets, "web")
	if err != nil {
		panic(err)
	}
	return sub
}
