package handlers

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sigplot-data-service/internal/bluefile"
	"sigplot-data-service/internal/cache"
	"sigplot-data-service/internal/config"
	"sigplot-data-service/internal/datasource"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var SPA = map[string]int{
	"S": 1,
	"C": 2,
	"V": 3,
	"Q": 4,
	"M": 9,
	"X": 10,
	"T": 16,
	"U": 1,
	"1": 1,
	"2": 2,
	"3": 3,
	"4": 4,
	"5": 5,
	"6": 6,
	"7": 7,
	"8": 8,
	"9": 9,
}

var BPS = map[string]float64{
	"P": 0.125,
	"A": 1,
	"O": 1,
	"B": 1,
	"I": 2,
	"L": 4,
	"X": 8,
	"F": 4,
	"D": 8,
}

func ServeDataHTTP(
	ctx *fasthttp.RequestCtx,
	logger *zap.Logger,
	configuration config.Configuration,
	location string,
	filename string,
) {
	var data []byte
	var inCache bool
	var fileFormat string
	var fileType int
	var fileXSize int
	var filexstart, filexdelta, fileystart, fileydelta, dataOffset float64
	var fileDataOffset int

	uri := string(ctx.RequestURI())

	// Get Rest of URL Parameters
	x1, x1err := ctx.QueryArgs().GetUint("x1")
	if x1err != nil {
		logger.Error(
			"X1 Missing. Required Field",
			zap.String("uri", uri),
		)
		ctx.Error("X1 Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}
	y1, y1err := ctx.QueryArgs().GetUint("y1")
	if y1err != nil {
		logger.Error(
			"Y1 Missing. Required Field",
			zap.String("uri", uri),
		)
		ctx.Error("Y1 Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}
	x2, x2err := ctx.QueryArgs().GetUint("x2")
	if x2err != nil {
		logger.Error(
			"X2 Missing. Required Field",
			zap.String("uri", uri),
		)
		ctx.Error("X2 Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}
	y2, y2err := ctx.QueryArgs().GetUint("y2")
	if y2err != nil {
		logger.Error(
			"Y2 Missing. Required Field",
			zap.String("uri", uri),
		)
		ctx.Error("Y2 Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}
	ystart := int(math.Min(float64(y1), float64(y2)))
	xstart := int(math.Min(float64(x1), float64(x2)))
	xsize := int(math.Abs(float64(x2) - float64(x1)))
	ysize := int(math.Abs(float64(y2) - float64(y1)))

	if xsize < 1 || ysize < 1 {
		logger.Error(
			"Bad Xsize or ysize",
			zap.String("uri", uri),
			zap.Int("xsize", xsize),
			zap.Int("ysize", ysize),
		)
		ctx.Error("Bad Xsize or Ysize", fasthttp.StatusBadRequest)
		return
	}

	outxsize, outxsizeErr := ctx.QueryArgs().GetUint("outxsize")
	if outxsizeErr != nil {
		logger.Error(
			"outxsize Missing. Required Field.",
			zap.String("uri", uri),
		)
		ctx.Error("outxsize Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}

	outysize, outysizeErr := ctx.QueryArgs().GetUint("outysize")
	if outysizeErr != nil {
		logger.Error(
			"outysize Missing. Required Field",
			zap.String("uri", uri),
		)
		ctx.Error("outysize Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}
	transform := string(ctx.QueryArgs().Peek("transform"))
	if transform == "" {
		logger.Error(
			"transform Missing. Required Field",
			zap.String("uri", uri),
		)
		ctx.Error("transform Missing. Required Field", fasthttp.StatusBadRequest)
		return
	}

	cxmodeSet := true
	cxmode := string(ctx.QueryArgs().Peek("cxmode"))
	if cxmode == "" {
		cxmode = "mag"
		cxmodeSet = false
	}

	zmin, zminErr := ctx.QueryArgs().GetUfloat("zmin")
	if zminErr != nil {
		logger.Warn(
			"Zmin Not Specified. Will estimate from file Selection",
			zap.String("uri", uri),
		)
		zmin = 0
	}

	zmax, zmaxErr := ctx.QueryArgs().GetUfloat("zmax")
	if zmaxErr != nil {
		logger.Warn(
			"Zmax Not Specified. Will estimate from file Selection",
			zap.String("uri", uri),
		)
		zmax = 0
	}

	zset := (zminErr != nil && zmaxErr != nil)
	colorMap := string(ctx.QueryArgs().Peek("colormap"))
	if colorMap == "" {
		logger.Warn(
			"colorMap Not Specified.Defaulting to RampColormap",
			zap.String("uri", uri),
		)
		colorMap = "RampColormap"
	}

	logger.Debug(
		"Collected all parameters",
		zap.String("uri", uri),
		zap.Int("xstart", xstart),
		zap.Int("ystart", ystart),
		zap.Int("xsize", xsize),
		zap.Int("ysize", ysize),
		zap.Int("outxsize", outxsize),
		zap.Int("outysize", outysize),
	)

	start := time.Now()
	queryString := string(ctx.URI().QueryString())

	// Check if request has been previously processed and is in cache. If not process Request.
	cacheFileName := cache.UrlToCacheFileName(
		location,
		filename,
		queryString,
	)
	data, inCache = cache.GetDataFromCache(
		configuration.CacheLocation,
		cacheFileName,
		"outputFiles/",
	)

	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
		reader, fileName, succeed := datasource.OpenDataSource(
			configuration,
			logger,
			location,
			filename,
		)
		if !succeed {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		if strings.HasSuffix(fileName, ".tmp") || strings.HasSuffix(fileName, ".prm") {
			logger.Info("Processing File as bluefile", zap.String("filename", fileName))
			blueheader := bluefile.ReadHeader(reader)
			fileFormat = string(blueheader.Format[:])
			fileType = int(blueheader.File_type)
			fileXSize = int(blueheader.Subsize)
			filexstart = blueheader.Xstart
			filexdelta = blueheader.Xdelta
			fileystart = blueheader.Ystart
			fileydelta = blueheader.Ydelta
			dataOffset = blueheader.Data_start
			fileDataOffset = int(dataOffset)
			if fileType != 2000 {
				logger.Error(
					"Only Supports type 2000 Bluefiles",
					zap.Int("filetype", fileType),
					zap.String("filename", fileName),
				)
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}

		} else if strings.Count(fileName, "_") == 3 {
			logger.Info(
				"Processing file as binary file with metadata in filename with underscores",
				zap.String("filename", fileName),
			)
			fileData := strings.Split(fileName, "_")
			// Need to get these parameters from file metadata
			fileFormat = fileData[1]
			fileDataOffset = 0
			var err error
			fileXSize, err = strconv.Atoi(fileData[2])
			if err != nil {
				logger.Error("Bad xfile size in filename", zap.Error(err))
				fileXSize = 0
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}
		} else {
			logger.Error("Invalid File Type", zap.String("filename", fileName))
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		outputFmt := string(ctx.QueryArgs().Peek("outfmt"))
		if outputFmt == "" {
			logger.Warn(
				"Outformat not specified. Setting equal to input format.",
				zap.String("fileformat", fileFormat),
			)
			outputFmt = fileFormat
		}

		req := ProcessRequest{
			fileFormat,
			fileDataOffset,
			fileXSize,
			xstart,
			ystart,
			xsize,
			ysize,
			outxsize,
			outysize,
			transform,
			cxmode,
			outputFmt,
			zmin,
			zmax,
			zset,
			cxmodeSet,
			colorMap,
		}
		data = HandleProcessRequest(reader, req)
		go cache.PutItemInCache(
			configuration.CacheLocation,
			cacheFileName,
			"outputFiles/",
			data,
		)

		var fileMData config.FileMetaData
		fileMData.Outxsize = outxsize
		fileMData.Outysize = outysize
		fileMData.Filexstart = filexstart
		fileMData.Filexdelta = filexdelta
		fileMData.Fileystart = fileystart
		fileMData.Fileydelta = fileydelta
		if !zset {
			fileMData.Zmin = fileZMin
			fileMData.Zmax = fileZMax
		} else {
			fileMData.Zmin = zmin
			fileMData.Zmax = zmax
		}
		//var marshalError error
		fileMDataJSON, marshalError := json.Marshal(fileMData)
		if marshalError != nil {
			logger.Error(
				"Error encoding metadata file to cache",
				zap.Error(marshalError),
			)
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		cache.PutItemInCache(configuration.CacheLocation, cacheFileName+"meta", "outputFiles/", fileMDataJSON)
	}

	elapsed := time.Since(start)
	logger.Info(
		"Data processed",
		zap.Int("data_len", len(data)),
		zap.Duration("processing_time", elapsed),
	)

	// Get the metadata for this request to put into the return header.
	// TODO: Don't hardcode "outputFiles"; pull from config file
	fileMetaDataJSON, metaInCache := cache.GetDataFromCache(configuration.CacheLocation, cacheFileName+"meta", "outputFiles/")
	if !metaInCache {
		logger.Error(
			"Error reading the metadata file from cache",
			zap.String("cache_filename", cacheFileName+"meta"),
			zap.String("filename", filename),
		)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	var fileMDataCache config.FileMetaData
	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
	if marshalError != nil {
		logger.Error(
			"Error decoding metadata file from cache",
			zap.Error(marshalError),
		)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)

	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Expose-Headers", "*")
	ctx.Response.Header.Set("outxsize", outxsizeStr)
	ctx.Response.Header.Set("outysize", outysizeStr)
	ctx.Response.Header.Set("zmin", fmt.Sprintf("%f", zmin))
	ctx.Response.Header.Set("zmax", fmt.Sprintf("%f", zmax))
	ctx.Response.Header.Set("filexstart", fmt.Sprintf("%f", filexstart))
	ctx.Response.Header.Set("filexdelta", fmt.Sprintf("%f", filexdelta))
	ctx.Response.Header.Set("fileystart", fmt.Sprintf("%f", fileystart))
	ctx.Response.Header.Set("fileydelta", fmt.Sprintf("%f", fileydelta))
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(data)
}

func ServeHeaderHTTP(
	ctx *fasthttp.RequestCtx,
	logger *zap.Logger,
	configuration config.Configuration,
	location string,
	fname string,
) {
	reader, fileName, succeed := datasource.OpenDataSource(configuration, logger, location, fname)
	if !succeed {
		logger.Error(
			"Error reading from data source",
			zap.String("filename", fname),
			zap.String("uri", string(ctx.RequestURI())),
		)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	var bluefileheader bluefile.BlueHeader
	var returnbytes []byte
	if strings.HasSuffix(fileName, ".tmp") || strings.HasSuffix(fileName, ".prm") {
		// Calculated Fields

		logger.Info(
			"Opening file for hdr mode",
			zap.String("filename", fileName),
		)
		binary.Read(reader, binary.LittleEndian, &bluefileheader)
		format := string(bluefileheader.Format[:])
		spa := SPA[string(format[0])]
		bps := BPS[string(format[1])]
		bpa := float64(spa) * bps
		ape := int(bluefileheader.Subsize)
		blueShort := bluefile.BlueHeaderShortenedFields{
			Version:    string(bluefileheader.Version[:]),
			Head_rep:   string(bluefileheader.Head_rep[:]),
			Data_rep:   string(bluefileheader.Data_rep[:]),
			Detached:   bluefileheader.Detached,
			Protected:  bluefileheader.Protected,
			Pipe:       bluefileheader.Pipe,
			Ext_start:  bluefileheader.Ext_start,
			Data_start: bluefileheader.Data_start,
			Data_size:  bluefileheader.Data_size,
			Format:     format,
			Flagmask:   bluefileheader.Flagmask,
			Timecode:   bluefileheader.Timecode,
			Xstart:     bluefileheader.Xstart,
			Xdelta:     bluefileheader.Xdelta,
			Xunits:     bluefileheader.Xunits,
			Subsize:    bluefileheader.Subsize,
			Ystart:     bluefileheader.Ystart,
			Ydelta:     bluefileheader.Ydelta,
			Yunits:     bluefileheader.Yunits,
			Spa:        spa,
			Bps:        bps,
			Bpa:        bpa,
			Ape:        ape,
			Bpe:        float64(ape) * bpa,
			Size:       int(bluefileheader.Data_size / (bpa * float64(ape))),
		}

		var marshalError error
		returnbytes, marshalError = json.Marshal(blueShort)
		if marshalError != nil {
			logger.Error(
				"Problem Marshalling Header to JSON",
				zap.Error(marshalError),
			)
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}

		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Expose-Headers", "*")
		ctx.SetStatusCode(http.StatusOK)
		ctx.SetBody(returnbytes)

		logger.Info("Successfully returning JSON header")
	} else {
		logger.Error(
			"Can only Return Headers for Blue Files. Looking for .tmp or .prm",
			zap.String("filename", fileName),
		)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	}
}

func ServeHTTP(logger *zap.Logger, configuration config.Configuration) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		// Valid url is /sds/<filename>/rds or //Valid url is /sds/<filename>
		filename := ctx.UserValue("filename").(string)
		location := ctx.UserValue("location").(string)
		mode := string(ctx.QueryArgs().Peek("mode"))

		logger.Info(
			"Received request",
			zap.String("uri", string(ctx.RequestURI())),
			zap.String("filename", filename),
			zap.String("location", location),
			zap.String("mode", mode),
		)
		switch mode {
		case "hdr": // Valid url is /sds/<locationName>/<filename>?mode=hdr
			ServeHeaderHTTP(ctx, logger, configuration, location, filename)
		case "rds": // Valid url is /sds/<locationName>/<filename>?mode=rds
			ServeDataHTTP(ctx, logger, configuration, location, filename)
		case "":
			logger.Error(
				"Required parameter 'mode' missing",
				zap.String("mode", mode),
			)
			ctx.Error("Required parameter 'mode' missing", fasthttp.StatusBadRequest)
		default:
			errorString := fmt.Sprintf("Unknown mode=%s", mode)
			logger.Error(
				"Unknown mode",
				zap.String("mode", mode),
			)
			ctx.Error(errorString, fasthttp.StatusBadRequest)
		}
	}
}
