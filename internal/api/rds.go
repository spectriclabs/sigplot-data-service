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

//func HandleRDS(tiled bool) {
//	rds := sds.RdsRequest{TileRequest: tiled}

//	if tiled {
//		ValidateTiled()
//	} else {
//		ValidateNotTiled()
//	}
//}

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
		OutputFmt: "RGBA",
	}

	if err := c.Bind(&tileRequest); err != nil {
		return err
	}

	// Extract URL Parameters
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
			fmt.Sprintf("tileYSize must be one of {100, 200, 300, 400, 500}; given %d", tileRequest.TileXSize),
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
		return fmt.Errorf("bad Xsize or ysize. xsize: %d, ysize: %d", tileRequest.Xsize, tileRequest.Ysize)
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

	var cacheErr error

	// Check if request has been previously processed and is in cache. If not process Request.
	if a.Cfg.UseCache {
		data, cacheErr = a.Cache.GetDataFromCache(cacheFileName, "outputFiles/")
		if cacheErr != nil {
			inCache = false
		} else {
			inCache = true
		}
	} else {
		inCache = false
	}

	// If the output is not already in the cache then read the data file and do the processing.
	if !inCache {
		var openErr error
		c.Logger().Info("RDS Request not in Cache, computing result")
		locationName := c.Param("location")
		tileRequest.Reader, openErr = sds.OpenDataSource(a.Cfg, a.Cache, locationName, tileRequest.FileName)
		if openErr != nil {
			return openErr
		}

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
		if tileRequest.Zmin == 0 && tileRequest.Zmax == 0 {
			tileRequest.FindZminMax(a.Cfg.MaxBytesZminZmax)
		}
		// Now that all the parameters have been computed as needed,
		// perform the actual request for data transformation.
		data = sds.ProcessRequest(tileRequest)
		if a.Cfg.UseCache {
			go a.Cache.PutItemInCache(cacheFileName, "outputFiles/", data)
		}

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
		a.Cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)
	} else {
		c.Logger().Info("Request in cache - returning data from cache")
	}

	elapsed := time.Since(start)
	c.Logger().Infof("Length of Output Data %d processed in %s", len(data), elapsed.String())

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

//func (s *rdsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	var data []byte
//	var inCache bool
//	var ok bool
//	var rdsRequest sds.RdsRequest
//
//	//Get URL Parameters
//	//url - /sds/rds/x1/y1/x2/y2/outxsize/outysize
//	rdsRequest.X1, ok = sds.GetURLArgumentInt(r.URL.Path, 3)
//	if !ok || rdsRequest.X1 < 0 {
//		log.Println("X1 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.Y1, ok = sds.GetURLArgumentInt(r.URL.Path, 4)
//	if !ok || rdsRequest.Y1 < 0 {
//		log.Println("Y1 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.X2, ok = sds.GetURLArgumentInt(r.URL.Path, 5)
//	if !ok || rdsRequest.X2 < 0 {
//		log.Println("X2 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.Y2, ok = sds.GetURLArgumentInt(r.URL.Path, 6)
//	if !ok || rdsRequest.Y2 < 0 {
//		log.Println("Y2 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.Outxsize, ok = sds.GetURLArgumentInt(r.URL.Path, 7)
//	if !ok || rdsRequest.Outxsize < 1 {
//		log.Println("outxsize Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.Outysize, ok = sds.GetURLArgumentInt(r.URL.Path, 8)
//	if !ok || rdsRequest.Outysize < 1 {
//		log.Println("outysize Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.GetQueryParams(r)
//
//	rdsRequest.ComputeRequestSizes()
//
//	if rdsRequest.Xsize < 1 || rdsRequest.Ysize < 1 {
//		log.Println("Bad Xsize or ysize. xsize: ", rdsRequest.Xsize, " ysize: ", rdsRequest.Ysize)
//		w.WriteHeader(400)
//		return
//	}
//
//	log.Println("RDS Request params xstart, ystart, xsize, ysize, outxsize, outysize:", rdsRequest.Xstart, rdsRequest.Ystart, rdsRequest.Xsize, rdsRequest.Ysize, rdsRequest.Outxsize, rdsRequest.Outysize)
//
//	start := time.Now()
//	cacheFileName := cache.UrlToCacheFileName(r.URL.Path, r.URL.RawQuery)
//	// Check if request has been previously processed and is in cache. If not process Request.
//	if *useCache {
//		data, inCache = cache.GetDataFromCache(cacheFileName, "outputFiles/")
//	} else {
//		inCache = false
//	}
//
//	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
//		log.Println("RDS Request not in Cache, computing result")
//		rdsRequest.Reader, rdsRequest.FileName, ok = OpenDataSource(r.URL.Path, 9)
//		if !ok {
//			w.WriteHeader(400)
//			return
//		}
//
//		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
//			rdsRequest.ProcessBlueFileHeader()
//			if rdsRequest.SubsizeSet {
//				rdsRequest.FileXSize = rdsRequest.Subsize
//
//			} else {
//				if rdsRequest.FileType == 1000 {
//					log.Println("For type 1000 files, a subsize needs to be set")
//					w.WriteHeader(400)
//					return
//				}
//			}
//			rdsRequest.ComputeYSize()
//		} else {
//			log.Println("Invalid File Type")
//			w.WriteHeader(400)
//			return
//		}
//
//		if rdsRequest.Xsize > rdsRequest.FileXSize {
//			log.Println("Invalid Request. Requested X size greater than file X size")
//			w.WriteHeader(400)
//			return
//		}
//		if rdsRequest.X1 > rdsRequest.FileXSize {
//			log.Println("Invalid Request. Requested X1 greater than file X size")
//			w.WriteHeader(400)
//			return
//		}
//		if rdsRequest.X2 > rdsRequest.FileXSize {
//			log.Println("Invalid Request. Requested X2 greater than file X size")
//			w.WriteHeader(400)
//			return
//		}
//		if rdsRequest.Y1 > rdsRequest.FileYSize {
//			log.Println("Invalid Request. Requested Y1 greater than file Y size")
//			w.WriteHeader(400)
//			return
//		}
//		if rdsRequest.Y2 > rdsRequest.FileYSize {
//			log.Println("Invalid Request. Requested Y2 greater than file Y size")
//			w.WriteHeader(400)
//			return
//		}
//
//		//If Zmin and Zmax were not explitily given then compute
//		if !rdsRequest.Zset && rdsRequest.OutputFmt == "RGBA" {
//			rdsRequest.FindZminMax()
//		}
//
//		data = ProcessRequest(rdsRequest)
//		if *useCache {
//			go cache.PutItemInCache(cacheFileName, "outputFiles/", data)
//		}
//
//		// Store MetaData of request off in cache
//		fileMetaData := sds.FileMetaData{
//			Outxsize: rdsRequest.Outxsize,
//			Outysize: rdsRequest.Outysize,
//			Filexstart: rdsRequest.Filexstart,
//			Filexdelta: rdsRequest.Filexdelta,
//			Fileystart: rdsRequest.Fileystart,
//			Fileydelta: rdsRequest.Fileydelta,
//			Xstart: rdsRequest.Xstart,
//			Ystart: rdsRequest.Ystart,
//			Xsize: rdsRequest.Xsize,
//			Ysize: rdsRequest.Ysize,
//			Zmin: rdsRequest.Zmin,
//			Zmax: rdsRequest.Zmax,
//		}
//
//		//var marshalError error
//		fileMDataJSON, marshalError := json.Marshal(fileMetaData)
//		if marshalError != nil {
//			log.Println("Error Encoding metadata file to cache", marshalError)
//			w.WriteHeader(400)
//			return
//		}
//		cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)
//
//	}
//
//	elapsed := time.Since(start)
//	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)
//
//	// Get the metadata for this request to put into the return header.
//	fileMetaDataJSON, metaInCache := cache.GetDataFromCache(cacheFileName+"meta", "outputFiles/")
//	if !metaInCache {
//		log.Println("Error reading the metadata file from cache")
//		w.WriteHeader(400)
//		return
//	}
//	var fileMDataCache sds.FileMetaData
//	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
//	if marshalError != nil {
//		log.Println("Error Decoding metadata file from cache", marshalError)
//		w.WriteHeader(400)
//		return
//	}
//
//	// Create a Return header with some metadata in it.
//	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
//	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)
//
//	w.Header().Add("Access-Control-Allow-Origin", "*")
//	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
//	w.Header().Add("outxsize", outxsizeStr)
//	w.Header().Add("outysize", outysizeStr)
//	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
//	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
//	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
//	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
//	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
//	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
//	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
//	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
//	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
//	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
//	w.WriteHeader(http.StatusOK)
//
//	w.Write(data)
//}
