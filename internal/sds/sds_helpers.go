package sds

import (
	"github.com/spectriclabs/sigplot-data-service/internal/bluefile"
	"github.com/spectriclabs/sigplot-data-service/internal/image"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
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
