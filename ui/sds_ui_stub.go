//go:build !ui

package ui

import (
	"io/fs"
	"net/http"
)

var stubHTML = []byte(`<!DOCTYPE html>
<html>
<body>
<p>SDS UI is not available in this build. Build with -tags ui to include the UI.</p>
</body>
</html>
`)

// GetFileSystem returns an empty filesystem when UI is not built in
func GetFileSystem() http.FileSystem {
	return emptyFileSystem{}
}

type emptyFileSystem struct{}

func (emptyFileSystem) Open(string) (http.File, error) {
	return nil, fs.ErrNotExist
}

// LoadUI returns a stub HTML page when UI is not built in
func LoadUI() ([]byte, error) {
	return stubHTML, nil
}
