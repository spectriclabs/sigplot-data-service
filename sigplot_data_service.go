package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"io/ioutil"

	"github.com/minio/minio-go/v6"
	"github.com/tkanos/gonfig"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	assetfs "github.com/elazarl/go-bindata-assetfs"

	//	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)
const TileXSize = 100
const TileYSize = 100
const MAXFILESIZEZMINMAX = 32000

var ioMutex = &sync.Mutex{}
var zminmaxMutex  = &sync.Mutex{}
var zminmaxtileMutex  = &sync.Mutex{}
var uiEnabled = true // set to false by stub_asset if the ui build isn't included
var stubHTML = "" // set to HTML by stub_asset if the ui build isn't included

type Zminzmax struct {
	Zmin float64
	Zmax float64
}

var zminzmaxFileMap map[string]Zminzmax

var decimationLookup = map[int]int{
	1: 1,
	2: 2,
	3: 4,
	4: 8,
	5: 16,
	6: 32,
	7: 64,
	8: 128,
	9: 256,
	10: 512,
}

var bytesPerAtomMap = map[string]float64{
	"P": .125,
	"B": 1,
	"I": 2,
	"L": 4,
	"F": 4,
	"D": 8,
}



type Location struct {
	LocationName   string `json:"locationName"`
	LocationType   string `json:"locationType"`
	Path           string `json:"path"`
	MinioBucket    string `json:"minioBucket"`
	Location       string `json:"location"`
	MinioAccessKey string `json:"minioAccessKey"`
	MinioSecretKey string `json:"minioSecretKey"`
}

// Configuration Struct for Configuraion File
type Configuration struct {
	Port            int        `json:"port"`
	CacheLocation   string     `json:"cacheLocation"`
	Logfile         string     `json:"logfile"`
	CacheMaxBytes   int64      `json:"cacheMaxBytes"`
	CheckCacheEvery int        `json:"checkCacheEvery"`
	LocationDetails []Location `json:"locationDetails"`
}

type fileMetaData struct {
	Outxsize   int     `json:"outxsize"`
	Outysize   int     `json:"outysize"`
	Zmin       float64 `json:"zmin"`
	Zmax       float64 `json:"zmax"`
	Filexstart float64 `json:"filexstart"`
	Filexdelta float64 `json:"filexdelta"`
	Fileystart float64 `json:"fileystart"`
	Fileydelta float64 `json:"fileydelta"`
	Xstart int `json:"xstart"`
	Xsize int `json:"xsize"`
	Ystart int `json:"ystart"`
	Ysize int `json:"ysize"`
}

var configuration Configuration

func createOutput(dataIn []float64, fileFormatString string, zmin, zmax float64, colorMap string) []byte {
	for i := 0; i < len(dataIn); i++ {
		if math.IsNaN(dataIn[i]) {
			log.Println("createOutput NaN", i)
		}
	}

	dataOut := new(bytes.Buffer)
	var numColors int = 1000
	//var dataOut []byte
	if fileFormatString == "RGBA" {
		controlColors := getColorConrolPoints(colorMap)
		colorPalette := makeColorPalette(controlColors, numColors)
		if zmax != zmin {
			colorsPerSpan := (zmax - zmin) / float64(numColors)
			for i := 0; i < len(dataIn); i++ {
				// Check for dataIn[i] is NaN
				colorIndex := math.Round((dataIn[i] - zmin) / colorsPerSpan)
				colorIndex = math.Min(math.Max(colorIndex, 0), float64(numColors-1)) //Ensure colorIndex is within the colorPalette
				a := 255
				//log.Println("colorIndex", colorIndex,dataIn[i],zmin,zmax,colorsPerSpan)
				dataOut.WriteByte(byte(colorPalette[int(colorIndex)].red))
				dataOut.WriteByte(byte(colorPalette[int(colorIndex)].green))
				dataOut.WriteByte(byte(colorPalette[int(colorIndex)].blue))
				dataOut.WriteByte(byte(a))
			}
		} else {
			for i := 0; i < len(dataIn); i++ {
				a := 255
				dataOut.WriteByte(byte(colorPalette[0].red))
				dataOut.WriteByte(byte(colorPalette[0].green))
				dataOut.WriteByte(byte(colorPalette[0].blue))
				dataOut.WriteByte(byte(a))
			}
		}
		//log.Println("out_data RGBA" , len(dataOut.Bytes()))
		return dataOut.Bytes()
	} else {
		log.Println("Processing for Type ", fileFormatString)
		switch string(fileFormatString[1]) {
		case "B":
			var numSlice = make([]int8, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int8(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		case "I":
			var numSlice = make([]int16, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int16(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		case "L":
			var numSlice = make([]int32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		case "F":
			var numSlice = make([]float32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = float32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		case "D":
			var numSlice = make([]float64, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = float64(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		default:
			log.Println("Unsupported output type")
		}
		//log.Println("out_data" , len(dataOut.Bytes()))

		//TODO for SP: Add a case for P. Need to pack in 8 numbers back into 1 byte

		return dataOut.Bytes()
	}

}

func processBlueFileHeader(reader io.ReadSeeker) (string, int, int, float64, float64, float64, float64, float64, float64) {

	var bluefileheader BlueHeader
	binary.Read(reader, binary.LittleEndian, &bluefileheader)

	fileFormat := string(bluefileheader.Format[:])
	file_type := int(bluefileheader.File_type)
	subsize := int(bluefileheader.Subsize)
	xstart := bluefileheader.Xstart
	xdelta := bluefileheader.Xdelta
	ystart := bluefileheader.Ystart
	ydelta := bluefileheader.Ydelta
	data_start := bluefileheader.Data_start
	data_size := bluefileheader.Data_size

	log.Println("header data", fileFormat, file_type, subsize)

	return fileFormat, file_type, subsize, xstart, xdelta, ystart, ydelta, data_start, data_size
}

func convertFileData(bytesin []byte, file_formatstring string) []float64 {
	var bytes_per_atom int = int(bytesPerAtomMap[string(file_formatstring[1])])
	//var atoms_in_file int= 1
	//var num_slice=make([]int8,atoms_in_file)
	var out_data []float64
	switch string(file_formatstring[1]) {

	case "B":
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*int8)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "I":
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*int16)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "L":
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*int32)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "F":
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*float32)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "D":
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*float64)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = num
		}
	case "P":
		//Case for Packed Data. Rad in as uint8, then create 8 floats from that.
		bytesInFile := len(bytesin)
		out_data = make([]float64, bytesInFile*8)
		for i := 0; i < bytesInFile; i++ {
			num := *(*uint8)(unsafe.Pointer(&bytesin[i]))
			for j := 0; j < 8; j++ {
				out_data[i*8+j] = float64((num & 0x80) >> 7)
				num = num << 1 // left shift to look at next bit
			}
		}

	}
	//log.Println("out_data" , len(out_data))
	return out_data

}

func doTransform(dataIn []float64, transform string) float64 {
	switch transform {
	case "mean":
		num := stat.Mean(dataIn[:], nil)
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "max":
		num := floats.Max(dataIn[:])
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "min":
		num := floats.Min(dataIn[:])
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "maxabs":
		absnums := make([]float64,len(dataIn))
		for i:=0; i<len(dataIn); i++ {
			absnums[i] = math.Abs(dataIn[i])
		}
		num := floats.Max(absnums[:])
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	case "first":
		num := dataIn[0]
		if math.IsNaN(num) {
			log.Println("DoTransform produced NaN")
			num = 0
		}
		return num
	}
	return 0
}

func getFileTypeInfo(fileFormat string) (float64, bool) {
	//log.Println("file_format", file_format)
	var complexFlag bool = false
	var bytesPerAtom float64 = 1
	if string(fileFormat[0]) == "C" {
		complexFlag = true
	}
	//log.Println("string(file_format[1])", string(file_format[1]))
	switch string(fileFormat[1]) {
	case "B":
		bytesPerAtom = 1
	case "I":
		bytesPerAtom = 2
	case "L":
		bytesPerAtom = 4
	case "F":
		bytesPerAtom = 4
	case "D":
		bytesPerAtom = 8
	case "P":
		bytesPerAtom = 0.125
	}

	return bytesPerAtom, complexFlag
}

func down_sample_line_inx(datain []float64, outxsize int, transform string, outData []float64, outLineNum int) {
	//var inputysize int =len(datain)/framesize
	var xelementsperoutput float64
	xelementsperoutput = float64(len(datain)) / float64(outxsize)
	//var thinxdata = make([]float64,outxsize)
	if xelementsperoutput > 1 {

		var xelementsperoutput_ceil int = int(math.Ceil(xelementsperoutput))
		//log.Println("x thin" ,xelementsperoutput,xelementsperoutput_ceil,len(datain),outxsize)

		for x := 0; x < outxsize; x++ {
			var startelement int
			var endelement int
			if x != (outxsize - 1) {
				startelement = int(math.Round(float64(x) * xelementsperoutput))
				endelement = startelement + xelementsperoutput_ceil

			} else {
				endelement = len(datain)
				startelement = endelement - xelementsperoutput_ceil
			}

			//log.Println("x thin" , x,xelementsperoutput,len(datain),outxsize,startelement,endelement)
			//out_data[x] =doTransform(datain[startelement:endelement],transform)
			//log.Println("thinxdata[x]", thinxdata[x])
			outData[outLineNum*outxsize+x] = doTransform(datain[startelement:endelement], transform)

		}
	} else { // Expand Data by repeating input values into output

		for x := 0; x < outxsize; x++ {
			index := int(math.Floor(float64(x) * xelementsperoutput))
			outData[outLineNum*outxsize+x] = datain[index]
		}
	}
	//copy(outData[outLineNum*outxsize:],out_data)
	//return thinxdata
}

func downSampleLineInY(datain []float64, outxsize int, transform string) []float64 {

	numLines := len(datain) / outxsize
	//log.Println("len(datain),outxsize" ,len(datain),outxsize)
	processSlice := make([]float64, numLines)
	outData := make([]float64, outxsize)
	for x := 0; x < outxsize; x++ {
		for y := 0; y < numLines; y++ {
			//log.Println("y thin" ,y,outxsize,x)
			processSlice[y] = datain[y*outxsize+x]
		}
		outData[x] = doTransform(processSlice[:], transform)
	}
	return outData
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

//file,err :=os.Open(fileName)
//reader := io.ReaderAt(file)
//check(err)
// offset,err:=file.Seek(int64(first_byte),0)
// if offset !=int64(first_byte) {
// 	panic ("Failed to Seek")
// }
// check(err)
// num_read,err:=file.Read(out_data)

func getBytesFromReader(reader io.ReadSeeker, firstByte int, numbytes int) ([]byte, bool) {

	outData := make([]byte, numbytes)

	ioMutex.Lock() //Multiple Concurrent goroutines will use this function with the same reader.
	reader.Seek(int64(firstByte), io.SeekStart)
	numRead, err := reader.Read(outData)
	ioMutex.Unlock()

	if numRead != numbytes || err != nil {
		log.Println("Failed to Read Requested Bytes", err, numRead, numbytes)
		return outData, false
	}
	//log.Println("Read Data Line" , len(out_data))
	return outData, true

}

func applyCXmode(datain []float64, cxmode string, complexData bool) []float64 {


	loThresh := 1.0e-20
	if complexData {
		outData := make([]float64, len(datain)/2)
		for i := 0; i < len(datain)-1; i += 2 {
			
			switch cxmode {
			case "Ma":
				outData[i/2] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
			case "Ph":
				outData[i/2] = math.Atan2(datain[i+1], datain[i])
			case "Re":
				outData[i/2] = datain[i]
			case "Im":
				outData[i/2] = datain[i+1]
			case "IR":
				outData[i/2] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
			case "Lo":
				mag2 := datain[i]*datain[i] + datain[i+1]*datain[i+1]
				mag2 = math.Max(mag2, loThresh)
				outData[i/2] = 10 * math.Log10(mag2)
			case "L2":
				mag2 := datain[i]*datain[i] + datain[i+1]*datain[i+1]
				mag2 = math.Max(mag2, loThresh)
				outData[i/2] = 20 * math.Log10(mag2)
			}

		}
		return outData
	} else {

		switch cxmode {
		case "Ma":
			floats.Mul(datain, datain)
			return datain
		case "Ph":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				outData[i] = math.Atan2(0, datain[i])
			}
			return outData
		case "Re":
			return datain
		case "Im":
			outData := make([]float64, len(datain))
			return outData
		case "IR":
			return datain
		case "Lo":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				mag2 := datain[i] * datain[i]
				mag2 = math.Max(mag2, loThresh)
				outData[i] = 10 * math.Log10(mag2)
			}
			return outData
		case "L2":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				mag2 := datain[i] * datain[i]
				mag2 = math.Max(mag2, loThresh)
				outData[i] = 20 * math.Log10(mag2)
			}
			return outData

		}
		return datain

	}
}

func processline(outData []float64, outLineNum int, done chan bool, reader io.ReadSeeker, fileFormat string, fileDataOffset int, fileXSize int, xstart int, ystart int, xsize int, outxsize int, transform, cxmode string, cxmodeSet bool) {
	bytesPerAtom, complexFlag := getFileTypeInfo(fileFormat)

	//log.Println("xsize,bytes_per_atom", xsize,bytes_per_atom)
	bytesPerElement := bytesPerAtom
	if complexFlag {
		bytesPerElement = bytesPerElement * 2
	}

	firstDataByte := float64(ystart*fileXSize+xstart) * bytesPerElement
	firstByteInt := int(math.Floor(firstDataByte))

	bytesLength := float64(xsize)*bytesPerElement + (firstDataByte - float64(firstByteInt))
	bytesLengthInt := int(math.Ceil(bytesLength))

	filedata, _ := getBytesFromReader(reader, fileDataOffset+firstByteInt, bytesLengthInt)
	//filedata := get_bytes_from_file(fileName ,first_byte ,bytes_length)
	dataToProcess := convertFileData(filedata, fileFormat)

	//If the data is SP then we might have processed a few more bits than we actually needed on both sides, so reassign data_to_process to correctly point to the numbers of interest
	if bytesPerAtom < 0 {
		dataStartBit := int(math.Mod(firstDataByte, 1) * 8)
		dataEndBit := int(math.Mod(bytesLength, 1) * 8)
		var extraBits int = 0
		if dataEndBit > 0 {
			extraBits = 8 - dataEndBit
		}
		dataToProcess = dataToProcess[dataStartBit : len(dataToProcess)-extraBits]
	}

	var realData []float64
	if complexFlag {
		realData = applyCXmode(dataToProcess, cxmode, true)
	} else {
		if cxmodeSet {
			realData = applyCXmode(dataToProcess, cxmode, false)
		} else {
			realData = dataToProcess
		}

	}

	//log.Println("processline", (outxsize),len(real_data),xsize)
	//out_data :=make([]float64,outxsize)
	down_sample_line_inx(realData, outxsize, transform, outData, outLineNum)

	//copy(outData[outLineNum*outxsize:],out_data)
	done <- true
}

func processRequest(reader io.ReadSeeker, file_format string, fileDataOffset int, fileXSize int, xstart int, ystart int, xsize int, ysize int, outxsize int, outysize int, transform, cxmode string, outputFmt string, zmin, zmax float64, cxmodeSet bool, colorMap string) []byte {
	var processedData []float64

	var yLinesPerOutput float64 = float64(ysize) / float64(outysize)
	var yLinesPerOutputCeil int = int(math.Ceil(yLinesPerOutput))
	log.Println("processRequest:", fileXSize , xstart , ystart , xsize , ysize , outxsize , outysize )
	// Loop over the output Y Lines
	for outputLine := 0; outputLine < outysize; outputLine++ {
		//log.Println("Processing Output Line ", outputLine)
		// For Each Output Y line Read and process the required lines from the input file
		var startLine int
		var endLine int

		if outputLine != (outysize - 1) {
			//log.Println("Not Last Line. yLinesPerOutput
			startLine = ystart + int(math.Round(float64(outputLine)*yLinesPerOutput))
			endLine = startLine + yLinesPerOutputCeil
		} else { //Last OutputLine, use the last line and work backwards the lineperoutput
			endLine = ystart + ysize
			startLine = endLine - yLinesPerOutputCeil
		}

		// Number of y lines that will be processed this time through the loop
		numLines := endLine - startLine

		// Make channels to collect the data from processing all the lines in parallel.
		//var chans [100]chan []float64
		chans := make([]chan []float64, numLines)
		for i := range chans {
			chans[i] = make(chan []float64)
		}
		xThinData := make([]float64, numLines*outxsize)
		//log.Println("Going to Process Input Lines", startLine, endLine)

		done := make(chan bool, 1)
		// Launch the processing of each line concurrently and put the data into a set of channels
		for inputLine := startLine; inputLine < endLine; inputLine++ {
			go processline(xThinData, inputLine-startLine, done, reader, file_format, fileDataOffset, fileXSize, xstart, inputLine, xsize, outxsize, transform, cxmode, cxmodeSet)

		}
		//Wait until all the lines have finished before moving on
		for i := 0; i < numLines; i++ {
			<-done
		}

		for i := 0; i < len(xThinData); i++ {
			if math.IsNaN(xThinData[i]) {
				log.Println("processedDataNaN", outputLine, i)
			}
		}
		// Thin in y direction the subsset of lines that have now been processed in x
		yThinData := downSampleLineInY(xThinData, outxsize, transform)
		//log.Println("Thin Y data is currently ", len(yThinData))

		for i := 0; i < len(yThinData); i++ {
			if math.IsNaN(yThinData[i]) {
				log.Println("processedDataNaN", outputLine, i)
			}
		}

		processedData = append(processedData, yThinData...)
		//log.Println("processedData is currently ", len(processedData))

		for i := 0; i < len(processedData); i++ {
			if math.IsNaN(processedData[i]) {
				log.Println("processedDataNaN", outputLine, i)
			}
		}

	}

	outData := createOutput(processedData, outputFmt, zmin, zmax, colorMap)
	return outData
}

func getURLArgumentFloat(r *http.Request, keyname string) (float64, bool) {
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

func getURLArgumentInt(r *http.Request, keyname string) (int, bool) {
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

func getURLArgumentString(r *http.Request, keyname string) (string, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return "", false
	}
	return keys[0], true
}

func openDataSource(url string) (io.ReadSeeker, string, bool) {

	pathData := strings.Split(url, "/")
	locationName := pathData[2]
	var urlPath string = ""
	for i := 3; i < len(pathData)-1; i++ {
		urlPath = urlPath + pathData[i] + "/"
	}

	fileName := pathData[len(pathData)-1]
	var currentLocation Location
	for i := range configuration.LocationDetails {
		if configuration.LocationDetails[i].LocationName == locationName {
			currentLocation = configuration.LocationDetails[i]
		}
	}

	switch currentLocation.LocationType {
	case "localFile":
		if string(currentLocation.Path[len(currentLocation.Path)-1]) != "/" {
			currentLocation.Path += "/"
		}
		fullFilepath := fmt.Sprintf("%s%s%s", currentLocation.Path, urlPath, fileName)
		log.Println("Reading Local File. LocationName=", locationName, "fileName=", fileName, "fullPath=", fullFilepath)
		file, err := os.Open(fullFilepath)
		if err != nil {
			log.Println("Error opening File,", err)
			return nil, "", false
		}
		reader := io.ReadSeeker(file)
		return reader, fileName, true
	case "minio":
		start := time.Now()
		fullFilepath := fmt.Sprintf("%s%s%s", currentLocation.Path, urlPath, fileName)
		cacheFileName := fmt.Sprintf("%s%s%s", currentLocation.MinioBucket, fullFilepath, "x1y1x2y2outxsizeoutysize")
		file, inCache := getItemFromCache(cacheFileName, "miniocache/")
		if !inCache {
			log.Println("Minio File not in local file Cache, Need to fetch")
			minioClient, err := minio.New(currentLocation.Location, currentLocation.MinioAccessKey, currentLocation.MinioSecretKey, false)
			elapsed := time.Since(start)
			log.Println(" Time to Make connection ", elapsed)
			if err != nil {
				log.Println("Error Establishing Connection to Minio", err)
				return nil, "", false
			}

			start = time.Now()
			object, err := minioClient.GetObject(currentLocation.MinioBucket, fullFilepath, minio.GetObjectOptions{})

			fi, _ := object.Stat()
			fileData := make([]byte, fi.Size)
			//var readerr error
			numRead, readerr := object.Read(fileData)
			if int64(numRead) != fi.Size {
				log.Println("Error Reading File from from Minio", readerr)
				log.Println("Expected Bytes: ", fi.Size, "Got Bytes", numRead)
				return nil, "", false
			}

			putItemInCache(cacheFileName, "miniocache/", fileData)
			cacheFileFullpath := fmt.Sprintf("%s%s%s", configuration.CacheLocation, "miniocache/", cacheFileName)
			file, err = os.Open(cacheFileFullpath)
			if err != nil {
				log.Println("Error opening Minio Cache File,", err)
				return nil, "", false
			}
		}
		reader := io.ReadSeeker(file)
		elapsed := time.Since(start)
		log.Println(" Time to Get Minio File ", elapsed)

		return reader, fileName, true

	default:
		log.Println("Unsupported Location Type")
		return nil, "", false
	}

}

type rdsServer struct{}

func (s *rdsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var data []byte
	var inCache bool

	var file_format string
	var file_type int
	var fileXSize,fileYSize int
	var filexstart, filexdelta, fileystart, fileydelta, data_offset,data_size float64
	var fileDataOffset int

	// Get Rest of URL Parameters
	x1, ok := getURLArgumentInt(r, "x1")
	if !ok || x1 < 0 {
		log.Println("X1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	y1, ok := getURLArgumentInt(r, "y1")
	if !ok || y1 < 0 {
		log.Println("Y1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	x2, ok := getURLArgumentInt(r, "x2")
	if !ok || x2 < 0 {
		log.Println("X2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	y2, ok := getURLArgumentInt(r, "y2")
	if !ok || y2 < 0 {
		log.Println("Y2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	ystart := int(math.Min(float64(y1), float64(y2)))
	xstart := int(math.Min(float64(x1), float64(x2)))
	xsize := int(math.Abs(float64(x2) - float64(x1)))
	ysize := int(math.Abs(float64(y2) - float64(y1)))

	if xsize < 1 || ysize < 1 {
		log.Println("Bad Xsize or ysize. xsize: ", xsize, " ysize: ", ysize)
		w.WriteHeader(400)
		return
	}

	outxsize, ok := getURLArgumentInt(r, "outxsize")
	if !ok || outxsize < 1 {
		log.Println("outxsize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	outysize, ok := getURLArgumentInt(r, "outysize")
	if !ok || outysize < 1 {
		log.Println("outysize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	transform, ok := getURLArgumentString(r, "transform")
	if !ok {
		log.Println("transform Missing. Required Field")
		w.WriteHeader(400)
		return
	}

	subsizeSet := true
	subsize, ok := getURLArgumentInt(r, "subsize")
	if !ok {
		subsize = 0
		subsizeSet = false
	}

	cxmodeSet := true
	cxmode, ok := getURLArgumentString(r, "cxmode")
	if !ok {
		cxmode = "mag"
		cxmodeSet = false
	}

	//log.Println("Reported file_data_size", file_data_size)

	zmin, zminSet := getURLArgumentFloat(r, "zmin")
	if !zminSet {
		log.Println("Zmin Not Specified. Will estimate from file Selection")
		zmin = 0
	}

	zmax, zmaxSet := getURLArgumentFloat(r, "zmax")
	if !zmaxSet {
		log.Println("Zmax Not Specified. Will estimate from file Selection")
		zmax = 0
	}

	zset := (zmaxSet && zminSet)
	colorMap, ok := getURLArgumentString(r, "colormap")
	if !ok {
		log.Println("colorMap Not Specified.Defaulting to RampColormap")
		colorMap = "RampColormap"
	}

	log.Println("params xstart, ystart, xsize, ysize, outxsize, outysize:", xstart, ystart, xsize, ysize, outxsize, outysize)

	start := time.Now()
	cacheFileName := urlToCacheFileName(r.URL.Path, r.URL.RawQuery)
	// Check if request has been previously processed and is in cache. If not process Request.
	if *useCache {
		data, inCache = getDataFromCache(cacheFileName, "outputFiles/")
	} else {
		inCache = false
	}

	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
		log.Println("RDS Request not in Cache, computing result")
		reader, fileName, succeed := openDataSource(r.URL.Path)
		if !succeed {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(fileName, ".tmp") || strings.Contains(fileName, ".prm") {
			log.Println("Processing File as Blue File")
			file_format, file_type, fileXSize, filexstart, filexdelta, fileystart, fileydelta, data_offset, data_size = processBlueFileHeader(reader)
			fileDataOffset = int(data_offset)
			if subsizeSet {
				fileXSize = subsize
			} else {
				if file_type == 1000 {
					log.Println("For type 1000 files, a subsize needs to be set")
					w.WriteHeader(400)
					return
				}
			}

			fileYSize = int(float64(data_size) / bytesPerAtomMap[string(file_format[1])])/fileXSize
			if string(file_format[0]) == "C" {
				fileYSize = fileYSize/2
			}

		} else if strings.Count(fileName, "_") == 3 {
			log.Println("Processing File as binary file with metadata in filename with underscores")
			fileData := strings.Split(fileName, "_")
			// Need to get these parameters from file metadata
			file_format = fileData[1]
			fileDataOffset = 0
			var err error
			fileXSize, err = strconv.Atoi(fileData[2])
			if err != nil {
				log.Println("Bad xfile size in filename")
				fileXSize = 0
				w.WriteHeader(400)
				return
			}
			fileYSize, err = strconv.Atoi(fileData[3])
			if err != nil {
				log.Println("Bad yfile size in filename")
				fileYSize = 0
				w.WriteHeader(400)
				return
			}
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}

		if xsize > fileXSize {
			log.Println("Invalid Request. Requested X size greater than file X size")
			w.WriteHeader(400)
			return
		}

		outputFmt, ok := getURLArgumentString(r, "outfmt")
		if !ok {
			log.Println("Outformat Not Specified. Setting Equal to Input Format")
			outputFmt = file_format

		}

		if !zset {
			log.Println("Zmin/Zmax not set need to find one")
			zminmaxMutex.Lock()
			zminmax, ok := zminzmaxFileMap[fileName+cxmode]
			if ok {
				zmin = zminmax.Zmin
				zmax = zminmax.Zmax
				log.Println("Zmin/Zmax for this file previously computed", zmin, zmax)
			} else {
				if (fileXSize*fileYSize) < MAXFILESIZEZMINMAX { // File is small enough, look at entire file for Zmax/Zmin
					min :=make([]float64,fileYSize)
					max :=make([]float64,fileYSize)
					done := make(chan bool, 1)
					for line:=0;line<fileYSize;line++ {
						go processline(min, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, line, xsize, 1, "min", cxmode, cxmodeSet)
						go processline(max, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, line, xsize, 1, "max", cxmode, cxmodeSet)
					}
					for i := 0; i < fileYSize*2; i++ {
						<-done
					}
					zmin = floats.Min(min)
					zmax = floats.Max(max)
					zminzmaxFileMap[fileName+cxmode] = Zminzmax{zmin,zmax}
				} else { // If file is large then check the first, last, and a number of middles lines 
					numMiddlesLines:= int(math.Max(float64((MAXFILESIZEZMINMAX/fileXSize)-2),0))
					min :=make([]float64,2+numMiddlesLines)
					max :=make([]float64,2+numMiddlesLines)
					done := make(chan bool, 1)
					go processline(min, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, 0, xsize, 1, "min", cxmode, cxmodeSet)
					go processline(max, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, 0, xsize, 1, "max", cxmode, cxmodeSet)
					go processline(min, 1, done, reader, file_format, fileDataOffset, fileXSize, xstart, fileYSize-1, xsize, 1, "min", cxmode, cxmodeSet)
					go processline(max, 1, done, reader, file_format, fileDataOffset, fileXSize, xstart, fileYSize-1, xsize, 1, "max", cxmode, cxmodeSet)
					for i:=0; i<numMiddlesLines; i++ {
						linenum:= int(((fileYSize)/numMiddlesLines)*i)
						go processline(min, i+2, done, reader, file_format, fileDataOffset, fileXSize, xstart, linenum, xsize, 1, "min", cxmode, cxmodeSet)
					}
	
					for i := 0; i < 2+numMiddlesLines; i++ {
						<-done
					}
					zmin = floats.Min(min)
					zmax = floats.Max(max)
					zminzmaxFileMap[fileName+cxmode] = Zminzmax{zmin,zmax}

				}
				log.Println("Found Zmin, Zmax to be", zmin, zmax)
				
			}
			zminmaxMutex.Unlock()
		}

		data = processRequest(reader, file_format, fileDataOffset, fileXSize, xstart, ystart, xsize, ysize, outxsize, outysize, transform, cxmode, outputFmt, zmin, zmax, cxmodeSet, colorMap)	
		if *useCache {
			go putItemInCache(cacheFileName, "outputFiles/", data)
		}

		// Store MetaData of request off in cache
		var fileMData fileMetaData
		fileMData.Outxsize = outxsize
		fileMData.Outysize = outysize
		fileMData.Filexstart = filexstart
		fileMData.Filexdelta = filexdelta
		fileMData.Fileystart = fileystart
		fileMData.Fileydelta = fileydelta
		fileMData.Xstart = xstart
		fileMData.Ystart = ystart
		fileMData.Xsize = xsize
		fileMData.Ysize = ysize
		fileMData.Zmin = zmin
		fileMData.Zmax = zmax

		//var marshalError error
		fileMDataJSON, marshalError := json.Marshal(fileMData)
		if marshalError != nil {
			log.Println("Error Encoding metadata file to cache", marshalError)
			w.WriteHeader(400)
			return
		}
		putItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)

	}

	elapsed := time.Since(start)
	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)

	// Get the metadata for this request to put into the return header.
	fileMetaDataJSON, metaInCache := getDataFromCache(cacheFileName+"meta", "outputFiles/")
	if !metaInCache {
		log.Println("Error reading the metadata file from cache")
		w.WriteHeader(400)
		return
	}
	var fileMDataCache fileMetaData
	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
	if marshalError != nil {
		log.Println("Error Decoding metadata file from cache", marshalError)
		w.WriteHeader(400)
		return
	}

	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	//w.Header().Add("Access-Control-Expose-Headers", "*")
	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

type rdsTileServer struct{}

func (s *rdsTileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var inCache bool

	var file_format string
	var file_type int
	var fileXSize,fileYSize int
	var filexstart, filexdelta, fileystart, fileydelta, data_offset,data_size float64
	var fileDataOffset int


	// Get URL Parameters


	decXMode, ok := getURLArgumentInt(r, "decx")
	if !ok || decXMode < 0 || decXMode > 10 {
		log.Println("decx Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	decYMode, ok := getURLArgumentInt(r, "decy")
	if !ok || decYMode < 0 || decYMode > 10 {
		log.Println("decy Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	tileX, ok := getURLArgumentInt(r, "tilex")
	if !ok || tileX < 0 {
		log.Println("tileX Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	tileY, ok := getURLArgumentInt(r, "tiley")
	if !ok || tileY < 0 {
		log.Println("tiley Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	// Convert decimation Mode to decimation Number
	decX := decimationLookup[decXMode]
	decY := decimationLookup[decYMode]

	xstart := tileX*TileXSize*decX
	ystart := tileY*TileYSize*decY
	xsize := int(TileXSize*decX)
	ysize := int(TileYSize*decY)
	log.Println("tileX: " , tileX, "TileXSize" , TileXSize, "decX" , decX )
	log.Println("tileY: " , tileX, "TileYSize" , TileXSize, "decY" , decX )

	if xsize < 1 || ysize < 1 {
		log.Println("Bad Xsize or ysize. xsize: ", xsize, " ysize: ", ysize)
		w.WriteHeader(400)
		return
	}

	outxsize := TileXSize
	outysize := TileYSize

	transform, ok := getURLArgumentString(r, "transform")
	if !ok {
		log.Println("transform Missing. Required Field")
		w.WriteHeader(400)
		return
	}

	subsizeSet := true
	subsize, ok := getURLArgumentInt(r, "subsize")
	if !ok {
		subsize = 0
		subsizeSet = false
	}

	cxmodeSet := true
	cxmode, ok := getURLArgumentString(r, "cxmode")
	if !ok {
		cxmode = "mag"
		cxmodeSet = false
	}

	//log.Println("Reported file_data_size", file_data_size)

	zmin, zminSet := getURLArgumentFloat(r, "zmin")
	if !zminSet {
		log.Println("Zmin Not Specified. Will estimate from file Selection")
		zmin = 0
	}

	zmax, zmaxSet := getURLArgumentFloat(r, "zmax")
	if !zmaxSet {
		log.Println("Zmax Not Specified. Will estimate from file Selection")
		zmax = 0
	}

	zset := (zmaxSet && zminSet)
	colorMap, ok := getURLArgumentString(r, "colormap")
	if !ok {
		log.Println("colorMap Not Specified.Defaulting to RampColormap")
		colorMap = "RampColormap"
	}

	log.Println("Tile Mode: params xstart, ystart, xsize, ysize, outxsize, outysize:", xstart, ystart, xsize, ysize, outxsize, outysize)

	start := time.Now()
	cacheFileName := urlToCacheFileName(r.URL.Path, r.URL.RawQuery)
	// Check if request has been previously processed and is in cache. If not process Request.
	if *useCache {
		data, inCache = getDataFromCache(cacheFileName, "outputFiles/")
	} else {
		inCache = false
	}

	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
		log.Println("RDS Request not in Cache, computing result")
		reader, fileName, succeed := openDataSource(r.URL.Path)
		if !succeed {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(fileName, ".tmp") || strings.Contains(fileName, ".prm") {
			log.Println("Processing File as Blue File")
			file_format, file_type, fileXSize, filexstart, filexdelta, fileystart, fileydelta, data_offset, data_size = processBlueFileHeader(reader)
			fileDataOffset = int(data_offset)
			if subsizeSet {
				fileXSize = subsize
			} else {
				if file_type == 1000 {
					log.Println("For type 1000 files, a subsize needs to be set")
					w.WriteHeader(400)
					return
				}
			}
			fileYSize = int(float64(data_size) / bytesPerAtomMap[string(file_format[1])])/fileXSize
			if string(file_format[0]) == "C" {
				fileYSize = fileYSize/2
			}

		} else if strings.Count(fileName, "_") == 3 {
			log.Println("Processing File as binary file with metadata in filename with underscores")
			fileData := strings.Split(fileName, "_")
			// Need to get these parameters from file metadata
			file_format = fileData[1]
			fileDataOffset = 0
			var err error
			fileXSize, err = strconv.Atoi(fileData[2])
			if err != nil {
				log.Println("Bad xfile size in filename")
				fileXSize = 0
				w.WriteHeader(400)
				return
			}
			fileYSize, err = strconv.Atoi(fileData[3])
			if err != nil {
				log.Println("Bad yfile size in filename")
				fileYSize = 0
				w.WriteHeader(400)
				return
			}
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}
		if xstart>fileXSize || ystart>fileYSize {
			log.Println("Invalid Tile Request. " , xstart, fileXSize, ystart, fileYSize)
			w.WriteHeader(400)
			return
		}

		if (xstart+xsize)>fileXSize {
			xsize=fileXSize-xstart
			outxsize=xsize/decX
		}
		if (ystart+ysize)>fileYSize {
			ysize=fileYSize-ystart
			outysize=ysize/decY
		}
		if xsize > fileXSize {
			log.Println("Invalid Request. Requested X size greater than file X size")
			w.WriteHeader(400)
			return
		}

		outputFmt, ok := getURLArgumentString(r, "outfmt")
		if !ok {
			log.Println("Outformat Not Specified. Setting Equal to Input Format")
			outputFmt = file_format

		}
		if !zset {
			zminmaxtileMutex.Lock()
			zminmax, ok := zminzmaxFileMap[fileName+cxmode]
			if ok {
				zmin = zminmax.Zmin
				zmax = zminmax.Zmax
				log.Println("Zmin/Zmax for this file previously computed", zmin, zmax)
			} else {
				log.Println("Computing Zmax/Zmin, not previously computed")
				if (fileXSize*fileYSize) < MAXFILESIZEZMINMAX { // File is small enough, look at entire file for Zmax/Zmin
					min :=make([]float64,fileYSize)
					max :=make([]float64,fileYSize)
					done := make(chan bool, 1)
					for line:=0;line<fileYSize;line++ {
						go processline(min, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, line, xsize, 1, "min", cxmode, cxmodeSet)
						go processline(max, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, line, xsize, 1, "max", cxmode, cxmodeSet)
					}
					for i := 0; i < fileYSize*2; i++ {
						<-done
					}
					zmin = floats.Min(min)
					zmax = floats.Max(max)
					zminzmaxFileMap[fileName+cxmode] = Zminzmax{zmin,zmax}
				} else { // If file is large then check the first, last, and a number of middles lines 
					numMiddlesLines:= int(math.Max(float64((MAXFILESIZEZMINMAX/fileXSize)-2),0))
					min :=make([]float64,2+numMiddlesLines)
					max :=make([]float64,2+numMiddlesLines)
					done := make(chan bool, 1)
					go processline(min, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, 0, xsize, 1, "min", cxmode, cxmodeSet)
					go processline(max, 0, done, reader, file_format, fileDataOffset, fileXSize, xstart, 0, xsize, 1, "max", cxmode, cxmodeSet)
					go processline(min, 1, done, reader, file_format, fileDataOffset, fileXSize, xstart, fileYSize-1, xsize, 1, "min", cxmode, cxmodeSet)
					go processline(max, 1, done, reader, file_format, fileDataOffset, fileXSize, xstart, fileYSize-1, xsize, 1, "max", cxmode, cxmodeSet)
					for i:=0; i<numMiddlesLines; i++ {
						linenum:= int(((fileYSize)/numMiddlesLines)*i)
						go processline(min, i+2, done, reader, file_format, fileDataOffset, fileXSize, xstart, linenum, xsize, 1, "min", cxmode, cxmodeSet)
						go processline(max, i+2, done, reader, file_format, fileDataOffset, fileXSize, xstart, linenum, xsize, 1, "max", cxmode, cxmodeSet)
					}
					for i := 0; i < (2+numMiddlesLines)*2; i++ {
						<-done
					}
					zmin = floats.Min(min)
					zmax = floats.Max(max)
					zminzmaxFileMap[fileName+cxmode] = Zminzmax{zmin,zmax}

				}
				log.Println("Found Zmin, Zmax to be", zmin, zmax)
			}
			zminmaxtileMutex.Unlock()

		}

		data = processRequest(reader, file_format, fileDataOffset, fileXSize, xstart, ystart, xsize, ysize, outxsize, outysize, transform, cxmode, outputFmt, zmin, zmax, cxmodeSet, colorMap)
		if *useCache {
			go putItemInCache(cacheFileName, "outputFiles/", data)
		}

		var fileMData fileMetaData
		fileMData.Outxsize = outxsize
		fileMData.Outysize = outysize
		fileMData.Filexstart = filexstart
		fileMData.Filexdelta = filexdelta
		fileMData.Fileystart = fileystart
		fileMData.Fileydelta = fileydelta
		fileMData.Xstart = xstart
		fileMData.Ystart = ystart
		fileMData.Xsize = xsize
		fileMData.Ysize = ysize
		fileMData.Zmin = zmin
		fileMData.Zmax = zmax

		//var marshalError error
		fileMDataJSON, marshalError := json.Marshal(fileMData)
		if marshalError != nil {
			log.Println("Error Encoding metadata file to cache", marshalError)
			w.WriteHeader(400)
			return
		}
		putItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)

	}

	elapsed := time.Since(start)
	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)

	// Get the metadata for this request to put into the return header.
	fileMetaDataJSON, metaInCache := getDataFromCache(cacheFileName+"meta", "outputFiles/")
	if !metaInCache {
		log.Println("Error reading the metadata file from cache")
		w.WriteHeader(400)
		return
	}
	var fileMDataCache fileMetaData
	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
	if marshalError != nil {
		log.Println("Error Decoding metadata file from cache", marshalError)
		w.WriteHeader(400)
		return
	}

	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	//w.Header().Add("Access-Control-Expose-Headers", "*")
	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	w.Header().Add("xmin",fmt.Sprintf("%f",fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
	w.Header().Add("xmax",fmt.Sprintf("%f",fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
	w.Header().Add("ymin",fmt.Sprintf("%f",fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
	w.Header().Add("ymax",fmt.Sprintf("%f",fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

//var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var configFile = flag.String("config", "./sdsConfig.json", "Location of Config File")
var useCache = flag.Bool("usecache", true, "Use SDS Cache. Can be disabled for certain cases like testing." )

type fileHeaderServer struct{}

func (s *fileHeaderServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	reader, fileName, succeed := openDataSource(r.URL.Path)
	if !succeed {
		log.Println("Error Reading from Data Source")
		w.WriteHeader(400)
		return
	}

	var bluefileheader BlueHeader
	var returnbytes []byte
	if strings.Contains(fileName, ".tmp") || strings.Contains(fileName, ".prm") {

		log.Println("Opening File for file Header Mode ", fileName)
		// file,err :=os.Open(fullFilepath)
		// if err !=nil {
		// 	log.Println("Error Opening File", err)
		// 	w.WriteHeader(400)
		// 	return
		// }

		binary.Read(reader, binary.LittleEndian, &bluefileheader)

		var blueShort BlueHeaderShortenedFields
		blueShort.Version = string(bluefileheader.Version[:])
		blueShort.Head_rep = string(bluefileheader.Head_rep[:])
		blueShort.Data_rep = string(bluefileheader.Data_rep[:])
		blueShort.Detached = bluefileheader.Detached
		blueShort.Protected = bluefileheader.Protected
		blueShort.Pipe = bluefileheader.Pipe
		blueShort.Ext_start = bluefileheader.Ext_start
		blueShort.Data_start = bluefileheader.Data_start
		blueShort.Data_size = bluefileheader.Data_size
		blueShort.File_type = bluefileheader.File_type
		blueShort.Format = string(bluefileheader.Format[:])
		blueShort.Flagmask = bluefileheader.Flagmask
		blueShort.Timecode = bluefileheader.Timecode
		blueShort.Xstart = bluefileheader.Xstart
		blueShort.Xdelta = bluefileheader.Xdelta
		blueShort.Xunits = bluefileheader.Xunits
		blueShort.Subsize = bluefileheader.Subsize
		blueShort.Ystart = bluefileheader.Ystart
		blueShort.Ydelta = bluefileheader.Ydelta
		blueShort.Yunits = bluefileheader.Yunits

		//Calculated Fields
		SPA := make(map[string]int)
		SPA["S"] = 1
		SPA["C"] = 2
		SPA["V"] = 3
		SPA["Q"] = 4
		SPA["M"] = 9
		SPA["X"] = 10
		SPA["T"] = 16
		SPA["U"] = 1
		SPA["1"] = 1
		SPA["2"] = 2
		SPA["3"] = 3
		SPA["4"] = 4
		SPA["5"] = 5
		SPA["6"] = 6
		SPA["7"] = 7
		SPA["8"] = 8
		SPA["9"] = 9

		BPS := make(map[string]float64)
		BPS["P"] = 0.125
		BPS["A"] = 1
		BPS["O"] = 1
		BPS["B"] = 1
		BPS["I"] = 2
		BPS["L"] = 4
		BPS["X"] = 8
		BPS["F"] = 4
		BPS["D"] = 8

		blueShort.Spa = SPA[string(blueShort.Format[0])]
		blueShort.Bps = BPS[string(blueShort.Format[1])]
		blueShort.Bpa = float64(blueShort.Spa) * blueShort.Bps
		if blueShort.File_type == 1000 {
			blueShort.Ape = 1
		} else {
			blueShort.Ape = int(blueShort.Subsize)
		}

		blueShort.Bpe = float64(blueShort.Ape) * blueShort.Bpa
		log.Println("Computing Size", blueShort.Data_size, blueShort.Bpa, blueShort.Ape)
		blueShort.Size = int(blueShort.Data_size / (blueShort.Bpa * float64(blueShort.Ape)))

		var marshalError error
		returnbytes, marshalError = json.Marshal(blueShort)
		if marshalError != nil {
			log.Println("Problem Marshalling Header to JSON ", marshalError)
			w.WriteHeader(400)
			return
		}

	} else {
		log.Println("Can only Return Headers for Blue Files. Looking for .tmp or .prm")
		w.WriteHeader(400)
		return
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Expose-Headers", "*")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(returnbytes)
}

type rawServer struct{}

func (s *rawServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    reader, fileName, succeed := openDataSource(r.URL.Path)
    if !succeed {
	    log.Println("Error Reading from Data Source")
	    w.WriteHeader(400)
	    return
    }

    if strings.Contains(fileName, ".tmp") || strings.Contains(fileName, ".prm") {
        w.Header().Add("Content-Type", "application/bluefile")
    } else {
        w.Header().Add("Content-Type", "application/binary")
    }
    w.Header().Add("Access-Control-Allow-Origin", "*")
    w.Header().Add("Access-Control-Expose-Headers", "*")

    http.ServeContent(w, r, fileName, time.Now(), reader)
}

func handleUI(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// TODO add any header things we want
		// header := w.Header()
		// header.Add(...)
		h.ServeHTTP(w, req)
		return
	})
}

type UIAssetWrapper struct {
	FileSystem *assetfs.AssetFS
}

func (fs *UIAssetWrapper) Open(name string) (http.File, error) {
	log.Println("Opening " + name)
	if file, err := fs.FileSystem.Open(name); err == nil {
		log.Println("found " + name)
		return file, nil
	} else {
		log.Println("Not found " + name)
		// serve index.html instead of 404ing
		if err == os.ErrNotExist {
			return fs.FileSystem.Open("index.html")
		}
		return nil, err
	}
}

type locationListServer struct{}

func (s *locationListServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Processing Request as locationListServer")
	locationDetailsJSONBytes, marshalError := json.Marshal(configuration.LocationDetails)
	if marshalError != nil {
		log.Println("Error Encoding LocationDetails ", marshalError)
		w.WriteHeader(400)
		return
	}
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Expose-Headers", "*")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(locationDetailsJSONBytes)

}

type directoryListServer struct{}

func (s *directoryListServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("Processing Request to List Directory Contents")
	pathData := strings.Split(r.URL.Path, "/")
	locationName := pathData[2]
	var urlPath string = ""
	for i := 3; i < len(pathData); i++ {
		urlPath = urlPath + pathData[i] + "/"
	}

	if string(r.URL.Path[len(r.URL.Path)-1]) != "/" {
		urlPath = strings.TrimSuffix(urlPath, "/")
	}

	var currentLocation Location
	for i := range configuration.LocationDetails {
		if configuration.LocationDetails[i].LocationName == locationName {
			currentLocation = configuration.LocationDetails[i]
		}
	}

	if  currentLocation.LocationType != "localFile" {
		log.Println("Error: Listing Files only support for localfile location Types" )
		w.WriteHeader(400)
		return
	}

	if string(currentLocation.Path[len(currentLocation.Path)-1]) != "/" {
		currentLocation.Path += "/"
	}
	fullFilepath := fmt.Sprintf("%s%s", currentLocation.Path, urlPath)
	log.Println("Looking in location:" ,fullFilepath)
	fi, err := os.Stat(fullFilepath)
    if err != nil {
		log.Println("Error reading path", fullFilepath, err)
		w.WriteHeader(400)
        return
    }
	mode := fi.Mode() 
	if mode.IsRegular() { //If the URL is to a file, then use raw mode to return file contents
		log.Println("Path is a file, so will return its contents in raw mode")
		rawServer := &rawServer{}
		rawServer.ServeHTTP(w, r)
		return 
	}


	files, err := ioutil.ReadDir(fullFilepath)
	if err != nil {
		log.Println("List Directory Error: ", err)
		w.WriteHeader(400)
		return
	}
	type fileList struct {
		filenames   []string`json:"filenames"`
	}
	var filelist fileList
	filenames := make([]string,len(files))
	i:=0
	for _, file := range files {
		if !(file.IsDir()) {
			filenames[i]=file.Name()
			i++
		}
	}

	filelist.filenames = filenames
	returnbytes, marshalError := json.Marshal(filenames)
	if marshalError != nil {
		log.Println("Problem Marshalling Header to JSON ", marshalError)
		w.WriteHeader(400)
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Expose-Headers", "*")
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(returnbytes)
}

type routerServer struct{}

func (s *routerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//This function serves as the router for the url to get to the correct handler.
	//URLs without a mode, act like URL to directories, location, or files which return the appropriate contents and don't do any specific SDS functions
	//URLs with ?mode=rds or ?mode=hdr route to their respective SDS handler types. 
	rdsServer := &rdsServer{}
	rdsTileServer := &rdsTileServer{}
	headerServer := &fileHeaderServer{}
	rawServer := &rawServer{}
	directoryListServer :=&directoryListServer{}
	locationListServer :=&locationListServer{}

	mode, ok := getURLArgumentString(r, "mode")
	if !ok {
		pathData := strings.Split(r.URL.Path, "/")

		if len(pathData) ==2 || (len(pathData) ==3 && pathData[2]==""){ //If no path is specified after /sds/ then list locations
			locationListServer.ServeHTTP(w, r)
			return
		} else {

		directoryListServer.ServeHTTP(w, r) // if mode not present then try to process as either filename or directory
		return
		}
	} else {

		switch mode {
		case "rds": //Valid url is /sds/path/to/file/<filename>?mode=rds
			rdsServer.ServeHTTP(w, r)
		case "rdstile": //Valid url is /sds/path/to/file/<filename>?mode=rdstile
			rdsTileServer.ServeHTTP(w, r)
		case "hdr": //Valid url is /sds/path/to/file/<filename>?mode=hdr
			headerServer.ServeHTTP(w, r)
		case "raw": //Valid url is /sds/path/to/file/<filename>?mode=raw
			rawServer.ServeHTTP(w, r)
		default:
			log.Println("Unknown Mode")
			w.WriteHeader(400)
			return
		}
	}
}

func setupConfigLogCache() {

	flag.Parse()

	// Load Configuration File
	err := gonfig.GetConf(*configFile, &configuration)
	if err != nil {
		log.Println("Error Reading Config File, ./sdsConfig.json :", err)
		return
	}

	if configuration.Logfile != "" {
		// Open and setup log file
		logFile, err := os.OpenFile(configuration.Logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("Error Reading logfile: ", configuration.Logfile, err)
			return
		}
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	//Create Directories for Cache if they don't exist
	ok := createDirectory(configuration.CacheLocation)
	if !ok {
		log.Println("Error Creating Cache File Directory ", configuration.CacheLocation)
		return
	}
	ok = createDirectory(configuration.CacheLocation + "outputFiles/")
	if !ok {
		log.Println("Error Creating Cache File/outputFiles Directory ", configuration.CacheLocation)
		return
	}
	ok = createDirectory(configuration.CacheLocation + "miniocache/")
	if !ok {
		log.Println("Error Creating Cache File/miniocache Directory ", configuration.CacheLocation)
		return
	}

	// Launch a seperate routine to monitor the cache size
	outputPath := fmt.Sprintf("%s%s", configuration.CacheLocation, "outputFiles/")
	minioPath := fmt.Sprintf("%s%s", configuration.CacheLocation, "miniocache/")
	go checkCache(outputPath, configuration.CheckCacheEvery, configuration.CacheMaxBytes)
	go checkCache(minioPath, configuration.CheckCacheEvery, configuration.CacheMaxBytes)
}

func main() {



	//Used to profile speed
	//if *cpuprofile != "" {
	//	f, err := os.Create(*cpuprofile)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	pprof.StartCPUProfile(f)
	//	defer pprof.StopCPUProfile()
	//}
	//start := time.Now()
	//data:=processRequest("mydata_SI_8192_20000" ,"SI",0,8192,0,0,8192,20000,300,700,"mean","RGBA",-20000,8192,true,"RampColormap")
	//elapsed := time.Since(start)
	//log.Println("Length of Output Data " ,len(data), " processed in: ", elapsed)

	
	setupConfigLogCache()



	// Serve up service on /sds
	log.Println("UI Enabled: ", uiEnabled)
	log.Println("Startup Server on Port: ", configuration.Port)

	sdsServer := &routerServer{}
	http.Handle("/sds/", sdsServer)

	if uiEnabled {
		http.Handle("/ui/", http.StripPrefix("/ui/", handleUI(http.FileServer(&UIAssetWrapper{FileSystem: assetFS()}))))
	} else {
		http.HandleFunc("/ui/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(stubHTML))
		})
	}

	zminzmaxFileMap= make(map[string]Zminzmax)

	msg := ":%d"
	bindAddr := fmt.Sprintf(msg, configuration.Port)

	svr := &http.Server{
		Addr: bindAddr,
		ReadTimeout: 240 * time.Second,
		WriteTimeout: 30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(svr.ListenAndServe())
}
