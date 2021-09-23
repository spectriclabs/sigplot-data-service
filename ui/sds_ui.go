package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed webapp/dist
var embeddedFiles embed.FS

var indexBytes []byte

// GetFileSystem wraps the embeddedFiles embedded
// filesystem in an http.FS to be used in Echo
func GetFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embeddedFiles, "webapp/dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

// LoadUI handles loading the static index.html
// from the embedded filesystem
func LoadUI() ([]byte, error) {
	if len(indexBytes) == 0 {
		data, err := fs.ReadFile(embeddedFiles, "webapp/dist/index.html")
		if err != nil {
			return nil, err
		}
		indexBytes = data
	}
	return indexBytes, nil
}
