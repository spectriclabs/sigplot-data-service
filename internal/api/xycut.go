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
	if err := c.Bind(rdsRequest); err != nil {
		return err
	}

	rdsRequest.ComputeRequestSizes()

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
	// Check if request has been previously processed and is in cache. If not process request.
	if a.Cfg.UseCache {
		data, err = a.Cache.GetDataFromCache(cacheFileName, "outputFiles/")
		if err != nil {
			c.Logger().Error("Unable to get data from cache")
			inCache = false
		}
	} else {
		inCache = false
	}

	// If the output is not already in the cache then read the data file and do the processing.
	if !inCache {
		c.Logger().Info("RDS request not in cache, computing result")
		rdsRequest.Reader, err = sds.OpenDataSource(a.Cfg, a.Cache, locationName, filename)
		if err != nil {
			return err
		}

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
			err := fmt.Errorf("invalid file type")
			return c.String(http.StatusBadRequest, err.Error())
		}

		// Check Request against File Size
		if rdsRequest.Xsize > rdsRequest.FileXSize {
			err := fmt.Errorf(
				"requested x size %d greater than file x size %d",
				rdsRequest.Xsize,
				rdsRequest.FileXSize,
			)
			return err
		}
		if rdsRequest.X1 > rdsRequest.FileXSize {
			err := fmt.Errorf(
				"requested x1 %d greater than file x size %d",
				rdsRequest.X1,
				rdsRequest.FileXSize,
			)
			return err
		}
		if rdsRequest.X2 > rdsRequest.FileXSize {
			err := fmt.Errorf(
				"requested x2 %d greater than file x size %d",
				rdsRequest.X2,
				rdsRequest.FileXSize,
			)
			return err
		}
		if rdsRequest.Y1 > rdsRequest.FileYSize {
			err := fmt.Errorf(
				"requested y1 %d greater than file y size %d",
				rdsRequest.Y1,
				rdsRequest.FileYSize,
			)
			return err
		}
		if rdsRequest.Y2 > rdsRequest.FileYSize {
			err := fmt.Errorf(
				"requested y2 %d greater than file y size %d",
				rdsRequest.Y2,
				rdsRequest.FileYSize,
			)
			return err
		}

		//If Zmin and Zmax were not explitily given then compute
		if !rdsRequest.Zset {
			rdsRequest.FindZminMax(a.Cfg.MaxBytesZminZmax)
		}

		data = sds.ProcessLineRequest(rdsRequest, cutType)
		if a.Cfg.UseCache {
			go a.Cache.PutItemInCache(cacheFileName, "outputFiles/", data)
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
		a.Cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)
	}

	elapsed := time.Since(start)
	c.Logger().Infof("Length of output data %d processed in %lf sec", len(data), elapsed)

	// Get the metadata for this request to put into the return header.
	fileMetaDataJSON, metaCacheErr := a.Cache.GetDataFromCache(cacheFileName+"meta", "outputFiles/")
	if metaCacheErr != nil {
		return metaCacheErr
	}
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
