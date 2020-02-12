package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"
	"github.com/tkanos/gonfig"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"io"
	"math"
	"math/bits"
	"net/http"
	"os"
	//	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var fileZMin float64
var fileZMax float64
var ioMutex = &sync.Mutex{}

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

var configuration Configuration

func createOutput(dataIn []float64, fileFormatString string, zmin, zmax float64, colorMap string) []byte {
	dataOut := new(bytes.Buffer)
	var numColors int = 1000
	//var dataOut []byte
	if fileFormatString == "RGBA" {
		controlColors := getColorConrolPoints(colorMap)
		colorPalette := makeColorPalette(controlColors, numColors)
		colorsPerSpan := (zmax - zmin) / float64(numColors)
		for i := 0; i < len(dataIn); i++ {
			colorIndex := math.Round((dataIn[i] - zmin) / colorsPerSpan)
			colorIndex = math.Min(math.Max(colorIndex, 0), float64(numColors-1)) //Ensure colorIndex is within the colorPalette
			a := 255
			dataOut.WriteByte(byte(colorPalette[int(colorIndex)].red))
			dataOut.WriteByte(byte(colorPalette[int(colorIndex)].green))
			dataOut.WriteByte(byte(colorPalette[int(colorIndex)].blue))
			dataOut.WriteByte(byte(a))
		}
		//log.Println("out_data RGBA" , len(dataOut.Bytes()))
		return dataOut.Bytes()
	} else {
		//log.Println("Processing for Type ",fileFormatString)
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
			log.Error("Unsupported output type")
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
	var bytes_per_atom int = 1
	//var atoms_in_file int= 1
	//var num_slice=make([]int8,atoms_in_file)
	var out_data []float64
	switch string(file_formatstring[1]) {

	case "B":
		bytes_per_atom = 1
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*int8)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "I":
		bytes_per_atom = 2
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*int16)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "L":
		bytes_per_atom = 4
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*int32)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "F":
		bytes_per_atom = 4
		atoms_in_file := len(bytesin) / bytes_per_atom
		out_data = make([]float64, atoms_in_file)
		for i := 0; i < atoms_in_file; i++ {
			num := *(*float32)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "D":
		bytes_per_atom = 8
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
				//Check if leading bit is a zero and add a float or 0 or 1
				if bits.LeadingZeros8(num) > 0 {
					out_data[i*8+j] = float64(0)
				} else {
					out_data[i*8+j] = float64(1)
				}
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
		return stat.Mean(dataIn[:], nil)
	case "max":
		return floats.Max(dataIn[:])
	case "min":
		return floats.Min(dataIn[:])
	case "absmax":
		return math.Abs(floats.Max(dataIn[:]))
	case "first":
		return dataIn[0]
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
		log.Println("Failed to Read Requested Bytes")
		return outData, false
	}
	//log.Println("Read Data Line" , len(out_data))
	return outData, true

}

func applyCXmode(datain []float64, cxmode string) []float64 {

	//var lo_thresh float64=1.0e-20
	out_data := make([]float64, len(datain)/2)
	for i := 0; i < len(datain)-1; i += 2 {
		switch cxmode {
		case "mag":
			out_data[i] = math.Sqrt(datain[i]*datain[i] + datain[i+1]*datain[i+1])
		case "phase":
			out_data[i] = math.Atan2(datain[i+1], datain[i])
		case "real":
			out_data[i] = datain[i]
		case "imag":
			out_data[i] = datain[i+1]
		case "10log":
			out_data[i] = 10 * math.Log10(datain[i]*datain[i]+datain[i+1]*datain[i+1])
		case "20log:":
			out_data[i] = 20 * math.Log10(datain[i]*datain[i]+datain[i+1]*datain[i+1])

			//TODO Add modes besides Magnitude
		}

	}
	return out_data

}

func processline(outData []float64, outLineNum int, done chan bool, reader io.ReadSeeker, fileFormat string, fileDataOffset int, fileXSize int, xstart int, ystart int, xsize int, outxsize int, transform, cxmode string, zet bool) {

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

	//log.Println("file Read info " ,ystart,xstart, firstByte ,bytes_length)
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

	// Finding the max and min of data we processed to get a zmax and zmin if they are not set.
	// Profiling suggests this is computationally intense.
	if !zet {
		localMax := floats.Max(dataToProcess[:])
		fileZMax = math.Max(fileZMax, localMax)

		localMin := floats.Min(dataToProcess[:])
		fileZMin = math.Min(fileZMin, localMin)

	}

	var realData []float64
	if complexFlag {
		realData = applyCXmode(dataToProcess, cxmode)
	} else {
		realData = dataToProcess
	}
	//log.Println("processline", (outxsize),len(real_data),xsize)
	//out_data :=make([]float64,outxsize)
	down_sample_line_inx(realData, outxsize, transform, outData, outLineNum)

	//copy(outData[outLineNum*outxsize:],out_data)
	//log.Println("processline Done", len(out_data))
	done <- true
}

func processRequest(reader io.ReadSeeker, file_format string, fileDataOffset int, fileXSize int, xstart int, ystart int, xsize int, ysize int, outxsize int, outysize int, transform, cxmode string, outputFmt string, zmin, zmax float64, zset bool, colorMap string) []byte {
	var processedData []float64

	var yLinesPerOutput float64 = float64(ysize) / float64(outysize)
	var yLinesPerOutputCeil int = int(math.Ceil(yLinesPerOutput))

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
			go processline(xThinData, inputLine-startLine, done, reader, file_format, fileDataOffset, fileXSize, xstart, inputLine, xsize, outxsize, transform, cxmode, zset)

		}
		//Wait until all the lines have finished before moving on
		for i := 0; i < numLines; i++ {
			<-done
		}

		// Thin in y direction the subsset of lines that have now been processed in x
		yThinData := downSampleLineInY(xThinData, outxsize, transform)
		//log.Println("Thin Y data is currently ", len(yThinData))

		processedData = append(processedData, yThinData...)
		//log.Println("processedData is currently ", len(processedData))

	}

	if !zset {
		zmin = fileZMin
		zmax = fileZMax
	}
	outData := createOutput(processedData, outputFmt, zmin, zmax, colorMap)
	return outData
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
		log.Error("Unsupported Location Type")
		return nil, "", false
	}

}

type rdsServer struct{}

func (s *rdsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var data []byte
	var inCache bool

	var file_format string
	var file_type int
	var fileXSize int
	var filexstart, filexdelta, fileystart, fileydelta, data_offset float64
	var fileDataOffset int

	// Get Rest of URL Parameters
	x1, ok := getURLArgumentInt(r, "x1")
	if !ok {
		log.Println("X1 Missing. Required Field")
		w.WriteHeader(400)
		return
	}
	y1, ok := getURLArgumentInt(r, "y1")
	if !ok {
		log.Println("Y1 Missing. Required Field")
		w.WriteHeader(400)
		return
	}
	x2, ok := getURLArgumentInt(r, "x2")
	if !ok {
		log.Println("X2 Missing. Required Field")
		w.WriteHeader(400)
		return
	}
	y2, ok := getURLArgumentInt(r, "y2")
	if !ok {
		log.Println("Y2 Missing. Required Field")
		w.WriteHeader(400)
		return
	}
	ystart := int(math.Min(float64(y1), float64(y2)))
	xstart := int(math.Min(float64(x1), float64(x2)))
	xsize := int(math.Abs(float64(x2) - float64(x1)))
	ysize := int(math.Abs(float64(y2) - float64(y1)))

	outxsize, ok := getURLArgumentInt(r, "outxsize")
	if !ok {
		log.Println("outxsize Missing. Required Field")
		w.WriteHeader(400)
		return
	}

	outysize, ok := getURLArgumentInt(r, "outysize")
	if !ok {
		log.Println("outysize Missing. Required Field")
		w.WriteHeader(400)
		return
	}
	transform, ok := getURLArgumentString(r, "transform")
	if !ok {
		log.Println("transform Missing. Required Field")
		w.WriteHeader(400)
		return
	}

	cxmode, ok := getURLArgumentString(r, "cxmode")
	if !ok {
		cxmode = "mag"
	}

	//log.Println("Reported file_data_size", file_data_size)

	zminInt, zminSet := getURLArgumentInt(r, "zmin")
	var zmin float64
	if !zminSet {
		log.Println("Zmin Not Specified. Will estimate from file Selection")
		zmin = 0
	} else {
		zmin = float64(zminInt)
	}

	zmaxInt, zmaxSet := getURLArgumentInt(r, "zmax")
	var zmax float64
	if !zmaxSet {
		log.Println("Zmax Not Specified. Will estimate from file Selection")
		zmax = 0
	} else {
		zmax = float64(zmaxInt)
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
	data, inCache = getDataFromCache(cacheFileName, "outputFiles/")

	if !inCache { // If the output is not already in the cache then read the data file and do the processing.

		reader, fileName, succeed := openDataSource(r.URL.Path)
		if !succeed {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(fileName, ".tmp") || strings.Contains(fileName, ".prm") {
			log.Println("Processing File as Blue File")
			file_format, file_type, fileXSize, filexstart, filexdelta, fileystart, fileydelta, data_offset, _ = processBlueFileHeader(reader)
			fileDataOffset = int(data_offset)
			if file_type != 2000 {
				log.Println("Only Supports type 2000 Bluefiles")
				w.WriteHeader(400)
				return
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
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}

		outputFmt, ok := getURLArgumentString(r, "outfmt")
		if !ok {
			log.Println("Outformat Not Specified. Setting Equal to Input Format")
			outputFmt = file_format

		}

		data = processRequest(reader, file_format, fileDataOffset, fileXSize, xstart, ystart, xsize, ysize, outxsize, outysize, transform, cxmode, outputFmt, zmin, zmax, zset, colorMap)
		go putItemInCache(cacheFileName, "outputFiles/", data)
	}

	elapsed := time.Since(start)
	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)

	if !zset {
		zmin = fileZMin
		zmax = fileZMax
	}

	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(outxsize)
	outysizeStr := strconv.Itoa(outysize)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%.0f", zmin))
	w.Header().Add("zmax", fmt.Sprintf("%.0f", zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileydelta))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

type fileHeaderServer struct{}

func (s *fileHeaderServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	reader, fileName, succeed := openDataSource(r.URL.Path)
	if !succeed {
		log.Error("Error Reading from Data Source")
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
		blueShort.Ape = int(blueShort.Subsize)
		blueShort.Bpe = float64(blueShort.Ape) * blueShort.Bpa
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
	w.WriteHeader(http.StatusOK)

	w.Write(returnbytes)
}

type routerServer struct{}

func (s *routerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("%s", r.URL)
	//Valid url is /sds/<filename>/rds or //Valid url is /sds/<filename>
	rdsServer := &rdsServer{}
	headerServer := &fileHeaderServer{}

	mode, ok := getURLArgumentString(r, "mode")
	if !ok {
		log.Println("Mode Missing. Required Field")
		w.WriteHeader(400)
		return
	}

	switch mode {
	case "rds": //Valid url is /sds/path/to/file/<filename>?mode=rds
		rdsServer.ServeHTTP(w, r)
	case "hdr": //Valid url is /sds/path/to/file/<filename>?mode=hdr
		headerServer.ServeHTTP(w, r)
	default:
		log.Println("Unknown Mode")
		w.WriteHeader(400)
		return
	}
}

func SetupLogging(debug bool) {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02T15:04:05.000"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	log.SetLevel(log.DebugLevel)
}

//func RunProfile(cpuprofile string) (time.Duration, int) {
//	f, err := os.Create(cpuprofile)
//	if err != nil {
//		log.Fatal(err)
//	}
//	pprof.StartCPUProfile(f)
//	defer pprof.StopCPUProfile()
//
//	start := time.Now()
//	data := processRequest(
//		"mydata_SI_8192_20000",
//		"SI",
//		0,
//		8192,
//		0,
//		0,
//		8192,
//		20000,
//		300,
//		700,
//		"mean",
//		"RGBA",
//		-20000,
//		8192,
//		true,
//		"RampColormap",
//	)
//	return time.Since(start), len(data)
//}

func main() {
	// Setup CLI flags
	// cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	configFile := flag.String("config", "./sdsConfig.json", "Location of Config File")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Used to profile speed
	//	if *cpuprofile != "" {
	//		elapsed, dataLen := RunProfile(*cpuprofile)
	//		log.Printf(
	//			"Length of Output Data %d processed in: %f",
	//			dataLen,
	//			elapsed,
	//		)
	//		return
	//	}
	SetupLogging(*debug)

	// Load Configuration File
	err := gonfig.GetConf(*configFile, &configuration)
	if err != nil {
		log.Fatalf("Error reading config file %s: %s", *configFile, err)
	}

	if configuration.Logfile != "" {
		// Open and setup log file
		logFile, err := os.OpenFile(configuration.Logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error Reading logfile: %s %s", configuration.Logfile, err)
		}
		log.SetOutput(logFile)
	}

	// Launch a seperate routine to monitor the cache size
	outputPath := fmt.Sprintf("%s%s", configuration.CacheLocation, "outputFiles/")
	minioPath := fmt.Sprintf("%s%s", configuration.CacheLocation, "miniocache/")
	go checkCache(outputPath, configuration.CheckCacheEvery, configuration.CacheMaxBytes)
	go checkCache(minioPath, configuration.CheckCacheEvery, configuration.CacheMaxBytes)

	// Serve up service on /sds
	log.Printf("Listening on :%d...", configuration.Port)
	s := &routerServer{}
	http.Handle("/sds/", s)
	msg := ":%d"
	result := fmt.Sprintf(msg, configuration.Port)
	log.Fatal(http.ListenAndServe(result, nil))
}
