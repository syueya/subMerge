// Package webui exposes the production React bundle embedded in the Go binary.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var files embed.FS

// FileSystem returns the bundle root and whether a production index is present.
func FileSystem() (fs.FS, bool) {
	root, err := fs.Sub(files, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(root, "index.html"); err != nil {
		return root, false
	}
	return root, true
}
