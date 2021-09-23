package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/spectriclabs/sigplot-data-service/internal/api"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/stretchr/testify/assert"
)

var sdsConfigString string = `[{"location_name":"ServiceDir","location_type":"localFile","path":"./"},{"location_name":"ServiceDirData","location_type":"localFile","path":"./data"},{"location_name":"sdsdata","location_type":"localFile","path":"/data/sdsdata/"},{"location_name":"minio","location_type":"minio","minio_bucket":"sdsdata","location":"192.168.1.229:9000","minio_access_key":"minio","minio_secret_key":"miniostorage"}]`

func TestFS(t *testing.T) {
	var locationDetails []config.Location
	err := json.Unmarshal([]byte(sdsConfigString), &locationDetails)
	if err != nil {
		t.Errorf("Error unmarshalling JSON: %v", err)
	}

	sdsConfig := config.Config{
		LocationDetails: locationDetails,
	}

	url := "/sds/fs"
	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("The request could not be created because of: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs")
	a := api.NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileLocations(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, sdsConfigString, strings.TrimSpace(rec.Body.String()))
	}
}

func TestFSDir(t *testing.T) {
	var locationDetails []config.Location
	err := json.Unmarshal([]byte(sdsConfigString), &locationDetails)
	if err != nil {
		t.Errorf("Error unmarshalling JSON: %v", err)
	}

	sdsConfig := config.Config{
		LocationDetails: locationDetails,
	}

	url := "/sds/fs"
	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("The request could not be created because of: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs")
	a := api.NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileLocations(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, sdsConfigString, strings.TrimSpace(rec.Body.String()))
	}
}

func TestFSFile(t *testing.T) {
	var locationDetails []config.Location
	err := json.Unmarshal([]byte(sdsConfigString), &locationDetails)
	if err != nil {
		t.Errorf("Error unmarshalling JSON: %v", err)
	}

	sdsConfig := config.Config{
		LocationDetails: locationDetails,
	}

	url := "/sds/fs"
	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("The request could not be created because of: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs")
	a := api.NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileLocations(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, sdsConfigString, strings.TrimSpace(rec.Body.String()))
	}
}

func TestFSMinio(t *testing.T) {
	var locationDetails []config.Location
	err := json.Unmarshal([]byte(sdsConfigString), &locationDetails)
	if err != nil {
		t.Errorf("Error unmarshalling JSON: %v", err)
	}

	sdsConfig := config.Config{
		LocationDetails: locationDetails,
	}

	url := "/sds/fs"
	e := echo.New()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("The request could not be created because of: %v", err)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/sds/fs")
	a := api.NewSDSAPI(&sdsConfig)

	if assert.NoError(t, a.GetFileLocations(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, sdsConfigString, strings.TrimSpace(rec.Body.String()))
	}
}
