package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spectriclabs/sigplot-data-service/internal/bluefile"
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/image"
	"github.com/spectriclabs/sigplot-data-service/internal/sds"
	"github.com/spectriclabs/sigplot-data-service/ui"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	flag "github.com/spf13/pflag"
	"github.com/tkanos/gonfig"

	"strings"
	"time"
)

func ProcessRequest(dataRequest sds.RdsRequest) []byte {
	var processedData []float64

	yLinesPerOutput := float64(dataRequest.Ysize) / float64(dataRequest.Outysize)
	yLinesPerOutputCeil := int(math.Ceil(yLinesPerOutput))
	log.Println("ProcessRequest:", dataRequest.FileXSize, dataRequest.Xstart, dataRequest.Ystart, dataRequest.Xsize, dataRequest.Ysize, dataRequest.Outxsize, dataRequest.Outysize)
	// Loop over the output Y Lines
	for outputLine := 0; outputLine < dataRequest.Outysize; outputLine++ {
		//log.Println("Processing Output Line ", outputLine)
		// For Each Output Y line Read and process the required lines from the input file
		var startLine int
		var endLine int
		if yLinesPerOutput > 1 { // Y Compression is needed.
			if outputLine != dataRequest.Outysize-1 { //Not the last output line of file
				startLine = dataRequest.Ystart + int(math.Round(float64(outputLine)*yLinesPerOutput))
				endLine = dataRequest.Ystart + int(math.Round(float64(outputLine+1)*yLinesPerOutput))
			} else { // Last outputline, work backwards from last line.
				endLine = dataRequest.Ystart + dataRequest.Ysize
				startLine = endLine - yLinesPerOutputCeil
			}
		} else { // Y expansion
			startLine = dataRequest.Ystart + int(math.Round(float64(outputLine)*yLinesPerOutput))
			endLine = startLine + 1
			if endLine > (dataRequest.Ystart + dataRequest.Ysize - 1) { // Last outputlines, work backwards from last line.
				endLine = dataRequest.Ystart + dataRequest.Ysize
				startLine = endLine - 1
			}
		}
		// Number of y lines that will be processed this time through the loop
		numLines := endLine - startLine

		// Make channels to collect the data from processing all the lines in parallel.
		//var chans [100]chan []float64
		chans := make([]chan []float64, numLines)
		for i := range chans {
			chans[i] = make(chan []float64)
		}
		xThinData := make([]float64, numLines*dataRequest.Outxsize)
		//log.Println("Going to Process Input Lines", startLine, endLine)

		done := make(chan bool, 1)
		// Launch the processing of each line concurrently and put the data into a set of channels
		for inputLine := startLine; inputLine < endLine; inputLine++ {
			var lineRequest sds.RdsRequest
			lineRequest = dataRequest
			lineRequest.Ystart = inputLine
			go sds.ProcessLine(xThinData, inputLine-startLine, done, lineRequest)

		}
		//Wait until all the lines have finished before moving on
		for i := 0; i < numLines; i++ {
			<-done
		}

		// for i := 0; i < len(xThinData); i++ {
		// 	if math.IsNaN(xThinData[i]) {
		// 		log.Println("processedDataNaN", outputLine, i)
		// 	}
		// }
		// Thin in y direction the subsset of lines that have now been processed in x
		yThinData := image.DownSampleLineInY(xThinData, dataRequest.Outxsize, dataRequest.Transform)
		//log.Println("Thin Y data is currently ", len(yThinData))

		// for i := 0; i < len(yThinData); i++ {
		// 	if math.IsNaN(yThinData[i]) {
		// 		log.Println("processedDataNaN", outputLine, i)
		// 	}
		// }

		processedData = append(processedData, yThinData...)
		//log.Println("processedData is currently ", len(processedData))

		// for i := 0; i < len(processedData); i++ {
		// 	if math.IsNaN(processedData[i]) {
		// 		log.Println("processedDataNaN", outputLine, i)
		// 	}
		// }

	}

	outData := image.CreateOutput(processedData, dataRequest.OutputFmt, dataRequest.Zmin, dataRequest.Zmax, dataRequest.ColorMap)
	return outData
}

func processLineRequest(dataRequest sds.RdsRequest, cutType string) []byte {
	bytesPerAtom, complexFlag := bluefile.GetFileTypeInfo(dataRequest.FileFormat)

	bytesPerElement := bytesPerAtom
	if complexFlag {
		bytesPerElement = bytesPerElement * 2
	}

	// Get the slice data out of the file. For x the data is continuous, for y cuts, we need to grab one element from each row.
	filedata := make([]byte, 0, int(math.Max(float64(dataRequest.FileXSize), float64(dataRequest.FileYSize))))
	var dataToProcess []float64
	if cutType == "rdsxcut" || cutType == "lds" {
		firstDataByte := float64(dataRequest.Ystart*dataRequest.FileXSize+dataRequest.Xstart) * bytesPerElement
		firstByteInt := int(math.Floor(firstDataByte))
		bytesLength := float64(dataRequest.Xsize)*bytesPerElement + (firstDataByte - float64(firstByteInt))
		bytesLengthInt := int(math.Ceil(bytesLength))
		filedata, _ = sds.GetBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt)
		dataToProcess = bluefile.ConvertFileData(filedata, dataRequest.FileFormat)
		//If the data is SP then we might have processed a few more bits than we actually needed on both sides, so reassign data_to_process to correctly point to the numbers of interest
		if bytesPerAtom < 1 {
			dataStartBit := int(math.Mod(firstDataByte, 1) * 8)
			dataEndBit := int(math.Mod(bytesLength, 1) * 8)
			extraBits := 0
			if dataEndBit > 0 {
				extraBits = 8 - dataEndBit
			}
			dataToProcess = dataToProcess[dataStartBit : len(dataToProcess)-extraBits]
		}

	} else if cutType == "rdsycut" {
		log.Println("Getting data from file for y cut")
		if bytesPerAtom < 1 {
			log.Println("Don't support y cut for SP data")
			var empty []byte
			return empty
		}
		for row := dataRequest.Ystart; row < (dataRequest.Ystart + dataRequest.Ysize); row++ {
			dataByte := float64(row*dataRequest.FileXSize+dataRequest.Xstart) * bytesPerElement
			dataByteInt := int(math.Floor(dataByte))
			data, _ := sds.GetBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+dataByteInt, int(bytesPerElement))
			filedata = append(filedata, data...)
		}
		dataToProcess = bluefile.ConvertFileData(filedata, dataRequest.FileFormat)
		log.Println("Got data from file for y cut", len(dataToProcess))

	}

	var realData []float64
	if complexFlag {
		realData = image.ApplyCXmode(dataToProcess, dataRequest.Cxmode, true)
	} else {
		if dataRequest.CxmodeSet {
			realData = image.ApplyCXmode(dataToProcess, dataRequest.Cxmode, false)
		} else {
			realData = dataToProcess
		}

	}

	//Output data will be x and z data of variable length up to Xsize. Allocation with size 0 but with a capacity. The x arrary will be used for both piece of data at the end.
	xThinData := make([]int16, 0, len(realData)*2)
	zThinData := make([]int16, 0, len(realData))

	xratio := float64(len(realData)) / float64(dataRequest.Outxsize-1)
	zratio := (dataRequest.Zmax - dataRequest.Zmin) / float64(dataRequest.Outzsize-1)
	for x := 0; x < len(realData); x++ {

		xpixel := int16(math.Round(float64(x) / xratio))
		zpixel := int16(math.Round((dataRequest.Zmax - float64(realData[x])) / zratio))

		// If the thinned array does not already have a value in it then append this value.
		if len(xThinData) >= 1 {
			//If this value is not duplicate to the last then append it.
			if !(xThinData[len(xThinData)-1] == xpixel && zThinData[len(zThinData)-1] == zpixel) {
				//log.Println("Adding Pixel", xpixel, zpixel)
				xThinData = append(xThinData, xpixel)
				zThinData = append(zThinData, zpixel)
			}

		} else {
			log.Println("Adding Pixel  1", xpixel, zpixel)
			xThinData = append(xThinData, xpixel)
			zThinData = append(zThinData, zpixel)
		}

	}
	// Return the data as bytes with x values followed by z values.
	xThinData = append(xThinData, zThinData...)
	outData := new(bytes.Buffer)

	_ = binary.Write(outData, binary.LittleEndian, &xThinData)
	return outData.Bytes()
}

func openDataSource(locationName string, filePath string) (io.ReadSeeker, error) {
	var currentLocation sds.Location
	for i := range sds.Config.LocationDetails {
		if sds.Config.LocationDetails[i].LocationName == locationName {
			currentLocation = sds.Config.LocationDetails[i]
		}
	}
	if len(currentLocation.Path) > 0 {
		if string(currentLocation.Path[len(currentLocation.Path)-1]) != "/" {
			currentLocation.Path += "/"
		}
	}
	switch currentLocation.LocationType {
	case "localFile":
		fullFilepath := fmt.Sprintf("%s%s", currentLocation.Path, filePath)
		log.Println("Reading Local File. LocationName=", locationName, "fullPath=", fullFilepath)
		file, err := os.Open(fullFilepath)
		if err != nil {
			log.Println("Error opening File,", err)
			return nil, err
		}
		reader := io.ReadSeeker(file)
		return reader, nil
	case "minio":
		start := time.Now()
		fullFilepath := fmt.Sprintf("%s%s%s", currentLocation.Path, filePath)
		cacheFileName := cache.UrlToCacheFileName(fmt.Sprintf("sds_%s%s", currentLocation.MinioBucket, fullFilepath))
		file, inCache := cache.GetItemFromCache(cacheFileName, "miniocache/")
		if !inCache {
			log.Println("Minio File not in local file Cache, Need to fetch")
			minioClient, err := minio.New(
				currentLocation.Location,
				&minio.Options{
					Creds:  credentials.NewStaticV4(currentLocation.MinioAccessKey, currentLocation.MinioSecretKey, ""),
					Secure: false,
				},
			)
			elapsed := time.Since(start)
			log.Println(" Time to Make connection ", elapsed)
			if err != nil {
				log.Println("Error Establishing Connection to Minio", err)
				return nil, err
			}

			start = time.Now()
			ctx := context.Background()
			object, err := minioClient.GetObject(ctx, currentLocation.MinioBucket, fullFilepath, minio.GetObjectOptions{})

			fi, _ := object.Stat()
			fileData := make([]byte, fi.Size)
			//var readerr error
			numRead, readerr := object.Read(fileData)
			if int64(numRead) != fi.Size || !(readerr == nil || readerr == io.EOF) {
				log.Println("Error Reading File from from Minio", readerr)
				log.Println("Expected Bytes: ", fi.Size, "Got Bytes", numRead)
				return nil, err
			}

			cache.PutItemInCache(cacheFileName, "miniocache/", fileData)
			cacheFileFullpath := fmt.Sprintf("%s%s%s", sds.Config.CacheLocation, "miniocache/", cacheFileName)
			file, err = os.Open(cacheFileFullpath)
			if err != nil {
				log.Println("Error opening Minio Cache File,", err)
				return nil, err
			}
		}
		reader := io.ReadSeeker(file)
		elapsed := time.Since(start)
		log.Println(" Time to Get Minio File ", elapsed)

		return reader, nil

	default:
		log.Println()
		return nil, fmt.Errorf("unsupported Location Type %s in %s", currentLocation.LocationType, currentLocation.LocationName)
	}

}

type rdsServer struct{}

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
//		rdsRequest.Reader, rdsRequest.FileName, ok = openDataSource(r.URL.Path, 9)
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

func GetRDSTile(c echo.Context) error {
	var data []byte
	var inCache bool

	tileRequest := sds.RdsRequest{
		TileRequest: true,
	}

	var err error

	// Extract URL Parameters
	// URL form: /sds/rdstile/tileXSize/tileYSize/decxMode/decYMode/tileX/tileY/locationName
	allowedTileSizes := [5]int{100, 200, 300, 400, 500}
	tileRequest.TileXSize, err = strconv.Atoi(c.Param("tileXsize"))
	if err != nil || !sds.IntInSlice(tileRequest.TileXSize, allowedTileSizes[:]) {
		err := fmt.Errorf("tileXSize must be one of {100, 200, 300, 400, 500}; given %d", tileRequest.TileXSize)
		return c.String(http.StatusBadRequest, err.Error())
	}
	tileRequest.TileYSize, err = strconv.Atoi(c.Param("tileYsize"))
	if err != nil || !sds.IntInSlice(tileRequest.TileYSize, allowedTileSizes[:]) {
		err := fmt.Errorf("tileYSize must be one of {100, 200, 300, 400, 500}; given %d", tileRequest.TileXSize)
		return c.String(http.StatusBadRequest, err.Error())
	}
	tileRequest.DecXMode, err = strconv.Atoi(c.Param("decXmode"))
	if err != nil || tileRequest.DecXMode < 0 || tileRequest.DecXMode > 10 {
		err := fmt.Errorf("decXMode Bad or out of range 0 to 10. got: %d", tileRequest.DecXMode)
		return c.String(http.StatusBadRequest, err.Error())
	}
	tileRequest.DecYMode, err = strconv.Atoi(c.Param("decYmode"))
	if err != nil || tileRequest.DecYMode < 0 || tileRequest.DecYMode > 10 {
		err := fmt.Errorf("decYMode Bad or out of range 0 to 10. got: %d", tileRequest.DecYMode)
		return c.String(http.StatusBadRequest, err.Error())
	}
	tileRequest.TileX, err = strconv.Atoi(c.Param("tileX"))
	if err != nil || tileRequest.TileX < 0 {
		err := fmt.Errorf("tileX must be great than zero")
		return c.String(http.StatusBadRequest, err.Error())
	}
	tileRequest.TileY, err = strconv.Atoi(c.Param("tileY"))
	if err != nil || tileRequest.TileY < 0 {
		err := fmt.Errorf("tileY must be great than zero")
		return c.String(http.StatusBadRequest, err.Error())
	}

	tileRequest.GetQueryParams(r)

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
	// Check if request has been previously processed and is in cache. If not process Request.
	if *useCache {
		data, inCache = cache.GetDataFromCache(cacheFileName, "outputFiles/")
	} else {
		inCache = false
	}

	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
		c.Logger().Info("RDS Request not in Cache, computing result")
		locationName := c.Param("location")
		tileRequest.FileName = c.Param("*")
		tileRequest.Reader, err = openDataSource(locationName, tileRequest.FileName)
		if err != nil {
			return err
		}

		if strings.Contains(tileRequest.FileName, ".tmp") || strings.Contains(tileRequest.FileName, ".prm") {
			tileRequest.ProcessBlueFileHeader()

			if tileRequest.SubsizeSet {
				tileRequest.FileXSize = tileRequest.Subsize

			} else {
				if tileRequest.FileType == 1000 {
					err = fmt.Errorf("for type 1000 files, a subsize needs to be set")
					return c.String(http.StatusBadRequest, err.Error())
				}
			}
			tileRequest.ComputeYSize()
		} else {
			err = fmt.Errorf("invalid File Type")
			return c.String(http.StatusBadRequest, err.Error())
		}

		if tileRequest.Xstart >= tileRequest.FileXSize || tileRequest.Ystart >= tileRequest.FileYSize {
			err = fmt.Errorf("invalid tile request: xstart=%d, filexsize=%d, ystart=%d, fileysize=%d", tileRequest.Xstart, tileRequest.FileXSize, tileRequest.Ystart, tileRequest.FileYSize)
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
			err = fmt.Errorf("invalid Request. Requested X size greater than file X size")
			return c.String(http.StatusBadRequest, err.Error())
		}

		//If Zmin and Zmax were not explitily given then compute
		if !tileRequest.Zset {
			tileRequest.FindZminMax()
		}
		// Now that all the parameters have been computed as needed, perform the actual request for data transformation.
		data = ProcessRequest(tileRequest)
		if *useCache {
			go cache.PutItemInCache(cacheFileName, "outputFiles/", data)
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
		cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)
	} else {
		c.Logger().Info("Request in cache - returning data from cache")
	}

	elapsed := time.Since(start)
	c.Logger().Infof("Length of Output Data %d processed in %lf sec", len(data), elapsed)

	// Get the metadata for this request to put into the return header.
	fileMetaDataJSON, metaInCache := cache.GetDataFromCache(cacheFileName+"meta", "outputFiles/")
	if !metaInCache {
		return fmt.Errorf("error reading the metadata file from cache")
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

type ldsServer struct{}

//func (s *ldsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	var data []byte
//	var inCache bool
//	var ok bool
//	var rdsRequest sds.RdsRequest
//
//	//Get URL Parameters
//	//url - /sds/lds/x1/x2/outxsize/outzsize
//
//	rdsRequest.X1, ok = sds.GetURLArgumentInt(r.URL.Path, 3)
//	if !ok || rdsRequest.X1 < 0 {
//		log.Println("X1 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.X2, ok = sds.GetURLArgumentInt(r.URL.Path, 4)
//	if !ok || rdsRequest.X2 < 0 {
//		log.Println("X2 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.Outxsize, ok = sds.GetURLArgumentInt(r.URL.Path, 5)
//	if !ok || rdsRequest.Outxsize < 1 {
//		log.Println("outxsize Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.Outzsize, ok = sds.GetURLArgumentInt(r.URL.Path, 6)
//	if !ok || rdsRequest.Outzsize < 1 {
//		log.Println("outzsize Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.GetQueryParams(r)
//
//	rdsRequest.ComputeRequestSizes()
//
//	rdsRequest.Ystart = 0
//	rdsRequest.Ysize = 1
//
//	if rdsRequest.Xsize < 1 {
//		log.Println("Bad Xsize: ", rdsRequest.Xsize)
//		w.WriteHeader(400)
//		return
//	}
//
//	log.Println("LDS Request params xstart, xsize, outxsize, outzsize:", rdsRequest.Xstart, rdsRequest.Xsize, rdsRequest.Outxsize, rdsRequest.Outzsize)
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
//		rdsRequest.Reader, rdsRequest.FileName, ok = openDataSource(r.URL.Path, 7)
//		if !ok {
//			w.WriteHeader(400)
//			return
//		}
//
//		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
//			rdsRequest.ProcessBlueFileHeader()
//			if rdsRequest.FileType != 1000 {
//				log.Println("Line Plots only support Type 100 files.")
//				w.WriteHeader(400)
//				return
//			}
//			rdsRequest.FileXSize = int(float64(rdsRequest.FileDataSize) / bluefile.BytesPerAtomMap[string(rdsRequest.FileFormat[1])])
//			rdsRequest.FileYSize = 1
//		} else {
//			log.Println("Invalid File Type")
//			w.WriteHeader(400)
//			return
//		}
//		// Check Request against File Size
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
//
//		//If Zmin and Zmax were not explitily given then compute
//		if !rdsRequest.Zset {
//			rdsRequest.FindZminMax()
//		}
//
//		data = processLineRequest(rdsRequest, "lds")
//
//		if *useCache {
//			go cache.PutItemInCache(cacheFileName, "outputFiles/", data)
//		}
//
//		// Store MetaData of request off in cache
//		var fileMData sds.FileMetaData
//		fileMData.Outxsize = rdsRequest.Outxsize
//		fileMData.Outysize = rdsRequest.Outysize
//		fileMData.Outzsize = rdsRequest.Outzsize
//		fileMData.Filexstart = rdsRequest.Filexstart
//		fileMData.Filexdelta = rdsRequest.Filexdelta
//		fileMData.Fileystart = rdsRequest.Fileystart
//		fileMData.Fileydelta = rdsRequest.Fileydelta
//		fileMData.Xstart = rdsRequest.Xstart
//		fileMData.Ystart = rdsRequest.Ystart
//		fileMData.Xsize = rdsRequest.Xsize
//		fileMData.Ysize = rdsRequest.Ysize
//		fileMData.Zmin = rdsRequest.Zmin
//		fileMData.Zmax = rdsRequest.Zmax
//
//		//var marshalError error
//		fileMDataJSON, marshalError := json.Marshal(fileMData)
//		if marshalError != nil {
//			log.Println("Error Encoding metadata file to cache", marshalError)
//			w.WriteHeader(400)
//			return
//		}
//		cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)
//
//	}
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
//	// Create a Return header with some metadata in it.
//	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
//	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)
//	outzsizeStr := strconv.Itoa(fileMDataCache.Outzsize)
//
//	w.Header().Add("Access-Control-Allow-Origin", "*")
//	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
//	w.Header().Add("outxsize", outxsizeStr)
//	w.Header().Add("outysize", outysizeStr)
//	w.Header().Add("outzsize", outzsizeStr)
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
//
//}

// func GetRDSXYCut(c echo.Context) error {
// 	var data []byte
// 	var inCache bool
// 	var ok bool
// 	var RdsRequest RdsRequest

// 	// Get URL Parameters
// 	// url - /sds/rdsxcut/x1/y1/x2/y2/outxsize/outzsize
// 	cutType := c.Param("cuttype") // rdsxcut or rdsycut

// 	RdsRequest = RdsRequest{
// 		X1:       c.Param("x1"),
// 		Y1:       c.Param("y1"),
// 		X2:       c.Param("x2"),
// 		Y2:       c.Param("y2"),
// 		Outxsize: c.Param("outxsize"),
// 		Outzsize: c.Param("outzsize"),
// 	}

// 	RdsRequest.GetQueryParams(r)

// 	RdsRequest.ComputeRequestSizes()

// 	if RdsRequest.Xsize < 1 || RdsRequest.Ysize < 1 {
// 		err := fmt.Errorf("Bad Xsize or ysize. xsize: %d ysize: %d", RdsRequest.Xsize, RdsRequest.Ysize)
// 		return c.String(http.StatusBadRequest, err.Error())
// 	}

// 	if cutType == "rdsxcut" {
// 		if RdsRequest.Ysize > 1 {
// 			err := fmt.Errorf("Currently only support cut of one y line. ysize: %d", RdsRequest.Ysize)
// 			return c.String(http.StatusBadRequest, err.Error())
// 		}
// 	} else if cutType == "rdsycut" {
// 		if RdsRequest.Xsize > 1 {
// 			err := fmt.Errorf("Currently only support cut of one x line. xsize: %d", RdsRequest.Xsize)
// 			return c.String(http.StatusBadRequest, err.Error())
// 		}
// 	}

// 	c.Logger().Info("RDS XY Cut Request params xstart, ystart, xsize, ysize, outxsize, outzsize:", cutType, RdsRequest.Xstart, RdsRequest.Ystart, RdsRequest.Xsize, RdsRequest.Ysize, RdsRequest.Outxsize, RdsRequest.Outzsize)

// 	start := time.Now()
// 	cacheFileName := UrlToCacheFileName(r.URL.Path, r.URL.RawQuery)
// 	// Check if request has been previously processed and is in cache. If not process Request.
// 	if *useCache {
// 		data, inCache = GetDataFromCache(cacheFileName, "outputFiles/")
// 	} else {
// 		inCache = false
// 	}

// 	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
// 		log.Println("RDS Request not in Cache, computing result")
// 		RdsRequest.Reader, RdsRequest.FileName, ok = openDataSource(r.URL.Path, 9)
// 		if !ok {
// 			w.WriteHeader(400)
// 			return
// 		}

// 		if strings.Contains(RdsRequest.FileName, ".tmp") || strings.Contains(RdsRequest.FileName, ".prm") {
// 			RdsRequest.ProcessBlueFileHeader()
// 			if RdsRequest.SubsizeSet {
// 				RdsRequest.FileXSize = RdsRequest.Subsize

// 			} else {
// 				if RdsRequest.FileType == 1000 {
// 					log.Println("For type 1000 files, a subsize needs to be set")
// 					w.WriteHeader(400)
// 					return
// 				}
// 			}
// 			RdsRequest.ComputeYSize()
// 		} else {
// 			log.Println("Invalid File Type")
// 			w.WriteHeader(400)
// 			return
// 		}

// 		// Check Request against File Size
// 		if RdsRequest.Xsize > RdsRequest.FileXSize {
// 			log.Println("Invalid Request. Requested X size greater than file X size")
// 			w.WriteHeader(400)
// 			return
// 		}
// 		if RdsRequest.X1 > RdsRequest.FileXSize {
// 			log.Println("Invalid Request. Requested X1 greater than file X size")
// 			w.WriteHeader(400)
// 			return
// 		}
// 		if RdsRequest.X2 > RdsRequest.FileXSize {
// 			log.Println("Invalid Request. Requested X2 greater than file X size")
// 			w.WriteHeader(400)
// 			return
// 		}
// 		if RdsRequest.Y1 > RdsRequest.FileYSize {
// 			log.Println("Invalid Request. Requested Y1 greater than file Y size")
// 			w.WriteHeader(400)
// 			return
// 		}
// 		if RdsRequest.Y2 > RdsRequest.FileYSize {
// 			log.Println("Invalid Request. Requested Y2 greater than file Y size")
// 			w.WriteHeader(400)
// 			return
// 		}

// 		//If Zmin and Zmax were not explitily given then compute
// 		if !RdsRequest.Zset {
// 			RdsRequest.FindZminMax()
// 		}

// 		data = processLineRequest(RdsRequest, cutType)

// 		if *useCache {
// 			go PutItemInCache(cacheFileName, "outputFiles/", data)
// 		}

// 		// Store MetaData of request off in cache
// 		var fileMData FileMetaData
// 		fileMData.Outxsize = RdsRequest.Outxsize
// 		fileMData.Outysize = RdsRequest.Outysize
// 		fileMData.Outzsize = RdsRequest.Outzsize
// 		fileMData.Filexstart = RdsRequest.Filexstart
// 		fileMData.Filexdelta = RdsRequest.Filexdelta
// 		fileMData.Fileystart = RdsRequest.Fileystart
// 		fileMData.Fileydelta = RdsRequest.Fileydelta
// 		fileMData.Xstart = RdsRequest.Xstart
// 		fileMData.Ystart = RdsRequest.Ystart
// 		fileMData.Xsize = RdsRequest.Xsize
// 		fileMData.Ysize = RdsRequest.Ysize
// 		fileMData.Zmin = RdsRequest.Zmin
// 		fileMData.Zmax = RdsRequest.Zmax

// 		//var marshalError error
// 		fileMDataJSON, marshalError := json.Marshal(fileMData)
// 		if marshalError != nil {
// 			log.Println("Error Encoding metadata file to cache", marshalError)
// 			w.WriteHeader(400)
// 			return
// 		}
// 		PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)

// 	}
// 	elapsed := time.Since(start)
// 	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)

// 	// Get the metadata for this request to put into the return header.
// 	fileMetaDataJSON, metaInCache := GetDataFromCache(cacheFileName+"meta", "outputFiles/")
// 	if !metaInCache {
// 		log.Println("Error reading the metadata file from cache")
// 		w.WriteHeader(400)
// 		return
// 	}
// 	var fileMDataCache FileMetaData
// 	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
// 	if marshalError != nil {
// 		log.Println("Error Decoding metadata file from cache", marshalError)
// 		w.WriteHeader(400)
// 		return
// 	}
// 	// Create a Return header with some metadata in it.
// 	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
// 	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)
// 	outzsizeStr := strconv.Itoa(fileMDataCache.Outzsize)

// 	w.Header().Add("Access-Control-Allow-Origin", "*")
// 	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
// 	w.Header().Add("outxsize", outxsizeStr)
// 	w.Header().Add("outysize", outysizeStr)
// 	w.Header().Add("outzsize", outzsizeStr)
// 	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
// 	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
// 	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
// 	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
// 	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
// 	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
// 	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
// 	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
// 	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
// 	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
// 	w.WriteHeader(http.StatusOK)

// 	w.Write(data)
// }

func GetBluefileHeader(c echo.Context) error {
	filePath := c.Param("*")
	locationName := c.Param("location")
	fmt.Println(filePath, locationName)
	reader, err := openDataSource(locationName, filePath)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var bluefileheader bluefile.BlueHeader
	if strings.Contains(filePath, ".tmp") || strings.Contains(filePath, ".prm") {
		c.Logger().Infof("Opening %s for file header mode.", filePath)

		err := binary.Read(reader, binary.LittleEndian, &bluefileheader)
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		blueShort := bluefile.BlueHeaderShortenedFields{
			Version:   string(bluefileheader.Version[:]),
			HeadRep:   string(bluefileheader.HeadRep[:]),
			DataRep:   string(bluefileheader.DataRep[:]),
			Detached:  bluefileheader.Detached,
			Protected: bluefileheader.Protected,
			Pipe:      bluefileheader.Pipe,
			ExtStart:  bluefileheader.ExtStart,
			DataStart: bluefileheader.DataStart,
			DataSize:  bluefileheader.DataSize,
			FileType:  bluefileheader.FileType,
			Format:    string(bluefileheader.Format[:]),
			Flagmask:  bluefileheader.Flagmask,
			Timecode:  bluefileheader.Timecode,
			Xstart:    bluefileheader.Xstart,
			Xdelta:    bluefileheader.Xdelta,
			Xunits:    bluefileheader.Xunits,
			Subsize:   bluefileheader.Subsize,
			Ystart:    bluefileheader.Ystart,
			Ydelta:    bluefileheader.Ydelta,
			Yunits:    bluefileheader.Yunits,
		}

		//Calculated Fields
		SPA := map[string]int{
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

		BPS := map[string]float64{
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

		blueShort.Spa = SPA[string(blueShort.Format[0])]
		blueShort.Bps = BPS[string(blueShort.Format[1])]
		blueShort.Bpa = float64(blueShort.Spa) * blueShort.Bps
		if blueShort.FileType == 1000 {
			blueShort.Ape = 1
		} else {
			blueShort.Ape = int(blueShort.Subsize)
		}

		blueShort.Bpe = float64(blueShort.Ape) * blueShort.Bpa
		log.Println("Computing Size", blueShort.DataSize, blueShort.Bpa, blueShort.Ape)
		blueShort.Size = int(blueShort.DataSize / (blueShort.Bpa * float64(blueShort.Ape)))

		return c.JSON(http.StatusOK, blueShort)
	} else {
		err := fmt.Errorf("can only Return Headers for Blue Files; looking for .tmp or .prm")
		return c.String(http.StatusBadRequest, err.Error())
	}
}

func GetFileContents(c echo.Context, locationName string, filePath string) error {
	reader, err := openDataSource(locationName, filePath)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var contentType string
	if strings.Contains(filePath, ".tmp") || strings.Contains(filePath, ".prm") {
		contentType = "application/bluefile"
	} else {
		contentType = "application/binary"
	}

	return c.Stream(http.StatusOK, contentType, reader)
}

func GetDirectoryContents(c echo.Context, directoryPath string) error {
	files, err := ioutil.ReadDir(directoryPath)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	type fileObj struct {
		Filename string `json:"filename"`
		Type     string `json:"type"`
	}
	filelist := make([]fileObj, len(files))

	for i, file := range files {
		filelist[i].Filename = file.Name()
		if file.IsDir() {
			filelist[i].Type = "directory"
		} else {
			filelist[i].Type = "file"
		}
	}

	return c.JSON(http.StatusOK, filelist)
}

func GetFileLocations(c echo.Context) error {
	return c.JSON(200, sds.Config.LocationDetails)
}

func GetFileOrDirectory(c echo.Context) error {
	filePath := c.Param("*")
	locationName := c.Param("location")

	var currentLocation sds.Location
	for i := range sds.Config.LocationDetails {
		if sds.Config.LocationDetails[i].LocationName == locationName {
			currentLocation = sds.Config.LocationDetails[i]
		}
	}

	if currentLocation.LocationType != "localFile" {
		err := fmt.Errorf("listing files only support for localfile location types")
		return c.String(http.StatusBadRequest, err.Error())
	}

	fi, err := os.Stat(filePath)
	if err != nil {
		err := fmt.Errorf("error reading path %s: %s", filePath, err)
		return c.String(http.StatusBadRequest, err.Error())
	}

	// If the URL is to a file, use raw mode to return file contents
	mode := fi.Mode()
	if mode.IsRegular() {
		c.Logger().Info("Path is a file; returning contents in raw mode")
		return GetFileContents(c, locationName, filePath)
	} else {
		// Otherwise, it is likely a directory
		c.Logger().Info("Path is a directory; returning directory listing")
		return GetDirectoryContents(c, filePath)
	}
}

func SetupConfigLogCache() {
	flag.Parse()

	// Load Configuration File
	err := gonfig.GetConf(*configFile, &sds.Config)
	if err != nil {
		log.Println("Error Reading Config File, ./sdsConfig.json :", err)
		return
	}

	if sds.Config.Logfile != "" {
		// Open and setup log file
		logFile, err := os.OpenFile(sds.Config.Logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("Error Reading logfile: ", sds.Config.Logfile, err)
			return
		}
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	//Create Directories for Cache if they don't exist
	err = os.MkdirAll(sds.Config.CacheLocation, 0755)
	if err != nil {
		log.Println("Error Creating Cache File Directory: ", sds.Config.CacheLocation, err)
		return
	}
	outputFilesDir := filepath.Join(sds.Config.CacheLocation, "outputFiles/")
	err = os.MkdirAll(outputFilesDir, 0755)
	if err != nil {
		log.Println("Error Creating Cache File/outputFiles Directory ", sds.Config.CacheLocation, err)
		return
	}

	miniocache := filepath.Join(sds.Config.CacheLocation, "miniocache/")
	err = os.MkdirAll(miniocache, 0755)
	if err != nil {
		log.Println("Error Creating Cache File/miniocache Directory ", sds.Config.CacheLocation, err)
		return
	}

	// Launch a seperate routine to monitor the cache size
	outputPath := fmt.Sprintf("%s%s", sds.Config.CacheLocation, "outputFiles/")
	minioPath := fmt.Sprintf("%s%s", sds.Config.CacheLocation, "miniocache/")
	go cache.CheckCache(outputPath, sds.Config.CheckCacheEvery, sds.Config.CacheMaxBytes)
	go cache.CheckCache(minioPath, sds.Config.CheckCacheEvery, sds.Config.CacheMaxBytes)

	sds.ZminzmaxFileMap = make(map[string]sds.Zminzmax)
}

func SetupServer() *echo.Echo {
	e := echo.New()

	// Setup Middleware
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Setup API Routes
	e.GET("/sds/fs", GetFileLocations)
	e.GET("/sds/fs/:location/*", GetFileOrDirectory)
	e.GET("/sds/hdr/:location/*", GetBluefileHeader)
	e.GET("/sds/rdstile/:location/:tileXsize/:tileYsize/:decimationXMode/:decimationYMode/:tileX/:tileY/:location/*", GetRDSTile)
	// e.GET("/sds/rdsxcut/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", GetRDSXYCut)
	// e.GET("/sds/rdsycut/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", GetRDSXYCut)
	// e.GET("/sds/lds/:x1/:x2/:outxsize/:outzsize/:location/*", GetLDS)

	// Setup SigPlot Data Service UI route
	webappFS := http.FileServer(ui.GetFileSystem())
	e.GET("/ui/", echo.WrapHandler(http.StripPrefix("/ui/", webappFS)))

	// Add Prometheus as middleware for metrics gathering
	p := prometheus.NewPrometheus("sigplot_data_service", nil)
	p.Use(e)

	return e
}

var cpuprofile = flag.String("cpuprofile", "", "write CPU profile to file")
var configFile = flag.String("config", "./sdsConfig.json", "Location of Config File")
var useCache = flag.Bool("usecache", true, "Use SDS Cache. Can be disabled for certain cases like testing.")

func main() {
	SetupConfigLogCache()

	// Setup HTTP server
	e := SetupServer()

	portString := fmt.Sprintf(":%d", sds.Config.Port)
	e.Logger.Fatal(e.Start(portString))
}
