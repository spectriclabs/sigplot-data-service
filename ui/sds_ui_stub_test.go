//go:build !ui

package ui

import (
	"net/http/httptest"
	"testing"
)

func TestStubFileSystemDoesNotServeWorkingDirectory(t *testing.T) {
	handler := FileServer()
	for _, path := range []string{
		"/sigplot/ui/sds_config.json",
		"/sigplot/ui/go.mod",
		"/sigplot/ui/",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != 404 {
				t.Fatalf("GET %s status = %d, want 404", path, rr.Code)
			}
		})
	}
}
