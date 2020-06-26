package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/minio/minio-go/v6"
	"github.com/tkanos/gonfig"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"

	//	"runtime/pprof"
	//	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var ioMutex = &sync.Mutex{}
var zminmaxMutex = &sync.Mutex{}
var zminmaxtileMutex = &sync.Mutex{}
var uiEnabled = true // set to false by stub_asset if the ui build isn't included
var stubHTML = ""    // set to HTML by stub_asset if the ui build isn't included
var configuration Configuration

func createOutput(dataIn []float64, fileFormatString string, zmin, zmax float64, colorMap string) []byte {
	// for i := 0; i < len(dataIn); i++ {
	// 	if math.IsNaN(dataIn[i]) {
	// 		log.Println("createOutput NaN", i)
	// 	}
	// }

	dataOut := new(bytes.Buffer)
	var numColors int = 1000
	//var dataOut []byte
	if fileFormatString == "RGBA" {
		controlColors := getColorConrolPoints(colorMap)
		colorPalette := makeColorPalette(controlColors, numColors)
		if zmax != zmin {
			colorsPerSpan := (zmax - zmin) / float64(numColors)
			for i := 0; i < len(dataIn); i++ {
				colorIndex := math.Round((dataIn[i]-zmin)/colorsPerSpan) - 1
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
		log.Println("Creating Output of Type ", fileFormatString)
		switch string(fileFormatString[1]) {
		case "B":
			var numSlice = make([]int8, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int8(math.Round(dataIn[i]))
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		case "I":
			var numSlice = make([]int16, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int16(math.Round(dataIn[i]))
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)

			check(err)

		case "L":
			var numSlice = make([]int32, len(dataIn))
			for i := 0; i < len(numSlice); i++ {
				numSlice[i] = int32(math.Round(dataIn[i]))
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

		case "P":
			extraBits := len(dataIn) % 8
			for extraBit := 0; extraBit < extraBits; extraBits++ { //Pad zeros to make the number of elements divisable by 8 so it can be packed into a byte
				dataIn = append(dataIn, 0)
			}
			numBytes := len(dataIn) / 8
			var numSlice = make([]uint8, numBytes)
			for i := 0; i < len(numSlice); i++ {
				for j := 0; j < 8; j++ {
					var bit uint8
					if dataIn[i*8+j] > 0 { //SP Data can only be 0 or 1, so if values is greater than 0, make it a 1.
						bit = 1
					} else {
						bit = 0
					}
					numSlice[i] = (numSlice[i] << 1) | bit
				}

			}
			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
			check(err)

		default:
			log.Println("Unsupported output type")
		}
		//log.Println("out_data" , len(dataOut.Bytes()))

		return dataOut.Bytes()
	}

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
		absnums := make([]float64, len(dataIn))
		for i := 0; i < len(dataIn); i++ {
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
	default: //Default to first if bad value.
		log.Println("Unknown transform", transform, "using first")
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
	if xelementsperoutput > 1 { // Expansion
		for x := 0; x < outxsize; x++ {
			var startelement int
			var endelement int
			if x != (outxsize - 1) { // Not last element
				startelement = int(math.Round(float64(x) * xelementsperoutput))
				endelement = int(math.Round(float64(x+1) * xelementsperoutput))
			} else { // Last element, work backwards
				endelement = len(datain)
				startelement = endelement - int(math.Ceil(xelementsperoutput))
			}

			outData[outLineNum*outxsize+x] = doTransform(datain[startelement:endelement], transform)

		}
	} else { // Expand Data by repeating input values into output

		for x := 0; x < outxsize; x++ {
			index := int(math.Floor(float64(x) * xelementsperoutput))
			outData[outLineNum*outxsize+x] = datain[index]
		}
	}
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
			default: //Defaults to "real"
				log.Println("Unkown cxmode", cxmode, "defaulting to Re")
				outData[i/2] = datain[i]
			}

		}
		return outData
	} else {

		switch cxmode {
		case "Ma":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				outData[i] = math.Sqrt(datain[i] * datain[i])
			}
			return outData
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
				mag2 := math.Max(datain[i], loThresh)
				outData[i] = 10 * math.Log10(mag2)
			}
			return outData
		case "L2":
			outData := make([]float64, len(datain))
			for i := 0; i < len(datain); i++ {
				mag2 := math.Max(datain[i], loThresh)
				outData[i] = 20 * math.Log10(mag2)
			}
			return outData

		}
		return datain //Defaults to "Real" or passthrough

	}
}

func processline(outData []float64, outLineNum int, done chan bool, dataRequest rdsRequest) {
	bytesPerAtom, complexFlag := getFileTypeInfo(dataRequest.FileFormat)

	bytesPerElement := bytesPerAtom
	if complexFlag {
		bytesPerElement = bytesPerElement * 2
	}

	firstDataByte := float64(dataRequest.Ystart*dataRequest.FileXSize+dataRequest.Xstart) * bytesPerElement
	firstByteInt := int(math.Floor(firstDataByte))

	bytesLength := float64(dataRequest.Xsize)*bytesPerElement + (firstDataByte - float64(firstByteInt))
	bytesLengthInt := int(math.Ceil(bytesLength))
	filedata, _ := getBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt)
	dataToProcess := convertFileData(filedata, dataRequest.FileFormat)

	//If the data is SP then we might have processed a few more bits than we actually needed on both sides, so reassign data_to_process to correctly point to the numbers of interest
	if bytesPerAtom < 1 {
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
		realData = applyCXmode(dataToProcess, dataRequest.Cxmode, true)
	} else {
		if dataRequest.CxmodeSet {
			realData = applyCXmode(dataToProcess, dataRequest.Cxmode, false)
		} else {
			realData = dataToProcess
		}

	}

	down_sample_line_inx(realData, dataRequest.Outxsize, dataRequest.Transform, outData, outLineNum)
	done <- true
}

func processRequest(dataRequest rdsRequest) []byte {
	var processedData []float64

	var yLinesPerOutput float64 = float64(dataRequest.Ysize) / float64(dataRequest.Outysize)
	var yLinesPerOutputCeil int = int(math.Ceil(yLinesPerOutput))
	log.Println("processRequest:", dataRequest.FileXSize, dataRequest.Xstart, dataRequest.Ystart, dataRequest.Xsize, dataRequest.Ysize, dataRequest.Outxsize, dataRequest.Outysize)
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
			var lineRequest rdsRequest
			lineRequest = dataRequest
			lineRequest.Ystart = inputLine
			go processline(xThinData, inputLine-startLine, done, lineRequest)

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
		yThinData := downSampleLineInY(xThinData, dataRequest.Outxsize, dataRequest.Transform)
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

	outData := createOutput(processedData, dataRequest.OutputFmt, dataRequest.Zmin, dataRequest.Zmax, dataRequest.ColorMap)
	return outData
}

func processLineRequest(dataRequest rdsRequest, cutType string) []byte {
	bytesPerAtom, complexFlag := getFileTypeInfo(dataRequest.FileFormat)

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
		filedata, _ = getBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+firstByteInt, bytesLengthInt)
		dataToProcess = convertFileData(filedata, dataRequest.FileFormat)
		//If the data is SP then we might have processed a few more bits than we actually needed on both sides, so reassign data_to_process to correctly point to the numbers of interest
		if bytesPerAtom < 1 {
			dataStartBit := int(math.Mod(firstDataByte, 1) * 8)
			dataEndBit := int(math.Mod(bytesLength, 1) * 8)
			var extraBits int = 0
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
			data, _ := getBytesFromReader(dataRequest.Reader, dataRequest.FileDataOffset+dataByteInt, int(bytesPerElement))
			filedata = append(filedata, data...)
		}
		dataToProcess = convertFileData(filedata, dataRequest.FileFormat)
		log.Println("Got data from file for y cut", len(dataToProcess))

	}

	var realData []float64
	if complexFlag {
		realData = applyCXmode(dataToProcess, dataRequest.Cxmode, true)
	} else {
		if dataRequest.CxmodeSet {
			realData = applyCXmode(dataToProcess, dataRequest.Cxmode, false)
		} else {
			realData = dataToProcess
		}

	}

	//Output data will be x and z data of variable length up to Xsize. Allocation with size 0 but with a capacity. The x arrary will be used for both piece of data at the end.
	xThinData := make([]int16, 0, len(realData)*2)
	zThinData := make([]int16, 0, len(realData))

	xratio := float64(len(realData)) / float64(dataRequest.Outxsize-1)
	zratio := float64((dataRequest.Zmax - dataRequest.Zmin)) / float64(dataRequest.Outzsize-1)
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

func openDataSource(url string, urlPosition int) (io.ReadSeeker, string, bool) {

	pathData := strings.Split(url, "/")
	locationName := pathData[urlPosition]
	var urlPath string = ""
	for i := urlPosition + 1; i < len(pathData)-1; i++ {
		urlPath = urlPath + pathData[i] + "/"
	}

	fileName := pathData[len(pathData)-1]
	var currentLocation Location
	for i := range configuration.LocationDetails {
		if configuration.LocationDetails[i].LocationName == locationName {
			currentLocation = configuration.LocationDetails[i]
		}
	}
	if len(currentLocation.Path) > 0 {
		if string(currentLocation.Path[len(currentLocation.Path)-1]) != "/" {
			currentLocation.Path += "/"
		}
	}
	switch currentLocation.LocationType {
	case "localFile":

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
		cacheFileName := urlToCacheFileName("sds", currentLocation.MinioBucket+fullFilepath)
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
			if int64(numRead) != fi.Size || !(readerr == nil || readerr == io.EOF) {
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
		log.Println("Unsupported Location Type", currentLocation.LocationName, currentLocation.LocationType)
		return nil, "", false
	}

}

func (request *rdsRequest) getQueryParams(r *http.Request) {
	var ok bool
	// Get URL Query Params
	request.Transform, ok = getURLQueryParamString(r, "transform")
	if !ok {
		request.Transform = "first"
	}
	request.SubsizeSet = true
	request.Subsize, ok = getURLQueryParamInt(r, "subsize")
	if !ok {
		request.Subsize = 1
		request.SubsizeSet = false
	}
	if request.Subsize < 1 {
		log.Println("Subsize Invalid. Ignoring")
		request.Subsize = 1
		request.SubsizeSet = false
	}
	request.CxmodeSet = true
	request.Cxmode, ok = getURLQueryParamString(r, "cxmode")
	if !ok {
		request.Cxmode = "Re"
		request.CxmodeSet = false
	}
	var zminSet, zmaxSet bool
	request.Zmin, zminSet = getURLQueryParamFloat(r, "zmin")
	if !zminSet {
		request.Zmin = 0
	}
	request.Zmax, zmaxSet = getURLQueryParamFloat(r, "zmax")
	if !zmaxSet {
		request.Zmax = 0
	}
	request.Zset = (zmaxSet && zminSet)
	request.ColorMap, ok = getURLQueryParamString(r, "colormap")
	if !ok {
		log.Println("colorMap Not Specified.Defaulting to RampColormap")
		request.ColorMap = "RampColormap"
	}
	request.OutputFmt, ok = getURLQueryParamString(r, "outfmt")
	if !ok {
		log.Println("Outformat Not Specified. Setting Equal to Input Format")
		request.OutputFmt = "RGBA"

	}
}

func (request *rdsRequest) findZminMax() {
	start := time.Now()
	zminmaxtileMutex.Lock()
	zminmax, ok := zminzmaxFileMap[request.FileName+request.Cxmode]
	if ok {
		request.Zmin = zminmax.Zmin
		request.Zmax = zminmax.Zmax
	} else {
		var zminmaxRequest rdsRequest
		zminmaxRequest = *request
		zminmaxRequest.Ysize = 1
		zminmaxRequest.Xsize = zminmaxRequest.FileXSize
		zminmaxRequest.Xstart = 0
		zminmaxRequest.Outysize = 1
		zminmaxRequest.Outxsize = 1
		zminmaxRequest.OutputFmt = "SD"
		bytesPerAtom, complexFlag := getFileTypeInfo(request.FileFormat)
		bytesPerElement := bytesPerAtom
		if complexFlag {
			bytesPerElement = bytesPerElement * 2
		}
		log.Println("Computing Zminmax", bytesPerElement, request.FileXSize, request.FileYSize, configuration.MaxBytesZminZmax)
		if (int(float64(request.FileXSize*request.FileYSize) * (bytesPerElement))) < configuration.MaxBytesZminZmax { // File is small enough, look at entire file for Zmax/Zmin
			log.Println("Computing Zmax/Zmin on whole file, not previously computed")
			min := make([]float64, request.FileYSize)
			max := make([]float64, request.FileYSize)
			done := make(chan bool, 1)
			for line := 0; line < request.FileYSize; line++ {
				zminmaxRequest.Ystart = line
				zminmaxRequest.Transform = "min"
				go processline(min, line, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go processline(max, line, done, zminmaxRequest)
			}
			for i := 0; i < request.FileYSize*2; i++ {
				<-done
			}
			request.Zmin = floats.Min(min)
			request.Zmax = floats.Max(max)
			zminzmaxFileMap[request.FileName+request.Cxmode] = Zminzmax{request.Zmin, request.Zmax}
		} else if request.FileYSize == 1 { //If the file is large but only has one line then we need to break it into section in the x direction.
			log.Println("Computing Zmax/Zmin on section of 1D file, not previously computed")
			numSubSections := 4
			min := make([]float64, numSubSections)
			max := make([]float64, numSubSections)
			done := make(chan bool, 1)
			spaceBytes := (float64(request.FileXSize) * bytesPerElement) - float64(configuration.MaxBytesZminZmax)
			elementsPerSpace := int((spaceBytes / bytesPerElement)) / (numSubSections - 1)
			elementsPerSection := int(configuration.MaxBytesZminZmax / numSubSections)

			zminmaxRequest.Xsize = elementsPerSection
			// First section of the file
			zminmaxRequest.Xstart = 0
			zminmaxRequest.Transform = "min"
			go processline(min, 0, done, zminmaxRequest)
			zminmaxRequest.Transform = "max"
			go processline(max, 0, done, zminmaxRequest)
			// Middle Sections of the file
			for section := 1; section < numSubSections-1; section++ {
				zminmaxRequest.Xstart = section * (elementsPerSection + elementsPerSpace)
				zminmaxRequest.Transform = "min"
				go processline(min, section, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go processline(max, section, done, zminmaxRequest)

			}

			// Last Section of the file
			zminmaxRequest.Xstart = request.FileXSize - elementsPerSection
			zminmaxRequest.Transform = "min"
			go processline(min, numSubSections-1, done, zminmaxRequest)
			zminmaxRequest.Transform = "max"
			go processline(max, numSubSections-1, done, zminmaxRequest)
			for i := 0; i < numSubSections*2; i++ {
				<-done
			}
			request.Zmin = floats.Min(min)
			request.Zmax = floats.Max(max)
			zminzmaxFileMap[request.FileName+request.Cxmode] = Zminzmax{request.Zmin, request.Zmax}

		} else { // If file is large and has multiple lines then check the first, last, and a number of middles lines
			numMiddlesLines := int(math.Max(float64((configuration.MaxBytesZminZmax/request.FileXSize)-2), 0))
			log.Println("Computing Zmax/Zmin on sampling of file, not previously computed. Number of middle lines:", numMiddlesLines)
			min := make([]float64, 2+numMiddlesLines)
			max := make([]float64, 2+numMiddlesLines)
			done := make(chan bool, 1)
			numRequested := 0
			// Process Min and Max of first line
			zminmaxRequest.Ystart = 0
			zminmaxRequest.Transform = "min"
			go processline(min, 0, done, zminmaxRequest)
			zminmaxRequest.Transform = "max"
			go processline(max, 0, done, zminmaxRequest)
			numRequested += 2

			//Process Min and Max of last line
			zminmaxRequest.Ystart = request.FileYSize - 1
			if zminmaxRequest.Ystart != 0 { // If the last line is the first line, don't do it again.
				zminmaxRequest.Transform = "min"
				go processline(min, 1, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go processline(max, 1, done, zminmaxRequest)
				numRequested += 2
			}

			//Process Min and Max from lines evenly spaced in the middle
			for i := 0; i < numMiddlesLines; i++ {
				zminmaxRequest.Ystart = int(((request.FileYSize) / numMiddlesLines) * i)
				zminmaxRequest.Transform = "min"
				go processline(min, i+2, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go processline(max, i+2, done, zminmaxRequest)
				numRequested += 2
			}
			for i := 0; i < numRequested; i++ {
				<-done
			}
			request.Zmin = floats.Min(min)
			request.Zmax = floats.Max(max)
			zminzmaxFileMap[request.FileName+request.Cxmode] = Zminzmax{request.Zmin, request.Zmax}

		}
		elapsed := time.Since(start)
		log.Println("Found Zmin, Zmax to be", request.Zmin, request.Zmax, " in ", elapsed)

	}
	zminmaxtileMutex.Unlock()
}

type rdsServer struct{}

func (s *rdsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var inCache bool
	var ok bool
	var rdsRequest rdsRequest

	//Get URL Parameters
	//url - /sds/rds/x1/y1/x2/y2/outxsize/outysize
	rdsRequest.X1, ok = getURLArgumentInt(r.URL.Path, 3)
	if !ok || rdsRequest.X1 < 0 {
		log.Println("X1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.Y1, ok = getURLArgumentInt(r.URL.Path, 4)
	if !ok || rdsRequest.Y1 < 0 {
		log.Println("Y1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.X2, ok = getURLArgumentInt(r.URL.Path, 5)
	if !ok || rdsRequest.X2 < 0 {
		log.Println("X2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.Y2, ok = getURLArgumentInt(r.URL.Path, 6)
	if !ok || rdsRequest.Y2 < 0 {
		log.Println("Y2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.Outxsize, ok = getURLArgumentInt(r.URL.Path, 7)
	if !ok || rdsRequest.Outxsize < 1 {
		log.Println("outxsize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	rdsRequest.Outysize, ok = getURLArgumentInt(r.URL.Path, 8)
	if !ok || rdsRequest.Outysize < 1 {
		log.Println("outysize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.getQueryParams(r)

	rdsRequest.computeRequestSizes()

	if rdsRequest.Xsize < 1 || rdsRequest.Ysize < 1 {
		log.Println("Bad Xsize or ysize. xsize: ", rdsRequest.Xsize, " ysize: ", rdsRequest.Ysize)
		w.WriteHeader(400)
		return
	}

	log.Println("RDS Request params xstart, ystart, xsize, ysize, outxsize, outysize:", rdsRequest.Xstart, rdsRequest.Ystart, rdsRequest.Xsize, rdsRequest.Ysize, rdsRequest.Outxsize, rdsRequest.Outysize)

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
		rdsRequest.Reader, rdsRequest.FileName, ok = openDataSource(r.URL.Path, 9)
		if !ok {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
			rdsRequest.processBlueFileHeader()
			if rdsRequest.SubsizeSet {
				rdsRequest.FileXSize = rdsRequest.Subsize

			} else {
				if rdsRequest.FileType == 1000 {
					log.Println("For type 1000 files, a subsize needs to be set")
					w.WriteHeader(400)
					return
				}
			}
			rdsRequest.computeYSize()
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}

		if rdsRequest.Xsize > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X size greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.X1 > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X1 greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.X2 > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X2 greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.Y1 > rdsRequest.FileYSize {
			log.Println("Invalid Request. Requested Y1 greater than file Y size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.Y2 > rdsRequest.FileYSize {
			log.Println("Invalid Request. Requested Y2 greater than file Y size")
			w.WriteHeader(400)
			return
		}

		//If Zmin and Zmax were not explitily given then compute
		if !rdsRequest.Zset && rdsRequest.OutputFmt == "RGBA" {
			rdsRequest.findZminMax()
		}

		data = processRequest(rdsRequest)
		if *useCache {
			go putItemInCache(cacheFileName, "outputFiles/", data)
		}

		// Store MetaData of request off in cache
		var fileMData fileMetaData
		fileMData.Outxsize = rdsRequest.Outxsize
		fileMData.Outysize = rdsRequest.Outysize
		fileMData.Filexstart = rdsRequest.Filexstart
		fileMData.Filexdelta = rdsRequest.Filexdelta
		fileMData.Fileystart = rdsRequest.Fileystart
		fileMData.Fileydelta = rdsRequest.Fileydelta
		fileMData.Xstart = rdsRequest.Xstart
		fileMData.Ystart = rdsRequest.Ystart
		fileMData.Xsize = rdsRequest.Xsize
		fileMData.Ysize = rdsRequest.Ysize
		fileMData.Zmin = rdsRequest.Zmin
		fileMData.Zmax = rdsRequest.Zmax

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
	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

type rdsTileServer struct{}

func (s *rdsTileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var inCache, ok bool

	var tileRequest rdsRequest
	tileRequest.TileRequest = true

	// Get URL Parameters
	//url - /sds/rdstile/tileXSize/tileYSize/decxMode/decYMode/tileX/tileY/locationName
	allowedTileSizes := [5]int{100, 200, 300, 400, 500}
	tileRequest.TileXSize, ok = getURLArgumentInt(r.URL.Path, 3)
	if !ok || !intInSlice(tileRequest.TileXSize, allowedTileSizes[:]) {
		log.Println("tileXSize must be in one of: 100,200,300,400,500", tileRequest.TileXSize)
		w.WriteHeader(400)
		return
	}
	tileRequest.TileYSize, ok = getURLArgumentInt(r.URL.Path, 4)
	if !ok || !intInSlice(tileRequest.TileYSize, allowedTileSizes[:]) {
		log.Println("tileYSize must be in one of: 100,200,300,400,500", tileRequest.TileYSize)
		w.WriteHeader(400)
		return
	}
	tileRequest.DecXMode, ok = getURLArgumentInt(r.URL.Path, 5)
	if !ok || tileRequest.DecXMode < 0 || tileRequest.DecXMode > 10 {
		log.Println("decXMode Bad or out of range 0 to 10. got:", tileRequest.DecXMode)
		w.WriteHeader(400)
		return
	}
	tileRequest.DecYMode, ok = getURLArgumentInt(r.URL.Path, 6)
	if !ok || tileRequest.DecYMode < 0 || tileRequest.DecYMode > 10 {
		log.Println("decYMode Bad or out of range 0 to 10. got:", tileRequest.DecYMode)
		w.WriteHeader(400)
		return
	}
	tileRequest.TileX, ok = getURLArgumentInt(r.URL.Path, 7)
	if !ok || tileRequest.TileX < 0 {
		log.Println("tileX must be great than zero")
		w.WriteHeader(400)
		return
	}
	tileRequest.TileY, ok = getURLArgumentInt(r.URL.Path, 8)
	if !ok || tileRequest.TileY < 0 {
		log.Println("tileY must be great than zero")
		w.WriteHeader(400)
		return
	}

	tileRequest.getQueryParams(r)

	tileRequest.computeTileSizes()

	if tileRequest.Xsize < 1 || tileRequest.Ysize < 1 {
		log.Println("Bad Xsize or ysize. xsize: ", tileRequest.Xsize, " ysize: ", tileRequest.Ysize)
		w.WriteHeader(400)
		return
	}

	log.Println("Tile Mode: params xstart, ystart, xsize, ysize, outxsize, outysize:", tileRequest.Xstart, tileRequest.Ystart, tileRequest.Xsize, tileRequest.Ysize, tileRequest.Outxsize, tileRequest.Outysize)

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
		tileRequest.Reader, tileRequest.FileName, ok = openDataSource(r.URL.Path, 9)
		if !ok {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(tileRequest.FileName, ".tmp") || strings.Contains(tileRequest.FileName, ".prm") {
			tileRequest.processBlueFileHeader()

			if tileRequest.SubsizeSet {
				tileRequest.FileXSize = tileRequest.Subsize

			} else {
				if tileRequest.FileType == 1000 {
					log.Println("For type 1000 files, a subsize needs to be set")
					w.WriteHeader(400)
					return
				}
			}
			tileRequest.computeYSize()
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}

		if tileRequest.Xstart >= tileRequest.FileXSize || tileRequest.Ystart >= tileRequest.FileYSize {
			log.Println("Invalid Tile Request. ", tileRequest.Xstart, tileRequest.FileXSize, tileRequest.Ystart, tileRequest.FileYSize)
			w.WriteHeader(400)
			return
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
			log.Println("Invalid Request. Requested X size greater than file X size")
			w.WriteHeader(400)
			return
		}

		//If Zmin and Zmax were not explitily given then compute
		if !tileRequest.Zset {
			tileRequest.findZminMax()
		}
		// Now that all the parameters have been computed as needed, perform the actual request for data transformation.
		data = processRequest(tileRequest)
		if *useCache {
			go putItemInCache(cacheFileName, "outputFiles/", data)
		}

		// Store MetaData of request off in cache
		var fileMData fileMetaData
		fileMData.Outxsize = tileRequest.Outxsize
		fileMData.Outysize = tileRequest.Outysize
		fileMData.Filexstart = tileRequest.Filexstart
		fileMData.Filexdelta = tileRequest.Filexdelta
		fileMData.Fileystart = tileRequest.Fileystart
		fileMData.Fileydelta = tileRequest.Fileydelta
		fileMData.Xstart = tileRequest.Xstart
		fileMData.Ystart = tileRequest.Ystart
		fileMData.Xsize = tileRequest.Xsize
		fileMData.Ysize = tileRequest.Ysize
		fileMData.Zmin = tileRequest.Zmin
		fileMData.Zmax = tileRequest.Zmax

		//var marshalError error
		fileMDataJSON, marshalError := json.Marshal(fileMData)
		if marshalError != nil {
			log.Println("Error Encoding metadata file to cache", marshalError)
			w.WriteHeader(400)
			return
		}
		putItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)

	} else {
		log.Println("Request in cache - returning data from cache")
	}

	elapsed := time.Since(start)
	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)

	//var fileMData rdsRequest
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
	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

type ldsServer struct{}

func (s *ldsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var inCache bool
	var ok bool
	var rdsRequest rdsRequest

	//Get URL Parameters
	//url - /sds/lds/x1/x2/outxsize/outzsize

	rdsRequest.X1, ok = getURLArgumentInt(r.URL.Path, 3)
	if !ok || rdsRequest.X1 < 0 {
		log.Println("X1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.X2, ok = getURLArgumentInt(r.URL.Path, 4)
	if !ok || rdsRequest.X2 < 0 {
		log.Println("X2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	rdsRequest.Outxsize, ok = getURLArgumentInt(r.URL.Path, 5)
	if !ok || rdsRequest.Outxsize < 1 {
		log.Println("outxsize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	rdsRequest.Outzsize, ok = getURLArgumentInt(r.URL.Path, 6)
	if !ok || rdsRequest.Outzsize < 1 {
		log.Println("outzsize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	rdsRequest.getQueryParams(r)

	rdsRequest.computeRequestSizes()

	rdsRequest.Ystart = 0
	rdsRequest.Ysize = 1

	if rdsRequest.Xsize < 1 {
		log.Println("Bad Xsize: ", rdsRequest.Xsize)
		w.WriteHeader(400)
		return
	}

	log.Println("LDS Request params xstart, xsize, outxsize, outzsize:", rdsRequest.Xstart, rdsRequest.Xsize, rdsRequest.Outxsize, rdsRequest.Outzsize)

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
		rdsRequest.Reader, rdsRequest.FileName, ok = openDataSource(r.URL.Path, 7)
		if !ok {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
			rdsRequest.processBlueFileHeader()
			if rdsRequest.FileType != 1000 {
				log.Println("Line Plots only support Type 100 files.")
				w.WriteHeader(400)
				return
			}
			rdsRequest.FileXSize = int(float64(rdsRequest.FileDataSize) / bytesPerAtomMap[string(rdsRequest.FileFormat[1])])
			rdsRequest.FileYSize = 1
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}
		// Check Request against File Size
		if rdsRequest.Xsize > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X size greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.X1 > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X1 greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.X2 > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X2 greater than file X size")
			w.WriteHeader(400)
			return
		}

		//If Zmin and Zmax were not explitily given then compute
		if !rdsRequest.Zset {
			rdsRequest.findZminMax()
		}

		data = processLineRequest(rdsRequest, "lds")

		if *useCache {
			go putItemInCache(cacheFileName, "outputFiles/", data)
		}

		// Store MetaData of request off in cache
		var fileMData fileMetaData
		fileMData.Outxsize = rdsRequest.Outxsize
		fileMData.Outysize = rdsRequest.Outysize
		fileMData.Outzsize = rdsRequest.Outzsize
		fileMData.Filexstart = rdsRequest.Filexstart
		fileMData.Filexdelta = rdsRequest.Filexdelta
		fileMData.Fileystart = rdsRequest.Fileystart
		fileMData.Fileydelta = rdsRequest.Fileydelta
		fileMData.Xstart = rdsRequest.Xstart
		fileMData.Ystart = rdsRequest.Ystart
		fileMData.Xsize = rdsRequest.Xsize
		fileMData.Ysize = rdsRequest.Ysize
		fileMData.Zmin = rdsRequest.Zmin
		fileMData.Zmax = rdsRequest.Zmax

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
	outzsizeStr := strconv.Itoa(fileMDataCache.Outzsize)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("outzsize", outzsizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
	w.WriteHeader(http.StatusOK)

	w.Write(data)

}

type rdsxyCutServer struct{}

func (s *rdsxyCutServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var inCache bool
	var ok bool
	var rdsRequest rdsRequest

	//Get URL Parameters
	//url - /sds/rdsxcut/x1/y1/x2/y2/outxsize/outzsize
	cutType := strings.Split(r.URL.Path, "/")[2] //rdsxcut or rdsycut

	rdsRequest.X1, ok = getURLArgumentInt(r.URL.Path, 3)
	if !ok || rdsRequest.X1 < 0 {
		log.Println("X1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.Y1, ok = getURLArgumentInt(r.URL.Path, 4)
	if !ok || rdsRequest.Y1 < 0 {
		log.Println("Y1 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.X2, ok = getURLArgumentInt(r.URL.Path, 5)
	if !ok || rdsRequest.X2 < 0 {
		log.Println("X2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.Y2, ok = getURLArgumentInt(r.URL.Path, 6)
	if !ok || rdsRequest.Y2 < 0 {
		log.Println("Y2 Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.Outxsize, ok = getURLArgumentInt(r.URL.Path, 7)
	if !ok || rdsRequest.Outxsize < 1 {
		log.Println("outxsize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}

	rdsRequest.Outzsize, ok = getURLArgumentInt(r.URL.Path, 8)
	if !ok || rdsRequest.Outzsize < 1 {
		log.Println("outzsize Missing or Bad. Required Field")
		w.WriteHeader(400)
		return
	}
	rdsRequest.getQueryParams(r)

	rdsRequest.computeRequestSizes()

	if rdsRequest.Xsize < 1 || rdsRequest.Ysize < 1 {
		log.Println("Bad Xsize or ysize. xsize: ", rdsRequest.Xsize, " ysize: ", rdsRequest.Ysize)
		w.WriteHeader(400)
		return
	}

	if cutType == "rdsxcut" {
		if rdsRequest.Ysize > 1 {
			log.Println("Currently only support cut of one y line. ysize:", rdsRequest.Ysize)
			w.WriteHeader(400)
			return
		}
	} else if cutType == "rdsycut" {
		if rdsRequest.Xsize > 1 {
			log.Println("Currently only support cut of one x line. xsize:", rdsRequest.Xsize)
			w.WriteHeader(400)
			return
		}
	}

	log.Println("RDS XY Cut Request params xstart, ystart, xsize, ysize, outxsize, outzsize:", cutType, rdsRequest.Xstart, rdsRequest.Ystart, rdsRequest.Xsize, rdsRequest.Ysize, rdsRequest.Outxsize, rdsRequest.Outzsize)

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
		rdsRequest.Reader, rdsRequest.FileName, ok = openDataSource(r.URL.Path, 9)
		if !ok {
			w.WriteHeader(400)
			return
		}

		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
			rdsRequest.processBlueFileHeader()
			if rdsRequest.SubsizeSet {
				rdsRequest.FileXSize = rdsRequest.Subsize

			} else {
				if rdsRequest.FileType == 1000 {
					log.Println("For type 1000 files, a subsize needs to be set")
					w.WriteHeader(400)
					return
				}
			}
			rdsRequest.computeYSize()
		} else {
			log.Println("Invalid File Type")
			w.WriteHeader(400)
			return
		}

		// Check Request against File Size
		if rdsRequest.Xsize > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X size greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.X1 > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X1 greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.X2 > rdsRequest.FileXSize {
			log.Println("Invalid Request. Requested X2 greater than file X size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.Y1 > rdsRequest.FileYSize {
			log.Println("Invalid Request. Requested Y1 greater than file Y size")
			w.WriteHeader(400)
			return
		}
		if rdsRequest.Y2 > rdsRequest.FileYSize {
			log.Println("Invalid Request. Requested Y2 greater than file Y size")
			w.WriteHeader(400)
			return
		}

		//If Zmin and Zmax were not explitily given then compute
		if !rdsRequest.Zset {
			rdsRequest.findZminMax()
		}

		data = processLineRequest(rdsRequest, cutType)

		if *useCache {
			go putItemInCache(cacheFileName, "outputFiles/", data)
		}

		// Store MetaData of request off in cache
		var fileMData fileMetaData
		fileMData.Outxsize = rdsRequest.Outxsize
		fileMData.Outysize = rdsRequest.Outysize
		fileMData.Outzsize = rdsRequest.Outzsize
		fileMData.Filexstart = rdsRequest.Filexstart
		fileMData.Filexdelta = rdsRequest.Filexdelta
		fileMData.Fileystart = rdsRequest.Fileystart
		fileMData.Fileydelta = rdsRequest.Fileydelta
		fileMData.Xstart = rdsRequest.Xstart
		fileMData.Ystart = rdsRequest.Ystart
		fileMData.Xsize = rdsRequest.Xsize
		fileMData.Ysize = rdsRequest.Ysize
		fileMData.Zmin = rdsRequest.Zmin
		fileMData.Zmax = rdsRequest.Zmax

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
	outzsizeStr := strconv.Itoa(fileMDataCache.Outzsize)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
	w.Header().Add("outxsize", outxsizeStr)
	w.Header().Add("outysize", outysizeStr)
	w.Header().Add("outzsize", outzsizeStr)
	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
	w.WriteHeader(http.StatusOK)

	w.Write(data)
}

//var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var configFile = flag.String("config", "./sdsConfig.json", "Location of Config File")
var useCache = flag.Bool("usecache", true, "Use SDS Cache. Can be disabled for certain cases like testing.")

type fileHeaderServer struct{}

func (s *fileHeaderServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("fileHeaderServer", r.URL.Path)
	reader, fileName, succeed := openDataSource(r.URL.Path, 3)
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
	reader, fileName, succeed := openDataSource(r.URL.Path, 3)
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

type fileSystemServer struct{}

func (s *fileSystemServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println("fileSystemServer", r.URL.Path)
	pathData := strings.Split(r.URL.Path, "/")

	if len(pathData) == 3 || (len(pathData) == 4 && pathData[3] == "") { //If no path is specified after /sds/ then list locations
		locationListServer := &locationListServer{}
		locationListServer.ServeHTTP(w, r)
		return
	}

	locationName := pathData[3]
	var urlPath string = ""
	for i := 4; i < len(pathData); i++ {
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

	if currentLocation.LocationType != "localFile" {
		log.Println("Error: Listing Files only support for localfile location Types")
		w.WriteHeader(400)
		return
	}

	if string(currentLocation.Path[len(currentLocation.Path)-1]) != "/" {
		currentLocation.Path += "/"
	}
	fullFilepath := fmt.Sprintf("%s%s", currentLocation.Path, urlPath)
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
	type fileObj struct {
		Filename string `json:"filename"`
		Type     string `json:"type"`
	}
	filelist := make([]fileObj, len(files))
	i := 0
	for _, file := range files {
		filelist[i].Filename = file.Name()
		if file.IsDir() {
			filelist[i].Type = "directory"
		} else {
			filelist[i].Type = "file"
		}
		i++
	}
	returnbytes, marshalError := json.Marshal(filelist)
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
	// Urls are /sds/<mode>/<mode Specific URL>
	rdsServer := &rdsServer{}
	rdsTileServer := &rdsTileServer{}
	headerServer := &fileHeaderServer{}
	fileSystemServer := &fileSystemServer{}
	rdsxyCutServer := &rdsxyCutServer{}
	ldsServer := &ldsServer{}

	if string(r.URL.Path[0]) != "/" {
		r.URL.Path = ("/") + string(r.URL.Path)
	}

	pathData := strings.Split(r.URL.Path, "/")
	mode := pathData[2]
	switch mode {
	case "fs":
		fileSystemServer.ServeHTTP(w, r)
	case "hdr":
		headerServer.ServeHTTP(w, r)
	case "rds":
		rdsServer.ServeHTTP(w, r)
	case "rdstile":
		rdsTileServer.ServeHTTP(w, r)
	case "rdsxcut":
		rdsxyCutServer.ServeHTTP(w, r)
	case "rdsycut":
		rdsxyCutServer.ServeHTTP(w, r)
	case "lds":
		ldsServer.ServeHTTP(w, r)
	default:
		log.Println("Unknown Mode", mode)
		w.WriteHeader(400)
		return
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

	zminzmaxFileMap = make(map[string]Zminzmax)
}

func main() {

	setupConfigLogCache()

	//Used to profile speed
	// if *cpuprofile != "" {
	// 	f, err := os.Create(*cpuprofile)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }

	// start := time.Now()

	// tileXsize := 500
	// tileYsize := 500
	// decX := 3
	// decY := 3
	// tileX := 0
	// tileY := 0
	// locationName := "TestData"
	// filename := "mydata_SI_8192_20000.tmp"
	// outfmt:= "RGBA"
	// sdsurl := "/sds/rdstile/" + strconv.Itoa(tileXsize)+"/"+strconv.Itoa(tileYsize)+"/"+strconv.Itoa(decX)+"/"+strconv.Itoa(decY)+"/"+strconv.Itoa(tileX)+"/"+strconv.Itoa(tileY)+"/"+locationName+"/"+filename+"?outfmt="+outfmt

	// req, _ := http.NewRequest("GET", sdsurl, nil)
	// rr := httptest.NewRecorder()
	// rdsServer := &routerServer{}
	// rdsServer.ServeHTTP(rr,req)

	// tileX =1
	// tileY = 0
	// sdsurl = "/sds/rdstile/" + strconv.Itoa(tileXsize)+"/"+strconv.Itoa(tileYsize)+"/"+strconv.Itoa(decX)+"/"+strconv.Itoa(decY)+"/"+strconv.Itoa(tileX)+"/"+strconv.Itoa(tileY)+"/"+locationName+"/"+filename+"?outfmt="+outfmt
	// req, _ = http.NewRequest("GET", sdsurl, nil)
	// rdsServer.ServeHTTP(rr,req)

	// tileX =0
	// tileY = 1
	// sdsurl = "/sds/rdstile/" + strconv.Itoa(tileXsize)+"/"+strconv.Itoa(tileYsize)+"/"+strconv.Itoa(decX)+"/"+strconv.Itoa(decY)+"/"+strconv.Itoa(tileX)+"/"+strconv.Itoa(tileY)+"/"+locationName+"/"+filename+"?outfmt="+outfmt
	// req, _ = http.NewRequest("GET", sdsurl, nil)
	// rdsServer.ServeHTTP(rr,req)

	// tileX =1
	// tileY = 1
	// sdsurl = "/sds/rdstile/" + strconv.Itoa(tileXsize)+"/"+strconv.Itoa(tileYsize)+"/"+strconv.Itoa(decX)+"/"+strconv.Itoa(decY)+"/"+strconv.Itoa(tileX)+"/"+strconv.Itoa(tileY)+"/"+locationName+"/"+filename+"?outfmt="+outfmt
	// req, _ = http.NewRequest("GET", sdsurl, nil)
	// rdsServer.ServeHTTP(rr,req)

	// log.Println("Computation Completed. Returned Code",rr.Code , "and bytes", len(rr.Body.Bytes()))
	// elapsed := time.Since(start)
	// log.Println("Total run time: ", elapsed)

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

	msg := ":%d"
	bindAddr := fmt.Sprintf(msg, configuration.Port)

	svr := &http.Server{
		Addr:           bindAddr,
		ReadTimeout:    240 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(svr.ListenAndServe())
}
