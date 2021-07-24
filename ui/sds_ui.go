package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed webapp/dist
var embededFiles embed.FS

// GetFileSystem wraps the embeddedFiles embedded
// filesystem in an http.FS to be used in Echo
func GetFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "ui/dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}
