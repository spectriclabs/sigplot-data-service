package sds

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spectriclabs/sigplot-data-service/internal/bluefile"
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/spectriclabs/sigplot-data-service/internal/image"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func GetURLQueryParamFloat(r *http.Request, keyname string) (float64, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return 0.0, false
	}
	retval, err := strconv.ParseFloat(keys[0], 64)
	if err != nil {
		log.Println("Url Param ", keyname, "  is invalid")
		return 0.0, false
	}
	return retval, true
}

func GetURLQueryParamInt(r *http.Request, keyname string) (int, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return 0, false
	}
	retval, err := strconv.Atoi(keys[0])
	if err != nil {
		log.Println("Url Param ", keyname, "  is invalid")
		return 0, false
	}
	return retval, true
}

func GetURLQueryParamString(r *http.Request, keyname string) (string, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return "", false
	}
	return keys[0], true
}

func GetURLArgumentInt(url string, positionNum int) (int, bool) {
	pathData := strings.Split(url, "/")
	param := pathData[positionNum]
	retval, err := strconv.Atoi(param)
	if err != nil {
		return 0, false
	}
	return retval, true
}

func IntInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func GetBytesFromReader(reader io.ReadSeeker, firstByte int, numbytes int) ([]byte, bool) {
	outData := make([]byte, numbytes)
	// Multiple Concurrent goroutines will use this function with the same reader.
	IoMutex.Lock()
	reader.Seek(int64(firstByte), io.SeekStart)
	numRead, err := reader.Read(outData)
	IoMutex.Unlock()

	if numRead != numbytes || err != nil {
		log.Println("Failed to Read Requested Bytes", err, numRead, numbytes)
		return outData, false
	}
	return outData, true
}

func ProcessLine(outData []float64, outLineNum int, done chan bool, dataRequest RdsRequest) {
	bytesPerAtom, complexFlag := bluefile.GetFileTypeInfo(dataRequest.FileFormat)

	bytesPerElement := bytesPerAtom
	if complexFlag {
		bytesPerElement = bytesPerElement * 2
	}

	firstDataByte := float64(dataRequest.Ystart*dataRequest.FileXSize+dataRequest.Xstart) * bytesPerElement
	firstByteInt := int(math.Floor(firstDataByte))

	bytesLength := float64(dataRequest.Xsize)*bytesPerElement + (firstDataByte - float64(firstByteInt))
	bytesLengthInt := int(math.Ceil(bytesLength))
	filedata, _ := GetBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt)
	dataToProcess := bluefile.ConvertFileData(filedata, dataRequest.FileFormat)

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

	image.DownSampleLineInX(realData, dataRequest.Outxsize, dataRequest.Transform, outData, outLineNum)
	done <- true
}

func OpenDataSource(cfg *config.Config, sdsCache *cache.Cache, locationName string, filePath string) (io.ReadSeeker, error) {
	var currentLocation config.Location
	for i := range cfg.LocationDetails {
		if cfg.LocationDetails[i].LocationName == locationName {
			currentLocation = cfg.LocationDetails[i]
		}
	}
	switch currentLocation.LocationType {
	case "localFile":
		currentPath := currentLocation.Path
		fullFilepath := path.Join(currentPath, filePath)
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
		fullFilepath := path.Join(currentLocation.Path, filePath)
		cacheFileName := cache.UrlToCacheFileName(fmt.Sprintf("sds_%s%s", currentLocation.MinioBucket, fullFilepath))
		file, err := sdsCache.GetItemFromCache(cacheFileName, "miniocache/")
		if err != nil {
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

			if cfg.UseCache {
				sdsCache.PutItemInCache(cacheFileName, "miniocache/", fileData)
				cacheFileFullpath := path.Join(cfg.CacheLocation, "miniocache", cacheFileName)
				file, err = os.Open(cacheFileFullpath)
				if err != nil {
					log.Println("Error opening Minio Cache File,", err)
					return nil, err
				}
			}
		}

		elapsed := time.Since(start)
		log.Println("Time to Get Minio File ", elapsed)

		return file, nil

	default:
		err := fmt.Errorf("unsupported location type %s in %s", currentLocation.LocationType, currentLocation.LocationName)
		return nil, err
	}

}

func ProcessRequest(dataRequest RdsRequest) []byte {
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
			var lineRequest RdsRequest
			lineRequest = dataRequest
			lineRequest.Ystart = inputLine
			go ProcessLine(xThinData, inputLine-startLine, done, lineRequest)

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

func ProcessLineRequest(dataRequest RdsRequest, cutType string) []byte {
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
		filedata, _ = GetBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt)
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
			data, _ := GetBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+dataByteInt, int(bytesPerElement))
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
