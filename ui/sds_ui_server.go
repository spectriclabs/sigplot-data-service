package ui

import (
	"net/http"
	"strings"
)

const (
	uiRoutePrefix   = "/sigplot/ui/"
	uiAssetsPrefix  = uiRoutePrefix + "assets/"
	uiIndexCacheCtl = "no-cache"
	uiAssetCacheCtl = "public, max-age=31536000, immutable"
)

func FileServer() http.Handler {
	fileServer := http.StripPrefix(uiRoutePrefix, http.FileServer(GetFileSystem()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == uiRoutePrefix || r.URL.Path == uiRoutePrefix+"index.html":
			SetIndexHeaders(w)
		case strings.HasPrefix(r.URL.Path, uiAssetsPrefix):
			w.Header().Set("Cache-Control", uiAssetCacheCtl)
		}
		fileServer.ServeHTTP(w, r)
	})
}

func SetIndexHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", uiIndexCacheCtl)
}
