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

func createOutput(dataIn []float64,fileFormatString string) []byte {
    dataOut := new(bytes.Buffer)
    //var dataOut []byte
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

	return dataOut.Bytes()

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

func processline(file_name string, file_format string,file_data_offset int,fileXSize int,xstart int, ystart int,xsize int,outxsize int,transform string) []float64 {

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

	var real_data []float64
	if complex_flag {
		real_data=apply_cxmode(data_to_process,"mag")
	} else {
		real_data=data_to_process
	}
	out_data:=down_sample_line_inx(real_data,outxsize,transform)

	//fmt.Println("processline", len(out_data))
	return out_data

}

func processRequest(file_name string,file_format string,fileDataOffset int,fileXSize int,xstart int, ystart int, xsize int,ysize int, outxsize int, outysize int, transform string,outputFmt string) []byte {
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

		var xThinData []float64
		//fmt.Println("Going to Process Input Lines", startLine, endLine)
		for inputLine:=startLine;inputLine<endLine;inputLine++ {
			lineData := processline(file_name,file_format,fileDataOffset,fileXSize,xstart,inputLine,xsize,outxsize,transform)
			xThinData = append(xThinData,lineData...)
		}
		//fmt.Println("Thin X data is currently ", len(xThinData))
		// Thin in y direction the subsset of lines that have now been processed in x
		yThinData:=downSampleLineInY(xThinData,outxsize,transform)
		//fmt.Println("Thin Y data is currently ", len(yThinData))

		processedData = append(processedData,yThinData...)
		//fmt.Println("processedData is currently ", len(processedData))

	}
	//fmt.Println("Spot Check 0,49,50,99,:",processedData[0], processedData[49], processedData[50], processedData[99]) 

	//var outData=make([]byte,len(processedData))
	outData:=createOutput(processedData,outputFmt)
	return outData
}

func getURLArgumentInt(r *http.Request,keyname string) int {
	keys, ok := r.URL.Query()[keyname]
    
    if !ok || len(keys[0]) < 1 {
        log.Println("Url Param ",keyname, "  is missing")
        return 0
	}
	retval,err := strconv.Atoi(keys[0])
	if err != nil{
		log.Println("Url Param ",keyname, "  is invalid")
		return 0
	}
	return retval
} 

func getURLArgumentString(r *http.Request,keyname string) string {
	keys, ok := r.URL.Query()[keyname]
    
    if !ok || len(keys[0]) < 1 {
        log.Println("Url Param ",keyname, "  is missing")
        return ""
	}
	return keys[0]
} 

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
	//w.Header().Set("Content-Type", "application/json")

	file_name  :=getURLArgumentString(r,"filename")
	fileData := strings.Split(file_name, "_")

	// Need to get these parameters from file metadata
	file_format  := fileData[1]
	fileDataOffset  := 0
	fileXSize,err := strconv.Atoi(fileData[2])
	if err != nil{
		log.Println("Bad xfile size in filename")
		fileXSize = 0
	}


	x1 :=getURLArgumentInt(r,"x1")
	y1 :=getURLArgumentInt(r,"y1")
	x2 :=getURLArgumentInt(r,"x2")
	y2 :=getURLArgumentInt(r,"y2")
	ystart := int(math.Min(float64(y1),float64(y2)))
	xstart := int(math.Min(float64(x1),float64(x2)))
	xsize :=int(math.Abs(float64(x2)-float64(x1)))
	ysize :=int(math.Abs(float64(y2)-float64(y1)))
	outxsize  :=getURLArgumentInt(r,"outxsize")
	outysize  :=getURLArgumentInt(r,"outysize")
	transform :=getURLArgumentString(r,"transform")
	outputFmt :=getURLArgumentString(r,"outfmt")
	if outputFmt == "" {
		outputFmt = file_format
	}
	//zmin :=getURLArgumentInt(r,"zmin")
	//zmax :=getURLArgumentInt(r,"zmax")
	//colormap :=getURLArgumentString(r,"colormap")

	fmt.Println("params: ", xstart, ystart, xsize, ysize, outxsize, outysize)
	start := time.Now()
	data:=processRequest(file_name ,file_format,fileDataOffset,fileXSize,xstart,ystart,xsize,ysize,outxsize,outysize,transform,outputFmt) 
	elapsed := time.Since(start)
	fmt.Println("Length of Output Data " ,len(data), " processed in: ", elapsed) 

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