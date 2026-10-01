package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/spectriclabs/sigplot-data-service/internal/sds"
)

func testAPI() (*echo.Echo, *API) {
	cfg := config.Config{
		UseCache:         false,
		CacheLocation:    ".api-test-cache/",
		MaxBytesZminZmax: 1000000,
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: "../../tests/data"},
		},
	}
	a := NewSDSAPI(&cfg)
	e := echo.New()
	e.GET("/sds/rdstile/:tileXsize/:tileYsize/:decXMode/:decYMode/:tileX/:tileY/:location/*", a.GetRDSTile)
	e.GET("/sds/:cuttype/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", a.GetRDSXYCut)
	e.GET("/sds/lds/:x1/:x2/:outxsize/:outzsize/:location/*", a.GetLDS)
	return e, a
}

func TestLineHandlersUseCachedData(t *testing.T) {
	cacheDir := ".api-test-cache/"
	if err := os.RemoveAll(cacheDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	cfg := config.Config{
		UseCache:      true,
		CacheLocation: cacheDir,
		LocationDetails: []config.Location{
			{LocationName: "MissingDir", LocationType: "localFile", Path: "does-not-exist"},
		},
	}
	a := NewSDSAPI(&cfg)
	e := echo.New()
	e.GET("/sds/:cuttype/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", a.GetRDSXYCut)
	e.GET("/sds/lds/:x1/:x2/:outxsize/:outzsize/:location/*", a.GetLDS)

	meta, err := json.Marshal(sds.FileMetaData{Outxsize: 2, Outysize: 2, Outzsize: 2})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"/sds/rdsxcut/0/0/1/1/2/2/MissingDir/missing.tmp",
		"/sds/lds/0/1/2/2/MissingDir/missing.tmp",
	} {
		key := cache.UrlToCacheFileName(path)
		if err := a.Cache.PutItemInCache(key, "outputFiles/", []byte{1, 2, 3, 4}); err != nil {
			t.Fatal(err)
		}
		if err := a.Cache.PutItemInCache(key+"meta", "outputFiles/", meta); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d; body=%s", path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

func TestRDSRecomputesWhenCachedMetadataMissing(t *testing.T) {
	cacheDir := ".api-test-cache-missing-meta/"
	if err := os.RemoveAll(cacheDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cacheDir) })

	cfg := config.Config{
		UseCache:         true,
		CacheLocation:    cacheDir,
		MaxBytesZminZmax: 1000000,
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: "../../tests/data"},
		},
	}
	a := NewSDSAPI(&cfg)
	e := echo.New()
	e.GET("/sds/:cuttype/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", a.GetRDSXYCut)

	path := "/sds/rds/0/0/10/10/10/10/TestDir/mydata_SB_60_60.tmp?outfmt=SB"
	key := cache.UrlToCacheFileName(path)
	if err := a.Cache.PutItemInCache(key, "outputFiles/", []byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("outxsize"); got != "10" {
		t.Fatalf("outxsize header = %q, want 10", got)
	}
}

func TestOutputSizeCap(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "rds",
			path: "/sds/rds/0/0/10/10/" + strconv.Itoa(sds.MaxOutputSize+1) + "/10/TestDir/mydata_SB_60_60.tmp",
		},
		{
			name: "xcut",
			path: "/sds/rdsxcut/0/0/10/1/10/" + strconv.Itoa(sds.MaxOutputSize+1) + "/TestDir/mydata_SB_60_60.tmp",
		},
		{
			name: "lds",
			path: "/sds/lds/0/10/" + strconv.Itoa(sds.MaxOutputSize+1) + "/10/TestDir/stairstep.tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := testAPI()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestZRangeRequiresBothQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantZmax string
	}{
		{name: "only zmin computes range", query: "?zmin=0&outfmt=SB", wantZmax: "10.000000"},
		{name: "both zmin zmax use explicit range", query: "?zmin=0&zmax=0&outfmt=SB", wantZmax: "0.000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := testAPI()
			req := httptest.NewRequest(http.MethodGet, "/sds/rds/0/0/10/10/10/10/TestDir/mydata_SB_60_60.tmp"+tt.query, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if got := rec.Header().Get("zmax"); got != tt.wantZmax {
				t.Fatalf("zmax = %q, want %q", got, tt.wantZmax)
			}
		})
	}
}

func TestRDSTileQueryParamsKeepDefaults(t *testing.T) {
	e, _ := testAPI()
	for _, query := range []string{"", "?colormap=Greyscale", "?transform=max", "?zmin=1&zmax=5", "?cxmode=Ma"} {
		req := httptest.NewRequest(http.MethodGet, "/sds/rdstile/100/100/1/1/0/0/TestDir/mydata_SB_60_60.tmp"+query, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%q: status %d: %s", query, rec.Code, rec.Body.String())
		}
		outx, _ := strconv.Atoi(rec.Header().Get("outxsize"))
		outy, _ := strconv.Atoi(rec.Header().Get("outysize"))
		if want := outx * outy * 4; want == 0 || rec.Body.Len() != want {
			t.Fatalf("%q: body length %d, want %d RGBA bytes", query, rec.Body.Len(), want)
		}
	}
}
