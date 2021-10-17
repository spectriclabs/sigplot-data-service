//go:build !ui
// +build !ui

package main

import (
	assetfs "github.com/elazarl/go-bindata-assetfs"
)

func init() {
	uiEnabled = false
	stubHTML = `<!DOCTYPE html>
<html>
<p>SDS UI is not available in this binary.</p>
</html>
`
}

// assetFS is a stub for building Nomad without a UI.
func assetFS() *assetfs.AssetFS {
	return nil
}
