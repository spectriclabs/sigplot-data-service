package handlers

import (
	"io"
	"log"
	"math"

	"gonum.org/v1/gonum/floats"

	"sigplot-data-service/internal/bluefile"
	"sigplot-data-service/internal/datasource"
	"sigplot-data-service/internal/image"
	"sigplot-data-service/internal/numerical"
)

var fileZMin float64 = 99999999
var fileZMax float64 = -99999999

type ProcessRequest struct {
	FileFormat     string
	FileDataOffset int
	FileXSize      int
	Xstart         int
	Ystart         int
	Xsize          int
	Ysize          int
	Outxsize       int
	Outysize       int
	Transform      string
	Cxmode         string
	OutputFmt      string
	Zmin           float64
	Zmax           float64
	Zset           bool
	CxmodeSet      bool
	ColorMap       string
}

func ProcessLine(
	outData []float64,
	outLineNum int,
	done chan bool,
	reader io.ReadSeeker,
	req ProcessRequest,
) {
	bytesPerAtom, complexFlag := bluefile.GetFileTypeInfo(req.FileFormat)

	//log.Println("xsize,bytes_per_atom", xsize,bytes_per_atom)
	bytesPerElement := bytesPerAtom
	if complexFlag {
		bytesPerElement = bytesPerElement * 2
	}

	firstDataByte := float64(req.Ystart*req.FileXSize+req.Xstart) * bytesPerElement
	firstByteInt := int(math.Floor(firstDataByte))

	bytesLength := float64(req.Xsize)*bytesPerElement + (firstDataByte - float64(firstByteInt))
	bytesLengthInt := int(math.Ceil(bytesLength))

	//log.Println("file Read info " ,ystart,xstart, firstByte ,bytes_length)
	filedata, _ := datasource.GetBytesFromReader(reader, req.FileDataOffset+firstByteInt, bytesLengthInt)

	//filedata := get_bytes_from_file(fileName ,first_byte ,bytes_length)
	dataToProcess := bluefile.ConvertFileData(filedata, req.FileFormat)

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
	var localMax float64
	var localMin float64
	if complexFlag {
		realData, localMin, localMax = numerical.ApplyCXmode(dataToProcess, req.Cxmode, true)
	} else {
		if req.CxmodeSet {
			realData, localMin, localMax = numerical.ApplyCXmode(dataToProcess, req.Cxmode, false)
		} else {
			realData = dataToProcess

			// Finding the max and min of data we processed to get
			// a zmax and zmin if they are not set.
			// Profiling suggests this is computationally intense.
			localMin = floats.Min(realData[:])
			localMax = floats.Max(realData[:])
		}

	}

	if !req.Zset {
		fileZMax = math.Max(fileZMax, localMax)
		fileZMin = math.Min(fileZMin, localMin)
	}

	//log.Println("processline", (outxsize),len(real_data),xsize)
	//out_data :=make([]float64,outxsize)
	numerical.DownSampleLineInX(
		realData,
		req.Outxsize,
		req.Transform,
		outData,
		outLineNum,
	)

	//copy(outData[outLineNum*outxsize:],out_data)
	//log.Println("processline Done", len(out_data))
	done <- true
}

func HandleProcessRequest(reader io.ReadSeeker, req ProcessRequest) []byte {
	var processedData []float64

	var yLinesPerOutput float64 = float64(req.Ysize) / float64(req.Outysize)
	var yLinesPerOutputCeil int = int(math.Ceil(yLinesPerOutput))

	fileZMin = 99999999
	fileZMax = -99999999

	// Loop over the output Y Lines
	for outputLine := 0; outputLine < req.Outysize; outputLine++ {
		// For Each Output Y line Read and process the required lines from the input file
		var startLine int
		var endLine int

		if outputLine != (req.Outysize - 1) {
			//log.Println("Not Last Line. yLinesPerOutput
			startLine = req.Ystart + int(math.Round(float64(outputLine)*yLinesPerOutput))
			endLine = startLine + yLinesPerOutputCeil
		} else { //Last OutputLine, use the last line and work backwards the lineperoutput
			endLine = req.Ystart + req.Ysize
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
		xThinData := make([]float64, numLines*req.Outxsize)
		//log.Println("Going to Process Input Lines", startLine, endLine)

		done := make(chan bool, 1)
		// Launch the processing of each line concurrently and put the data into a set of channels
		for inputLine := startLine; inputLine < endLine; inputLine++ {
			req.Ystart = inputLine
			go ProcessLine(
				xThinData,
				inputLine-startLine,
				done,
				reader,
				req,
			)

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
		yThinData := numerical.DownSampleLineInY(xThinData, req.Outxsize, req.Transform)
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
	log.Println("Process Request Zset ", req.Zset)
	if !req.Zset {
		req.Zmin = fileZMin
		req.Zmax = fileZMax
		log.Println("Getting Zmin, ZMax For File", req.Zmin, req.Zmax)
	}
	log.Println("Creating Output with Zmin, ZMax", req.Zmin, req.Zmax)
	outData := image.CreateOutput(
		processedData,
		req.OutputFmt,
		req.Zmin,
		req.Zmax,
		req.ColorMap,
	)
	return outData
}
