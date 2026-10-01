package sds

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/spectriclabs/sigplot-data-service/internal/bluefile"
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/spectriclabs/sigplot-data-service/internal/image"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"sync"
	"time"
)

func IntInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func getBytesFromReaderMu(reader io.ReadSeeker, firstByte int, numbytes int, mu *sync.Mutex) ([]byte, bool) {
	outData := make([]byte, numbytes)
	mu.Lock()
	reader.Seek(int64(firstByte), io.SeekStart)
	numRead, err := reader.Read(outData)
	mu.Unlock()

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
	mu := dataRequest.ReaderMutex
	filedata, _ := getBytesFromReaderMu(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt, mu)
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
		fullFilepath, err := ResolvePath(currentLocation.Path, filePath)
		if err != nil {
			return nil, err
		}
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
		fullFilepath, err := ResolveObjectKey(currentLocation.Path, filePath)
		if err != nil {
			return nil, err
		}
		cacheFileName := cache.UrlToCacheFileName(fmt.Sprintf("sds_%s%s", currentLocation.MinioBucket, fullFilepath))
		var file io.ReadSeeker
		if cfg.UseCache {
			file, err = sdsCache.GetItemFromCache(cacheFileName, "miniocache/")
			if err == nil {
				if cachedFile, ok := file.(*os.File); ok {
					fi, statErr := cachedFile.Stat()
					if statErr != nil {
						cachedFile.Close()
						return nil, statErr
					}
					if fi.Size() <= 0 {
						cachedFile.Close()
						return nil, fmt.Errorf("cached minio object %s is empty", fullFilepath)
					}
				}
			}
		} else {
			err = os.ErrNotExist
		}
		if err != nil {
			log.Println("Minio File not in local file Cache, Need to fetch")
			minioClient, err := MinioClientForLocation(currentLocation)
			elapsed := time.Since(start)
			log.Println(" Time to Make connection ", elapsed)
			if err != nil {
				log.Println("Error Establishing Connection to Minio", err)
				return nil, err
			}

			start = time.Now()
			ctx := context.Background()
			object, err := minioClient.GetObject(ctx, currentLocation.MinioBucket, fullFilepath, minio.GetObjectOptions{})
			if err != nil {
				log.Println("Error getting Minio object", err)
				return nil, err
			}
			defer object.Close()

			fi, err := object.Stat()
			if err != nil {
				log.Println("Error statting Minio object", err)
				return nil, err
			}
			if fi.Size <= 0 {
				err := fmt.Errorf("minio object %s is empty", fullFilepath)
				log.Println(err)
				return nil, err
			}
			fileData, readerr := io.ReadAll(object)
			if readerr != nil || int64(len(fileData)) != fi.Size {
				log.Println("Error Reading File from from Minio", readerr)
				log.Println("Expected Bytes: ", fi.Size, "Got Bytes", len(fileData))
				if readerr != nil {
					return nil, readerr
				}
				return nil, fmt.Errorf("short read from minio object %s", fullFilepath)
			}

			if cfg.UseCache {
				if err := sdsCache.PutItemInCache(cacheFileName, "miniocache/", fileData); err != nil {
					log.Println("Error writing Minio Cache File,", err)
					return nil, err
				}
			}
			return bytes.NewReader(fileData), nil
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
	if err := ValidateOutputSize("outxsize", dataRequest.Outxsize); err != nil {
		log.Println("invalid ProcessRequest:", err)
		return nil
	}
	if err := ValidateOutputSize("outysize", dataRequest.Outysize); err != nil {
		log.Println("invalid ProcessRequest:", err)
		return nil
	}
	processedData := make([]float64, dataRequest.Outxsize*dataRequest.Outysize)

	yLinesPerOutput := float64(dataRequest.Ysize) / float64(dataRequest.Outysize)
	yLinesPerOutputCeil := int(math.Ceil(yLinesPerOutput))
	log.Println("ProcessRequest:", dataRequest.FileXSize, dataRequest.Xstart, dataRequest.Ystart, dataRequest.Xsize, dataRequest.Ysize, dataRequest.Outxsize, dataRequest.Outysize)

	sem := make(chan struct{}, runtime.NumCPU())

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

		xThinData := make([]float64, numLines*dataRequest.Outxsize)
		//log.Println("Going to Process Input Lines", startLine, endLine)

		done := make(chan bool, numLines)
		// Launch the processing of each line concurrently with bounded concurrency
		for inputLine := startLine; inputLine < endLine; inputLine++ {
			var lineRequest RdsRequest
			lineRequest = dataRequest
			lineRequest.Ystart = inputLine
			sem <- struct{}{}
			go func(lineReq RdsRequest, lineIdx int) {
				defer func() { <-sem }()
				ProcessLine(xThinData, lineIdx, done, lineReq)
			}(lineRequest, inputLine-startLine)

		}
		//Wait until all the lines have finished before moving on
		for i := 0; i < numLines; i++ {
			<-done
		}

		// Thin in y direction the subsset of lines that have now been processed in x
		yThinData := image.DownSampleLineInY(xThinData, dataRequest.Outxsize, dataRequest.Transform)

		copy(processedData[outputLine*dataRequest.Outxsize:], yThinData)

	}

	outData := image.CreateOutput(processedData, dataRequest.OutputFmt, dataRequest.Zmin, dataRequest.Zmax, dataRequest.ColorMap)
	return outData
}

func ProcessLineRequest(dataRequest RdsRequest, cutType string) ([]byte, error) {
	bytesPerAtom, complexFlag := bluefile.GetFileTypeInfo(dataRequest.FileFormat)

	bytesPerElement := bytesPerAtom
	if complexFlag {
		bytesPerElement = bytesPerElement * 2
	}

	mu := dataRequest.ReaderMutex

	// Get the slice data out of the file. For x the data is continuous, for y cuts, we need to grab one element from each row.
	var filedata []byte
	var dataToProcess []float64
	if cutType == "rdsxcut" || cutType == "lds" {
		firstDataByte := float64(dataRequest.Ystart*dataRequest.FileXSize+dataRequest.Xstart) * bytesPerElement
		firstByteInt := int(math.Floor(firstDataByte))
		bytesLength := float64(dataRequest.Xsize)*bytesPerElement + (firstDataByte - float64(firstByteInt))
		bytesLengthInt := int(math.Ceil(bytesLength))
		var ok bool
		filedata, ok = getBytesFromReaderMu(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt, mu)
		if !ok {
			return nil, fmt.Errorf("failed to read %d bytes at offset %d", bytesLengthInt, dataRequest.FileDataOffset+firstByteInt)
		}
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
			return nil, fmt.Errorf("y cut for SP data is not supported")
		}
		elemSize := int(bytesPerElement)
		filedata = make([]byte, dataRequest.Ysize*elemSize)
		for i, row := 0, dataRequest.Ystart; row < (dataRequest.Ystart + dataRequest.Ysize); i, row = i+1, row+1 {
			dataByte := float64(row*dataRequest.FileXSize+dataRequest.Xstart) * bytesPerElement
			dataByteInt := int(math.Floor(dataByte))
			data, ok := getBytesFromReaderMu(dataRequest.Reader, dataRequest.FileDataOffset+dataByteInt, elemSize, mu)
			if !ok {
				return nil, fmt.Errorf("failed to read y cut row %d at offset %d", row, dataRequest.FileDataOffset+dataByteInt)
			}
			copy(filedata[i*elemSize:], data)
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
	return outData.Bytes(), nil
}
