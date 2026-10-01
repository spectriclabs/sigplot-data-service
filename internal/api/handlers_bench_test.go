package api

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
)

func benchmarkRouter(b *testing.B) *echo.Echo {
	logOutput := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(logOutput) })

	cacheDir := ".api-bench-cache/"
	_ = os.RemoveAll(cacheDir)
	b.Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	cfg := config.Config{
		UseCache:         false,
		CacheLocation:    cacheDir,
		MaxBytesZminZmax: 1000000,
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: "../../tests/data"},
		},
	}
	a := NewSDSAPI(&cfg)
	e := echo.New()
	e.GET("/sds/rdstile/:tileXsize/:tileYsize/:decXMode/:decYMode/:tileX/:tileY/:location/*", a.GetRDSTile)
	e.GET("/sds/:cuttype/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", a.GetRDSXYCut)
	return e
}

func BenchmarkGetRDSTileRouter(b *testing.B) {
	e := benchmarkRouter(b)
	url := "/sds/rdstile/100/100/1/1/0/0/TestDir/mydata_SB_600_600.tmp?outfmt=SB"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
		}
	}
}

func BenchmarkGetRDSYCutRouter(b *testing.B) {
	e := benchmarkRouter(b)
	url := "/sds/rdsycut/20/0/21/60/60/10/TestDir/mydata_SB_60_60.tmp"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
		}
	}
}
