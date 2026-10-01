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

// GetRDSXYCut handles calculating and returning the X or Y
// cut for the given input
func (a *API) GetRDSXYCut(c echo.Context) error {
	var data []byte
	var inCache bool

	// Get URL Parameters
	// url - /sds/rdsxcut/x1/y1/x2/y2/outxsize/outzsize
	cutType := c.Param("cuttype") // rdsxcut or rdsycut
	locationName := c.Param("location")
	filename := c.Param("*")

	var rdsRequest sds.RdsRequest
	if err := bindRDSRequest(c, &rdsRequest); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	rdsRequest.ApplyBindDefaults()

	// Validate input parameters
	if rdsRequest.X1 < 0 || rdsRequest.X2 < 0 || rdsRequest.Y1 < 0 || rdsRequest.Y2 < 0 {
		return c.String(http.StatusBadRequest, fmt.Sprintf(
			"coordinates must be >= 0: x1=%d, y1=%d, x2=%d, y2=%d",
			rdsRequest.X1, rdsRequest.Y1, rdsRequest.X2, rdsRequest.Y2,
		))
	}
	if err := sds.ValidateOutputSize("outxsize", rdsRequest.Outxsize); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := sds.ValidateOutputSize("outysize", rdsRequest.Outysize); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	rdsRequest.ComputeRequestSizes()

	c.Logger().Debugf("RDS Request: %v", rdsRequest)

	if rdsRequest.Xsize < 1 || rdsRequest.Ysize < 1 {
		err := fmt.Errorf("bad Xsize or ysize. xsize: %d ysize: %d", rdsRequest.Xsize, rdsRequest.Ysize)
		return c.String(http.StatusBadRequest, err.Error())
	}

	if cutType == "rdsxcut" {
		if rdsRequest.Ysize > 1 {
			err := fmt.Errorf("currently only support cut of one y line. ysize: %d", rdsRequest.Ysize)
			return c.String(http.StatusBadRequest, err.Error())
		}
	} else if cutType == "rdsycut" {
		if rdsRequest.Xsize > 1 {
			err := fmt.Errorf("currently only support cut of one x line. xsize: %d", rdsRequest.Xsize)
			return c.String(http.StatusBadRequest, err.Error())
		}
	}

	c.Logger().Info(
		"RDS XY Cut Request params xstart, ystart, xsize, ysize, outxsize, outzsize:",
		cutType,
		rdsRequest.Xstart,
		rdsRequest.Ystart,
		rdsRequest.Xsize,
		rdsRequest.Ysize,
		rdsRequest.Outxsize,
		rdsRequest.Outzsize,
	)

	var err error
	start := time.Now()
	cacheFileName := cache.UrlToCacheFileName(c.Request().URL.String())
	var fileMetaDataJSON []byte
	// Check if request has been previously processed and is in cache. If not process request.
	data, fileMetaDataJSON, inCache = a.getCachedOutput(cacheFileName)
	if !inCache && a.Cfg.UseCache {
		c.Logger().Error("Unable to get data from cache")
	}

	// If the output is not already in the cache then read the data file and do the processing.
	if !inCache {
		c.Logger().Info("RDS request not in cache, computing result")
		rdsRequest.Reader, err = sds.OpenDataSource(a.Cfg, a.Cache, locationName, filename)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		rdsRequest.ReaderMutex = &sync.Mutex{}

		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
			rdsRequest.ProcessBlueFileHeader()
			if rdsRequest.SubsizeSet {
				rdsRequest.FileXSize = rdsRequest.Subsize
			} else {
				if rdsRequest.FileType == 1000 {
					err = fmt.Errorf("for type 1000 files, a subsize needs to be set")
					return c.String(http.StatusBadRequest, err.Error())
				}
			}
			rdsRequest.ComputeYSize()
		} else {
			err := fmt.Errorf("invalid file type: %s", rdsRequest.FileName)
			return c.String(http.StatusBadRequest, err.Error())
		}

		// Check Request against File Size
		if rdsRequest.Xsize > rdsRequest.FileXSize {
			return c.String(http.StatusBadRequest, fmt.Sprintf(
				"requested x size %d greater than file x size %d",
				rdsRequest.Xsize, rdsRequest.FileXSize,
			))
		}
		if rdsRequest.X1 > rdsRequest.FileXSize {
			return c.String(http.StatusBadRequest, fmt.Sprintf(
				"requested x1 %d greater than file x size %d",
				rdsRequest.X1, rdsRequest.FileXSize,
			))
		}
		if rdsRequest.X2 > rdsRequest.FileXSize {
			return c.String(http.StatusBadRequest, fmt.Sprintf(
				"requested x2 %d greater than file x size %d",
				rdsRequest.X2, rdsRequest.FileXSize,
			))
		}
		if rdsRequest.Y1 > rdsRequest.FileYSize {
			return c.String(http.StatusBadRequest, fmt.Sprintf(
				"requested y1 %d greater than file y size %d",
				rdsRequest.Y1, rdsRequest.FileYSize,
			))
		}
		if rdsRequest.Y2 > rdsRequest.FileYSize {
			return c.String(http.StatusBadRequest, fmt.Sprintf(
				"requested y2 %d greater than file y size %d",
				rdsRequest.Y2, rdsRequest.FileYSize,
			))
		}

		//If Zmin and Zmax were not explitily given then compute
		if !rdsRequest.Zset {
			rdsRequest.FindZminMax(a.Cfg.MaxBytesZminZmax)
		}

		if cutType == "rds" {
			data = sds.ProcessRequest(rdsRequest)
		} else {
			// For xcut/ycut, the outysize route param serves as outzsize
			rdsRequest.Outzsize = rdsRequest.Outysize
			data, err = sds.ProcessLineRequest(rdsRequest, cutType)
			if err != nil {
				return c.String(http.StatusBadRequest, err.Error())
			}
		}

		// Store MetaData of request off in cache
		fileMData := sds.FileMetaData{
			Outxsize:   rdsRequest.Outxsize,
			Outysize:   rdsRequest.Outysize,
			Outzsize:   rdsRequest.Outzsize,
			Filexstart: rdsRequest.Filexstart,
			Filexdelta: rdsRequest.Filexdelta,
			Fileystart: rdsRequest.Fileystart,
			Fileydelta: rdsRequest.Fileydelta,
			Xstart:     rdsRequest.Xstart,
			Ystart:     rdsRequest.Ystart,
			Xsize:      rdsRequest.Xsize,
			Ysize:      rdsRequest.Ysize,
			Zmin:       rdsRequest.Zmin,
			Zmax:       rdsRequest.Zmax,
		}

		fileMDataJSON, marshalError := json.Marshal(fileMData)
		if marshalError != nil {
			return marshalError
		}
		if err := a.putCachedOutput(cacheFileName, data, fileMDataJSON); err != nil {
			return err
		}
		fileMetaDataJSON = fileMDataJSON
	}

	elapsed := time.Since(start)
	c.Logger().Infof("Length of output data %d processed in %s", len(data), elapsed.String())

	var fileMDataCache sds.FileMetaData
	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
	if marshalError != nil {
		return marshalError
	}
	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)
	outzsizeStr := strconv.Itoa(fileMDataCache.Outzsize)

	c.Response().Header().Set(
		echo.HeaderAccessControlExposeHeaders,
		"outxsize,outysize,outzsize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax",
	)
	c.Response().Header().Set("outxsize", outxsizeStr)
	c.Response().Header().Set("outysize", outysizeStr)
	c.Response().Header().Set("outzsize", outzsizeStr)
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
