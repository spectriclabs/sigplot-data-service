//go:build ui

package ui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFileServerCacheHeaders(t *testing.T) {
	asset := findDistAsset(t)
	handler := FileServer()

	req := httptest.NewRequest(http.MethodGet, "/sigplot/ui/assets/"+asset, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != uiAssetCacheCtl {
		t.Fatalf("asset Cache-Control = %q, want %q", got, uiAssetCacheCtl)
	}

	req = httptest.NewRequest(http.MethodGet, "/sigplot/ui/", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("index status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); got != uiIndexCacheCtl {
		t.Fatalf("index Cache-Control = %q, want %q", got, uiIndexCacheCtl)
	}
}

func BenchmarkFileServerAsset(b *testing.B) {
	asset := findDistAsset(b)
	handler := FileServer()
	requestPath := "/sigplot/ui/assets/" + asset

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, requestPath, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("asset status = %d, want 200", rr.Code)
		}
	}
}

func findDistAsset(tb testing.TB) string {
	tb.Helper()
	entries, err := fs.ReadDir(embeddedFiles, "webapp/dist/assets")
	if err != nil {
		tb.Fatalf("read dist assets: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
			return entry.Name()
		}
	}
	tb.Fatal("no dist js asset found")
	return ""
}
