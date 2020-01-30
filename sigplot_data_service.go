package main

import (
    "log"
    "net/http"
    "encoding/binary"
    "bytes"
	"fmt"
	"math"
	"os"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/floats"
	"time"
	"strconv"
	"strings"
	"flag"
	"runtime/pprof"
	"unsafe"
)

var fileZMin float64
var fileZMax float64

func createOutput(dataIn []float64,fileFormatString string,zmin,zmax float64,colorMap string) []byte {
	dataOut := new(bytes.Buffer)
	var numColors int = 1000
	//var dataOut []byte
	if fileFormatString=="RGBA" {
		controlColors := getColorConrolPoints(colorMap)
		colorPalette:=makeColorPalette(controlColors,numColors)
		//fmt.Println("colorPalette 0 " ,colorPalette[0].red,colorPalette[0].green,colorPalette[0].blue)
		//fmt.Println("colorPalette 1 " ,colorPalette[1].red,colorPalette[1].green,colorPalette[1].blue)
		//fmt.Println("colorPalette 1 " ,colorPalette[2].red,colorPalette[2].green,colorPalette[2].blue)
		colorsPerSpan := (zmax-zmin) / float64(numColors)
		for i:=0;i<len(dataIn);i++ {
			colorIndex:= math.Round((dataIn[i]-zmin)/colorsPerSpan)
			colorIndex = math.Min(math.Max(colorIndex,0),float64(numColors-1)) //Ensure colorIndex is within the colorPalette
			//r := uint32(math.Round((float64(colorPalette[int(colorIndex)].red)/65535)*float64(255)))
			//g := uint32(math.Round((float64(colorPalette[int(colorIndex)].green)/65535)*float64(255)))
			//b := uint32(math.Round((float64(colorPalette[int(colorIndex)].blue)/65535)*float64(255)))
			a := 255
			dataOut.WriteByte(byte(colorPalette[int(colorIndex)].red))
			dataOut.WriteByte(byte(colorPalette[int(colorIndex)].green))
			dataOut.WriteByte(byte(colorPalette[int(colorIndex)].blue))
			dataOut.WriteByte(byte(a))
		}
	//fmt.Println("out_data RGBA" , len(dataOut.Bytes()))
	return dataOut.Bytes()
	} else {
		//fmt.Println("Processing for Type ",fileFormatString)
		switch string(fileFormatString[1]) {
		case "B":
			var numSlice=make([]int8,len(dataIn))
			for i:=0;i<len(numSlice);i++ {
				numSlice[i] = int8(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
		
			check(err)

		case "I":
			var numSlice=make([]int16,len(dataIn))
			for i:=0;i<len(numSlice);i++ {
				numSlice[i] = int16(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
		
			check(err)      

		case "L":
			fmt.Println("Processing for Type L")
			var numSlice=make([]int32,len(dataIn))
			for i:=0;i<len(numSlice);i++ {
				numSlice[i] = int32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
		
			check(err)    

		case "F":
			var numSlice=make([]float32,len(dataIn))
			for i:=0;i<len(numSlice);i++ {
				numSlice[i] = float32(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
		
			check(err)     

		case "D":
			var numSlice=make([]float64,len(dataIn))
			for i:=0;i<len(numSlice);i++ {
				numSlice[i] = float64(dataIn[i])
			}

			err := binary.Write(dataOut, binary.LittleEndian, &numSlice)
		
			check(err)  
		}
	//fmt.Println("out_data" , len(dataOut.Bytes()))
	return dataOut.Bytes()
	}

}

func processBlueFileHeader(fileName string) (string,int,int,float64,float64,float64,float64,float64,float64) {

	var bluefileheader BlueHeader
	file,err :=os.Open(fileName)
	check(err)
	binary.Read(file,binary.LittleEndian,&bluefileheader)
	//num_read,err:=file.Read(bluefileheader)

	fileFormat:=string(bluefileheader.Format[:])
	file_type :=int(bluefileheader.File_type)
	subsize:= int(bluefileheader.Subsize)
	xstart:=bluefileheader.Xstart
	xdelta:=bluefileheader.Xdelta
	ystart:=bluefileheader.Ystart
	ydelta:=bluefileheader.Ydelta
	data_start:=bluefileheader.Data_start
	data_size:=bluefileheader.Data_size

	fmt.Println("header data" , fileFormat,file_type,subsize)

	return fileFormat,file_type,subsize,xstart,xdelta,ystart,ydelta,data_start,data_size
}

func convert_file_data(bytesin []byte,file_formatstring string) []float64 {
	var bytes_per_atom int= 1
	//var atoms_in_file int= 1
	//var num_slice=make([]int8,atoms_in_file)
	var out_data []float64
	switch string(file_formatstring[1]) {
	case "B":
		bytes_per_atom = 1
		atoms_in_file :=len(bytesin)/bytes_per_atom
		out_data=make([]float64,atoms_in_file)
		for i:=0;i<atoms_in_file;i++ {
			num := *(*int8)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "I":
		bytes_per_atom = 2
		atoms_in_file :=len(bytesin)/bytes_per_atom	
		out_data=make([]float64,atoms_in_file)
		for i:=0;i<atoms_in_file;i++ {
			num := *(*int16)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "L":
		bytes_per_atom = 4
		atoms_in_file :=len(bytesin)/bytes_per_atom
		out_data=make([]float64,atoms_in_file)
		for i:=0;i<atoms_in_file;i++ {
			num := *(*int32)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "F":
		bytes_per_atom = 4
		atoms_in_file :=len(bytesin)/bytes_per_atom
		out_data=make([]float64,atoms_in_file)
		for i:=0;i<atoms_in_file;i++ {
			num := *(*float32)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = float64(num)
		}
	case "D":
		bytes_per_atom = 8
		atoms_in_file :=len(bytesin)/bytes_per_atom
		out_data=make([]float64,atoms_in_file)
		for i:=0;i<atoms_in_file;i++ {
			num := *(*float64)(unsafe.Pointer(&bytesin[i*bytes_per_atom]))
			out_data[i] = num
		}

	}
	//fmt.Println("out_data" , len(out_data))
	return out_data

}

func doTransform(dataIn []float64,transform string) float64 {
	switch transform{
	case "mean":
		return stat.Mean(dataIn[:],nil)
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

func get_file_type_info(file_format string) (int,bool){
	//fmt.Println("file_format", file_format)
	var complex_flag bool= false
	var bytes_per_atom int= 1
	if string(file_format[0]) =="C" {
		complex_flag=true
	} 
	//fmt.Println("string(file_format[1])", string(file_format[1]))
	switch string(file_format[1]) {
	case "B":
		bytes_per_atom = 1
	case "I":
		bytes_per_atom = 2
	case "L":
		bytes_per_atom = 4
	case "F":
		bytes_per_atom = 4
	case "D":
		bytes_per_atom = 8
	}
	return bytes_per_atom,complex_flag
}

func down_sample_line_inx(datain []float64, outxsize int,transform string) []float64 {
	//var inputysize int =len(datain)/framesize
	var xelementsperoutput float64  
	xelementsperoutput = float64(len(datain)/outxsize)

	var xelementsperoutput_ceil int = int(math.Ceil(xelementsperoutput))
	//fmt.Println("x thin" ,xelementsperoutput,xelementsperoutput_ceil,len(datain),outxsize)
	var thinxdata = make([]float64,outxsize)
	for x:=0; x<outxsize;x++ {
		var startelement int 
		var endelement int
		if x != (outxsize-1) {
			startelement  = x*int(math.Round(xelementsperoutput))
			endelement  = startelement + xelementsperoutput_ceil
			
		} else{
			endelement  =  len(datain)
			startelement  = endelement - xelementsperoutput_ceil
		}

		//fmt.Println("x thin" , x,outxsize,startelement,endelement)
		thinxdata[x] =doTransform(datain[startelement:endelement],transform)
		//fmt.Println("thinxdata[x]", thinxdata[x])

	}
	
	return thinxdata
}

func downSampleLineInY(datain []float64, outxsize int,transform string) []float64 {

	numLines := len(datain) / outxsize
	//fmt.Println("len(datain),outxsize" ,len(datain),outxsize) 
	processSlice:=make([]float64,numLines)
	outData:=make([]float64,outxsize)
	for x:=0;x<outxsize;x++ {
		for y:=0;y<numLines;y++ {
			//fmt.Println("y thin" ,y,outxsize,x) 
			processSlice[y] = datain[y*outxsize+x]
		}
		outData[x] = doTransform(processSlice[:],transform)
	}
	return outData
}
 	

func check(e error) {
    if e != nil {
        panic(e)
    }
}


func get_bytes_from_file(file_name string,first_byte int,numbytes int) []byte{

	out_data := make([]byte,numbytes)
	file,err :=os.Open(file_name)
	check(err)
	offset,err:=file.Seek(int64(first_byte),0)
	if offset !=int64(first_byte) {
		panic ("Failed to Seek")
	}
	check(err)
	num_read,err:=file.Read(out_data)
	check(err)
	if num_read !=numbytes {
		panic ("Failed to Read Requested Bytes")
	}
	//fmt.Println("Read Data Line" , len(out_data))
	return out_data

}

func apply_cxmode(datain []float64,cxmode string) []float64{

	//var lo_thresh float64=1.0e-20
	out_data := make([]float64,len(datain)/2)
	for i:=0;i<len(datain)-1;i+=2 {
		out_data[i] = math.Sqrt(datain[i]*datain[i]+datain[i+1]*datain[i+1])
	    //TODO Add modes besides Magnitude
	}
	return out_data

}

func processline(outChannel chan []float64, file_name string, file_format string,file_data_offset int,fileXSize int,xstart int, ystart int,xsize int,outxsize int,transform string) {

	bytes_per_atom,complex_flag := get_file_type_info(file_format)
	//fmt.Println("xsize,bytes_per_atom", xsize,bytes_per_atom)
	bytes_per_element := bytes_per_atom
	if complex_flag {
		bytes_per_element = bytes_per_element*2
	}
	bytes_length := xsize*bytes_per_element
	
	first_byte := file_data_offset +((ystart*fileXSize+xstart)*bytes_per_element)
	//fmt.Println("file Read info " ,ystart,xstart, first_byte ,bytes_length)
	filedata := get_bytes_from_file(file_name ,first_byte ,bytes_length)
	data_to_process :=convert_file_data(filedata,file_format)

	localMax := floats.Max(data_to_process[:])
	fileZMax = math.Max(fileZMax,localMax)

	localMin := floats.Min(data_to_process[:])
	fileZMin = math.Min(fileZMin,localMin)

	var real_data []float64
	if complex_flag {
		real_data=apply_cxmode(data_to_process,"mag")
	} else {
		real_data=data_to_process
	}
	out_data:=down_sample_line_inx(real_data,outxsize,transform)

	//fmt.Println("processline", len(out_data))
	outChannel<- out_data

}

func processRequest(file_name string,file_format string,fileDataOffset int,fileXSize int,xstart int, ystart int, xsize int,ysize int, outxsize int, outysize int, transform string,outputFmt string,zmin,zmax float64,zset bool,colorMap string) []byte {
	var processedData []float64

	var yLinesPerOutput float64 = float64(ysize)/float64(outysize)
	var yLinesPerOutputCeil int = int(math.Ceil(yLinesPerOutput))
	
	// Loop over the output Y Lines
	for outputLine:=0;outputLine<outysize;outputLine++ {
		//fmt.Println("Processing Output Line ", outputLine)
		// For Each Output Y line Read and process the required lines from the input file
		var startLine int
		var endLine int
		
		if outputLine != (outysize-1) {
			//fmt.Println("Not Last Line. yLinesPerOutput
			startLine = ystart+int(math.Round(float64(outputLine)*yLinesPerOutput))
			endLine = startLine + yLinesPerOutputCeil
		} else { //Last OutputLine, use the last line and work backwards the lineperoutput
			endLine = ystart+ysize
			startLine= endLine - yLinesPerOutputCeil
		}

		// Number of y lines that will be processed this time through the loop 
		numLines := endLine - startLine

		// Make channels to collect the data from processing all the lines in parallel. 
		var chans [100]chan []float64
		for i:=range chans {
			chans[i] = make(chan []float64)
		}
		var xThinData []float64
		//fmt.Println("Going to Process Input Lines", startLine, endLine)

		// Launch the processing of each line concurrently and put the data into a set of channels
		for inputLine:=startLine;inputLine<endLine;inputLine++ {
			go processline(chans[inputLine-startLine],file_name,file_format,fileDataOffset,fileXSize,xstart,inputLine,xsize,outxsize,transform)

		}

		// Pull Data out of concurrent channels in order into input arrary.
		for i:=0; i<numLines; i++ {
			data := <-chans[i]
			for j:=0; j<len(data); j++{
				xThinData = append(xThinData,data[j])
			}
		}
		
		// Thin in y direction the subsset of lines that have now been processed in x
		yThinData:=downSampleLineInY(xThinData,outxsize,transform)
		//fmt.Println("Thin Y data is currently ", len(yThinData))

		processedData = append(processedData,yThinData...)
		//fmt.Println("processedData is currently ", len(processedData))

	}
	//fmt.Println("Spot Check 0,49,50,99,:",processedData[0], processedData[49], processedData[50], processedData[99]) 

	if !zset {
		zmin = fileZMin
		zmax = fileZMax
	} 

	//var outData=make([]byte,len(processedData))
	outData:=createOutput(processedData,outputFmt,zmin,zmax,colorMap)
	return outData
}

func getURLArgumentInt(r *http.Request,keyname string) (int,bool) {
	keys, ok := r.URL.Query()[keyname]
    
    if !ok || len(keys[0]) < 1 {
    //    log.Println("Url Param ",keyname, "  is missing")
        return 0,false
	}
	retval,err := strconv.Atoi(keys[0])
	if err != nil{
		log.Println("Url Param ",keyname, "  is invalid")
		return 0,false
	}
	return retval,true
} 

func getURLArgumentString(r *http.Request,keyname string) (string,bool) {
	keys, ok := r.URL.Query()[keyname]
    
    if !ok || len(keys[0]) < 1 {
       // log.Println("Url Param ",keyname, "  is missing")
        return "",false
	}
	return keys[0],true
} 

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	file_name,ok  :=getURLArgumentString(r,"filename")
	if !ok {
		log.Println("Filename Missing. Required Field")
		w.WriteHeader(400)
		return 
	}

	var file_format string 
	var file_type int 
	var fileXSize int 
	var filexstart,filexdelta,fileystart,fileydelta,data_offset,file_data_size float64
	var fileDataOffset int
	if strings.Contains(file_name,".tmp") {
		log.Println("Processing File as Blue File")
		file_format,file_type,fileXSize,filexstart,filexdelta,fileystart,fileydelta,data_offset,file_data_size = processBlueFileHeader(file_name)
		fileDataOffset  = int(data_offset)
		if file_type !=2000 {
			log.Println("Only Supports type 2000 Bluefiles")
			w.WriteHeader(400)
			return 
		}

	} else if strings.Count(file_name,"_")==3 {
		log.Println("Processing File as binary file with metadata in filename with underscores")
		fileData := strings.Split(file_name, "_")
		// Need to get these parameters from file metadata
		file_format  = fileData[1]
		fileDataOffset  = 0
		var err error
		fileXSize,err = strconv.Atoi(fileData[2])
		if err != nil{
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

	// Get Rest of URL Parameters
	x1,ok :=getURLArgumentInt(r,"x1")
	if !ok {
		log.Println("X1 Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	y1,ok :=getURLArgumentInt(r,"y1")
	if !ok {
		log.Println("Y1 Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	x2,ok :=getURLArgumentInt(r,"x2")
	if !ok {
		log.Println("X2 Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	y2,ok :=getURLArgumentInt(r,"y2")
	if !ok {
		log.Println("Y2 Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	ystart := int(math.Min(float64(y1),float64(y2)))
	xstart := int(math.Min(float64(x1),float64(x2)))
	xsize :=int(math.Abs(float64(x2)-float64(x1)))
	ysize :=int(math.Abs(float64(y2)-float64(y1)))

	outxsize,ok  :=getURLArgumentInt(r,"outxsize")
	if !ok {
		log.Println("outxsize Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	
	outysize,ok :=getURLArgumentInt(r,"outysize")
	if !ok {
		log.Println("outysize Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	transform,ok :=getURLArgumentString(r,"transform")
	if !ok {
		log.Println("transform Missing. Required Field")
		w.WriteHeader(400)
		return 
	}
	outputFmt,ok :=getURLArgumentString(r,"outfmt")
	if !ok {
		log.Println("Outformat Not Specified. Setting Equal to Input Format")
		outputFmt = file_format
 
	}

	fmt.Println("Reported file_data_size", file_data_size)


	zminInt,zminSet := getURLArgumentInt(r,"zmin")
	var zmin float64
	if !zminSet {
		log.Println("Zmin Not Specified. Will estimate from file Selection")
		zmin=0
	} else {
		zmin=float64(zminInt)
	}
	
	zmaxInt,zmaxSet := getURLArgumentInt(r,"zmax")
	var zmax float64
	if !zmaxSet {
		log.Println("Zmax Not Specified. Will estimate from file Selection")
		zmax= 0
	} else {
		zmax=float64(zmaxInt)
	}

	zset := (zmaxSet && zminSet)
	colorMap,ok :=getURLArgumentString(r,"colormap")
	if !ok {
		log.Println("colorMap Not Specified.Defaulting to RampColormap")
		colorMap = "RampColormap"
	}


	fmt.Println("params xstart, ystart, xsize, ysize, outxsize, outysize:", xstart, ystart, xsize, ysize, outxsize, outysize)
	start := time.Now()
	data:=processRequest(file_name ,file_format,fileDataOffset,fileXSize,xstart,ystart,xsize,ysize,outxsize,outysize,transform,outputFmt,zmin,zmax,zset,colorMap) 
	elapsed := time.Since(start)
	fmt.Println("Length of Output Data " ,len(data), " processed in: ", elapsed) 

	if !zset {
		zmin = fileZMin
		zmax = fileZMax
	} 

	// Create a Return header with some metadata in it.
	outxsizeStr := strconv.Itoa(outxsize)
	outysizeStr := strconv.Itoa(outysize)
	w.Header().Add("Access-Control-Allow-Origin" ,"*")
	w.Header().Add("outxsize" ,outxsizeStr)
	w.Header().Add("outysize" ,outysizeStr)
	w.Header().Add("zmin" ,fmt.Sprintf("%.0f",zmin))
	w.Header().Add("zmax" ,fmt.Sprintf("%.0f",zmax))
	w.Header().Add("filexstart", fmt.Sprintf("%f",filexstart))
	w.Header().Add("filexdelta",fmt.Sprintf("%f",filexdelta))
	w.Header().Add("fileystart",fmt.Sprintf("%f",fileystart))
	w.Header().Add("fileydelta",fmt.Sprintf("%f",fileydelta))
	w.WriteHeader(http.StatusOK)

    w.Write(data)
}
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
    flag.Parse()
    if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
	}
	
	//Used to profile speed
	// start := time.Now()
	// data:=processRequest("mydata_SI_8192_20000" ,"SI",0,8192,0,0,8192,20000,100,250,"mean","SI") 
	// elapsed := time.Since(start)
	// fmt.Println("Length of Output Data " ,len(data), " processed in: ", elapsed) 

    s := &server{}
    http.Handle("/sds", s)
    log.Fatal(http.ListenAndServe(":5055", nil))
}