package api

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/sds"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GetRDSTile handles retrieving tiles in a WMS-like
// tiling manner.
//
// The URL is of the form:
// /sds/rdstile/tileXSize/tileYSize/decxMode/decYMode/tileX/tileY/locationName
func (a *API) GetRDSTile(c echo.Context) error {
	var data []byte
	var inCache bool

	tileRequest := sds.RdsRequest{
		TileRequest: true,
	}

	if err := bindRDSRequest(c, &tileRequest); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	tileRequest.ApplyBindDefaults()
	allowedTileSizes := [5]int{100, 200, 300, 400, 500}
	if !sds.IntInSlice(tileRequest.TileXSize, allowedTileSizes[:]) {
		return c.String(
			http.StatusBadRequest,
			fmt.Sprintf("tileXSize must be one of {100, 200, 300, 400, 500}; given %d", tileRequest.TileXSize),
		)
	}
	if !sds.IntInSlice(tileRequest.TileYSize, allowedTileSizes[:]) {
		return c.String(
			http.StatusBadRequest,
			fmt.Sprintf("tileYSize must be one of {100, 200, 300, 400, 500}; given %d", tileRequest.TileYSize),
		)
	}
	if tileRequest.DecXMode < 0 || tileRequest.DecXMode > 10 {
		return c.String(
			http.StatusBadRequest,
			fmt.Sprintf("decXMode Bad or out of range 0 to 10. got: %d", tileRequest.DecXMode),
		)
	}
	if tileRequest.DecYMode < 0 || tileRequest.DecYMode > 10 {
		return c.String(
			http.StatusBadRequest,
			fmt.Sprintf("decYMode Bad or out of range 0 to 10. got: %d", tileRequest.DecYMode),
		)
	}
	if tileRequest.TileX < 0 {
		return c.String(http.StatusBadRequest, fmt.Sprintf("tileX must be great than zero"))
	}
	if tileRequest.TileY < 0 {
		return c.String(http.StatusBadRequest, fmt.Sprintf("tileY must be great than zero"))
	}

	tileRequest.ComputeTileSizes()

	if tileRequest.Xsize < 1 || tileRequest.Ysize < 1 {
		return c.String(http.StatusBadRequest, fmt.Sprintf("bad Xsize or ysize. xsize: %d, ysize: %d", tileRequest.Xsize, tileRequest.Ysize))
	}

	c.Logger().Infof(
		"Tile Mode: params xstart=%d, ystart=%d, xsize=%d, ysize=%d, outxsize=%d, outysize=%d",
		tileRequest.Xstart,
		tileRequest.Ystart,
		tileRequest.Xsize,
		tileRequest.Ysize,
		tileRequest.Outxsize,
		tileRequest.Outysize,
	)

	start := time.Now()
	cacheFileName := cache.UrlToCacheFileName(c.Request().URL.String())
	var fileMetaDataJSON []byte

	// Check if request has been previously processed and is in cache. If not process Request.
	data, fileMetaDataJSON, inCache = a.getCachedOutput(cacheFileName)

	// If the output is not already in the cache then read the data file and do the processing.
	if !inCache {
		var openErr error
		c.Logger().Info("RDS Request not in Cache, computing result")
		locationName := c.Param("location")
		tileRequest.Reader, openErr = sds.OpenDataSource(a.Cfg, a.Cache, locationName, tileRequest.FileName)
		if openErr != nil {
			return c.String(http.StatusBadRequest, openErr.Error())
		}
		tileRequest.ReaderMutex = &sync.Mutex{}

		if strings.Contains(tileRequest.FileName, ".tmp") || strings.Contains(tileRequest.FileName, ".prm") {
			tileRequest.ProcessBlueFileHeader()
			if tileRequest.SubsizeSet {
				tileRequest.FileXSize = tileRequest.Subsize
			} else {
				if tileRequest.FileType == 1000 {
					return c.String(http.StatusBadRequest, "for type 1000 files, a subsize needs to be set")
				}
			}
			tileRequest.ComputeYSize()
		} else {
			err := fmt.Errorf("invalid File Type")
			return c.String(http.StatusBadRequest, err.Error())
		}

		if tileRequest.Xstart >= tileRequest.FileXSize || tileRequest.Ystart >= tileRequest.FileYSize {
			err := fmt.Errorf("invalid tile request: xstart=%d, filexsize=%d, ystart=%d, fileysize=%d", tileRequest.Xstart, tileRequest.FileXSize, tileRequest.Ystart, tileRequest.FileYSize)
			return c.String(http.StatusBadRequest, err.Error())
		}

		if (tileRequest.Xstart + tileRequest.Xsize) > tileRequest.FileXSize {
			tileRequest.Xsize = tileRequest.FileXSize - tileRequest.Xstart
			tileRequest.Outxsize = tileRequest.Xsize / tileRequest.DecX
		}
		if (tileRequest.Ystart + tileRequest.Ysize) > tileRequest.FileYSize {
			tileRequest.Ysize = tileRequest.FileYSize - tileRequest.Ystart
			tileRequest.Outysize = tileRequest.Ysize / tileRequest.DecY
		}
		if tileRequest.Xsize > tileRequest.FileXSize {
			err := fmt.Errorf("invalid Request. Requested X size greater than file X size")
			return c.String(http.StatusBadRequest, err.Error())
		}

		//If Zmin and Zmax were not explicitly given then compute
		if !tileRequest.Zset {
			tileRequest.FindZminMax(a.Cfg.MaxBytesZminZmax)
		}
		// Now that all the parameters have been computed as needed,
		// perform the actual request for data transformation.
		data = sds.ProcessRequest(tileRequest)

		// Store MetaData of request off in cache
		fileMData := sds.FileMetaData{
			Outxsize:   tileRequest.Outxsize,
			Outysize:   tileRequest.Outysize,
			Filexstart: tileRequest.Filexstart,
			Filexdelta: tileRequest.Filexdelta,
			Fileystart: tileRequest.Fileystart,
			Fileydelta: tileRequest.Fileydelta,
			Xstart:     tileRequest.Xstart,
			Ystart:     tileRequest.Ystart,
			Xsize:      tileRequest.Xsize,
			Ysize:      tileRequest.Ysize,
			Zmin:       tileRequest.Zmin,
			Zmax:       tileRequest.Zmax,
		}

		fileMDataJSON, marshalError := json.Marshal(fileMData)
		if marshalError != nil {
			return marshalError
		}
		if err := a.putCachedOutput(cacheFileName, data, fileMDataJSON); err != nil {
			return err
		}
		fileMetaDataJSON = fileMDataJSON
	} else {
		c.Logger().Info("Request in cache - returning data from cache")
	}

	elapsed := time.Since(start)
	c.Logger().Infof("Length of Output Data %d processed in %s", len(data), elapsed.String())

	var fileMDataCache sds.FileMetaData
	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
	if marshalError != nil {
		return marshalError
	}

	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)

	c.Response().Header().Set(
		echo.HeaderAccessControlExposeHeaders,
		"outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax",
	)
	c.Response().Header().Set("outxsize", outxsizeStr)
	c.Response().Header().Set("outysize", outysizeStr)
	c.Response().Header().Set("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	c.Response().Header().Set("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	c.Response().Header().Set("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	c.Response().Header().Set("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	c.Response().Header().Set("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	c.Response().Header().Set("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	c.Response().Header().Set("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
	c.Response().Header().Set("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
	c.Response().Header().Set("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
	c.Response().Header().Set("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
	return c.Blob(http.StatusOK, "application/binary", data)
}
