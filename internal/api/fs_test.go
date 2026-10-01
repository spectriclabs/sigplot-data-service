package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/stretchr/testify/assert"
)

var sdsConfigString string = `[{"location_name":"ServiceDir","location_type":"localFile","path":"./"},{"location_name":"ServiceDirData","location_type":"localFile","path":"./data"},{"location_name":"sdsdata","location_type":"localFile","path":"/data/sdsdata/"},{"location_name":"minio","location_type":"minio","minio_bucket":"sdsdata","location":"192.168.1.229:9000","minio_access_key":"minio","minio_secret_key":"miniostorage"}]`

func TestFS(t *testing.T) {
	var locationDetails []config.Location
	err := json.Unmarshal([]byte(sdsConfigString), &locationDetails)
	if err != nil {
		t.Fatalf("Error unmarshalling JSON: %v", err)
	}

	sdsConfig := config.Config{
		LocationDetails: locationDetails,
	}

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, "/sds/fs", nil)
	if err != nil {
		t.Fatalf("The request could not be created because of: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs")
	a := NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileLocations(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "minio_access_key")
		assert.NotContains(t, rec.Body.String(), "minio_secret_key")
		assert.Contains(t, rec.Body.String(), `"location_name":"minio"`)
		assert.Contains(t, rec.Body.String(), `"location_type":"minio"`)
	}
}

func TestFSDir(t *testing.T) {
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: "."},
		},
	}

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, "/sds/fs/TestDir/", nil)
	if err != nil {
		t.Fatalf("The request could not be created: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs/:location/*")
	c.SetParamNames("location", "*")
	c.SetParamValues("TestDir", "")
	a := NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileOrDirectory(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "filename")
	}
}

func TestFSFile(t *testing.T) {
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: "."},
		},
	}

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, "/sds/fs/TestDir/fs_test.go", nil)
	if err != nil {
		t.Fatalf("The request could not be created: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs/:location/*")
	c.SetParamNames("location", "*")
	c.SetParamValues("TestDir", "fs_test.go")
	a := NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileOrDirectory(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, rec.Body.Len() > 0, "File contents should not be empty")
	}
}

func TestFSMinio(t *testing.T) {
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "minio", LocationType: "minio", MinioBucket: "sdsdata"},
		},
	}

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, "/sds/fs/minio/", nil)
	if err != nil {
		t.Fatalf("The request could not be created: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs/:location/*")
	c.SetParamNames("location", "*")
	c.SetParamValues("minio", "")
	a := NewSDSAPI(&sdsConfig)

	a.GetFileOrDirectory(c)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestFSLocalTraversalRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.tmp"), []byte("ok"), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	tests := []string{
		"../fs.go",
		"dir/../../fs.go",
		"/etc/hosts",
		`dir\..\fs.go`,
	}
	for _, filePath := range tests {
		t.Run(filePath, func(t *testing.T) {
			sdsConfig := config.Config{
				LocationDetails: []config.Location{
					{LocationName: "TestDir", LocationType: "localFile", Path: root},
				},
			}

			e := echo.New()
			req, err := http.NewRequest(http.MethodGet, "/sds/fs/TestDir/"+filePath, nil)
			if err != nil {
				t.Fatalf("The request could not be created: %v", err)
			}

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/sds/fs/:location/*")
			c.SetParamNames("location", "*")
			c.SetParamValues("TestDir", filePath)
			a := NewSDSAPI(&sdsConfig)

			if assert.NoError(t, a.GetFileOrDirectory(c)) {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			}
		})
	}
}

func TestFSMinioTraversalRejectedBeforeClient(t *testing.T) {
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "minio", LocationType: "minio", MinioBucket: "sdsdata"},
		},
	}

	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, "/sds/fs/minio/../secret", nil)
	if err != nil {
		t.Fatalf("The request could not be created: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs/:location/*")
	c.SetParamNames("location", "*")
	c.SetParamValues("minio", "../secret")
	a := NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileOrDirectory(c)) {
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}
}

func TestGetFileContentsLocalTraversalRejected(t *testing.T) {
	root := t.TempDir()
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: root},
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/sds/raw/TestDir/../secret", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	a := NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileContents(c, "TestDir", "../secret")) {
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}
}

func BenchmarkGetFileOrDirectoryLocalListing(b *testing.B) {
	prevLog := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(prevLog)

	root := b.TempDir()
	for i := 0; i < 32; i++ {
		name := filepath.Join(root, "file"+string(rune('a'+i%26))+".tmp")
		if err := os.WriteFile(name, []byte("ok"), 0644); err != nil {
			b.Fatal(err)
		}
	}
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: root},
		},
	}
	a := NewSDSAPI(&sdsConfig)
	e := echo.New()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/sds/fs/TestDir/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/sds/fs/:location/*")
		c.SetParamNames("location", "*")
		c.SetParamValues("TestDir", "")
		if err := a.GetFileOrDirectory(c); err != nil {
			b.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d", rec.Code)
		}
	}
}

func BenchmarkGetFileContentsLocalFile(b *testing.B) {
	prevLog := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(prevLog)

	root := b.TempDir()
	data := []byte("blue data")
	if err := os.WriteFile(filepath.Join(root, "file.tmp"), data, 0644); err != nil {
		b.Fatal(err)
	}
	sdsConfig := config.Config{
		LocationDetails: []config.Location{
			{LocationName: "TestDir", LocationType: "localFile", Path: root},
		},
	}
	a := NewSDSAPI(&sdsConfig)
	e := echo.New()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/sds/raw/TestDir/file.tmp", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		if err := a.GetFileContents(c, "TestDir", "file.tmp"); err != nil {
			b.Fatal(err)
		}
		if rec.Code != http.StatusOK {
			b.Fatalf("status = %d", rec.Code)
		}
	}
}
