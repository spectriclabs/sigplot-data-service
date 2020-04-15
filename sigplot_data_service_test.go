package main

import (
    "net/http"
	"net/http/httptest"
	"encoding/binary"
	"bytes"
//	"net/url"
	"testing"
	"encoding/json"
	"strconv"
	"os"
	"math"
//	"log"
//	"fmt"
)

// Tests use the data file, "mydata_SB_600_600.tmp". This file is a 600 by 600 scaler byte file where it is 0 for the first 100  lines and 10 for the last 100 lines. 
// For lines between 101-500 it changes based on x value with 10 equal sized portions. Each section of 60 columns increases by 1 starting from 0 and going to 9.
// For example, lines 0-59, are 0, 60-119 are 1 ... 540-599 are 9. 


func FSHandler(t *testing.T,locationName string) []byte{
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
    // Create a request to pass to our handler. 
	
	sdsurl := "/sds/fs/" + locationName 
	//t.Log("url", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	headerServer := &routerServer{}
	headerServer.ServeHTTP(rr,req)

	if (rr.Code != http.StatusOK) {
		t.Errorf("handler returned wrong status code: got %v want %v",rr.Code, http.StatusOK)
	}
	return rr.Body.Bytes()
}
func TestDirectoryHandler(t *testing.T) {
	locationName := "TestDir/"
	_ = FSHandler(t,locationName)
}

func TestDirectoryHandler2(t *testing.T) {
	locationName := "TestDir"
	_ = FSHandler(t,locationName)
}

func TestDirectoryHandler3(t *testing.T) {
	locationName := "ServiceDir/tests/"
	_ = FSHandler(t,locationName)
}

func TestDirectoryHandler4(t *testing.T) {
	locationName := "ServiceDir/tests"
	_ = FSHandler(t,locationName)
}

func TestFileHandler(t *testing.T) {
	locationName := "TestDir/mydata_SB_60_60.tmp"
	returnData := FSHandler(t,locationName)
	if len(returnData) !=(60*60)+512 {
		t.Errorf("File URL Mode failed. Expecting %v bytes got %v", (60*60)+512, len(returnData))
	}
}

func TestLocationListHandler(t *testing.T) {
	locationName := ""
	returnData := FSHandler(t,locationName)
	var locationDetails []Location
	marshalError := json.Unmarshal(returnData, &locationDetails)
	if marshalError != nil {
		t.Errorf("Error with Rturn Data")
	}
	for i:=0;i<len(configuration.LocationDetails);i++ {
		if configuration.LocationDetails[i] != locationDetails[i] {
			t.Errorf("Location Details Don't match Configuration.")
		}
	}
}

func HDRHandler(t *testing.T,locationName string) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
    // Create a request to pass to our handler. 
	filename := "mydata_SB_60_60.tmp"
	sdsurl := "/sds/hdr/" + locationName +"/" +filename 
	//t.Log("url", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	headerServer := &routerServer{}
	headerServer.ServeHTTP(rr,req)

	if (rr.Code != http.StatusOK) {
		t.Errorf("handler returned wrong status code: got %v want %v",rr.Code, http.StatusOK)
	}

	var fileHeaderData BlueHeaderShortenedFields
	marshalError := json.Unmarshal(rr.Body.Bytes(), &fileHeaderData)
	if marshalError != nil {
		t.Errorf("Error unMarshaling JSON from hdr return: %v",marshalError)
	}
	//Check some of the header fields
	if fileHeaderData.Version != "BLUE" || fileHeaderData.Data_start != 512 ||
	   fileHeaderData.Data_size != 3600 ||  fileHeaderData.File_type != 2000 || 
	   fileHeaderData.Subsize != 60 || fileHeaderData.Xdelta != 1  {
		t.Errorf("Incorrect Header Data Returned")
	}

}

func TestHDRHandlerTestDir(t *testing.T) {

	HDRHandler(t,"TestDir")
}

func RDSTileHandler(t *testing.T,filename string,tileXsize,tileYsize,decX,decY,tileX,tileY int,outfmt string,expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rdstile/" + strconv.Itoa(tileXsize)+"/"+strconv.Itoa(tileYsize)+"/"+strconv.Itoa(decX)+"/"+strconv.Itoa(decY)+"/"+strconv.Itoa(tileX)+"/"+strconv.Itoa(tileY)+"/"+locationName+"/"+filename+"?outfmt="+outfmt

	req, err := http.NewRequest("GET", sdsurl, nil)

    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	rdsServer := &routerServer{}
	rdsServer.ServeHTTP(rr,req)

	if (rr.Code != http.StatusOK) {
		t.Errorf("handler returned wrong status code: got %v want %v",rr.Code, http.StatusOK)
	}
	for i:=0; i<len(expectedReturn); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v",i, rr.Body.Bytes()[i] , expectedReturn[i])
		}
	}
}

func TestFirstTile(t *testing.T) {
	tileXsize := 100
	tileYsize := 100
	tileX := 0
	tileY := 0
	expectedReturn := makeTileExpectedData(600,tileXsize,tileYsize,tileX,tileY)
	RDSTileHandler(t,"mydata_SB_600_600.tmp",tileXsize,tileYsize,1,1,tileX,tileY,"SB",expectedReturn)
}
func TestMiddleTile(t *testing.T) {
	tileXsize := 100
	tileYsize := 100
	tileX := 1
	tileY := 1
	expectedReturn := makeTileExpectedData(600,tileXsize,tileYsize,tileX,tileY)
	RDSTileHandler(t,"mydata_SB_600_600.tmp",tileXsize,tileYsize,1,1,tileX,tileY,"SB",expectedReturn)
}
func TestMiddleLargeTile(t *testing.T) {
	tileXsize := 200
	tileYsize := 300
	tileX := 1
	tileY := 1
	expectedReturn := makeTileExpectedData(600,tileXsize,tileYsize,tileX,tileY)
	RDSTileHandler(t,"mydata_SB_600_600.tmp",tileXsize,tileYsize,1,1,tileX,tileY,"SB",expectedReturn)
}

func TestPartialTile(t *testing.T) {
	tileXsize := 400
	tileYsize := 400
	tileX := 1
	tileY := 1
	expectedReturn := makeTileExpectedData(600,tileXsize,tileYsize,tileX,tileY)
	RDSTileHandler(t,"mydata_SB_600_600.tmp",tileXsize,tileYsize,1,1,tileX,tileY,"SB",expectedReturn)
}

func BaseicRDSHandlerColormap(t *testing.T,filename string,x1,y1,x2,y2,outxsize,outysize int, transform, cxmode, colormap , zmin,zmax string, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rds/" + strconv.Itoa(x1)+"/"+strconv.Itoa(y1)+"/"+strconv.Itoa(x2)+"/"+strconv.Itoa(y2)+"/"+strconv.Itoa(outxsize)+"/"+strconv.Itoa(outysize)+"/"+locationName+"/"+filename

	sdsurl = sdsurl+"?transform=" + transform + "&cxmode=" + cxmode +"&colormap=" + colormap + "&outfmt=RGBA" 
	
	if zmin != "skip" {
		sdsurl =sdsurl+"&zmin=" +  zmin
	}
	if zmax != "skip" {
		sdsurl =sdsurl+"&zmax=" +  zmax
	}

	t.Log("url:", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	rdsServer := &routerServer{}
	rdsServer.ServeHTTP(rr,req)

	if (rr.Code != http.StatusOK) {
		t.Errorf("handler returned wrong status code: got %v want %v",rr.Code, http.StatusOK)
	}

	for i:=0; i<len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v",i, rr.Body.Bytes()[i] , expectedReturn[i])
		}
	}

}


func BaseicRDSHandler(t *testing.T,filename string, x1,y1,x2,y2,outxsize,outysize int, transform, cxmode,outfmt string, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rds/" + strconv.Itoa(x1)+"/"+strconv.Itoa(y1)+"/"+strconv.Itoa(x2)+"/"+strconv.Itoa(y2)+"/"+strconv.Itoa(outxsize)+"/"+strconv.Itoa(outysize)+"/"+locationName+"/"+filename
	sdsurl = sdsurl+"?transform=" + transform + "&cxmode=" + cxmode +"&outfmt=" +outfmt

	t.Log("url:", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	rdsServer := &routerServer{}
	rdsServer.ServeHTTP(rr,req)

	if (rr.Code != http.StatusOK) {
		t.Errorf("handler returned wrong status code: got %v want %v",rr.Code, http.StatusOK)
	}

	for i:=0; i<len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			//t.Errorf("Values Did not match expected for %v byte: got %v expected %v",i, rr.Body.Bytes()[i] , expectedReturn[i])
		}
	}

}

func makeWholeExpectedData(size int) []byte {
	expectedReturn := make([]byte,size*size) 
	for line:=0; line<size; line++ {	
		if line<(size/6) {
			//Lines all 0 
			for column:=0; column<size; column++ {
				expectedReturn[line*size+column] = 0
			}

		} else if line>=(5*size/6) {
			//Lines all 10
			for column:=0; column<size; column++ {
				expectedReturn[line*size+column] = 10
			}

		} else {
			//Lines moving up from 0 to 9 
			for column:=0; column<size; column++ {
				expectedReturn[line*size+column] = uint8(column/(size/10))
			}
		}
	}
	return expectedReturn
}

func makeTileExpectedData(size, tileXsize,tileYsize,tileX,tileY int) []byte {
	var startLine,endLine,startColumn,endColumn int
	
	startLine = tileY*tileYsize
	endLine = int(math.Min(float64(startLine+tileYsize),float64(size)))
	startColumn = tileX*tileXsize
	endColumn = int(math.Min(float64(startColumn+tileXsize),float64(size)))
	expectedReturn := make([]byte,(endColumn-startColumn)*(endLine-startLine)) 
	for line:=startLine; line<endLine; line++ {	
		if line<(size/6) {
			//Lines all 0 
			for column:=startColumn; column<endColumn; column++ {
				expectedReturn[(line-startLine)*(endColumn-startColumn)+(column-startColumn)] = 0
			}

		} else if line>=(5*size/6) {
			//Lines all 10
			for column:=startColumn; column<endColumn; column++ {
				expectedReturn[(line-startLine)*(endColumn-startColumn)+(column-startColumn)] = 10
			}

		} else {
			//Lines moving up from 0 to 9 
			for column:=startColumn; column<endColumn; column++ {
				expectedReturn[(line-startLine)*(endColumn-startColumn)+(column-startColumn)] = uint8(column/(size/10))
			}
		}
	}
	return expectedReturn
}

func TestFirstPoint(t *testing.T) {
	expectedReturn := make([]byte,1)
	expectedReturn[0] = 0 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,1,1,1,1,"first","Re", "SB",expectedReturn)
}

func TestLastPoint(t *testing.T) {
	expectedReturn := make([]byte,1)
	expectedReturn[0] = 10 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",59,59,60,60,1,1,"first","Re","SB",expectedReturn)
}

func TestAverageFirst10Point(t *testing.T) {
	expectedReturn := make([]byte,1)
	expectedReturn[0] = 0 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,10,10,1,1,"mean","Re","SB",expectedReturn)
}

func TestAverageMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 1 //Input Data is values of 0,1,2 in equal amounts, averaged together be 1 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"mean","Re","SB",expectedReturn)
}
func TestFirstMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 0 //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","SB",expectedReturn)
}
func TestMaxMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 2 //Input Data is values of 0,1,2 in equal amounts, max value will return 2. 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"max","Re","SB",expectedReturn)
}

func TestMiddlePoint20Log(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 12 //Input Data 2. 20*log10(2*2) = 12.04 which will return 12 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",17,20,18,21,1,1,"first","L2","SB",expectedReturn)
}

func TestMiddlePoint10Log(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 6 //Input Data 2. 20*log10(2*2) = 12.04 which will return 12 
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",17,20,18,21,1,1,"first","Lo","SB",expectedReturn)
}

func TestMiddlePoints20LogColormap(t *testing.T) {
	expectedReturn := make([]byte,4) 
	expectedReturn[0] = 0 //Input Data 2. 20*log10(2*2) = 12.04. By setting the zmin to 12.04, the colormap should return the first value which is 0,0,38. 
	expectedReturn[1] = 0 
	expectedReturn[2] = 38 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",17,20,18,21,1,1,"first","L2","Ramp Colormap", "12.04","50", expectedReturn)
}

func TestMiddlePoints20LogColormapMax(t *testing.T) {
	expectedReturn := make([]byte,4) 
	expectedReturn[0] = 255 //Input Data 2. 20*log10(2*2) = 12.04. By setting the zmax to 12.04, the colormap should return the last value which is 255,0,0. 
	expectedReturn[1] = 0 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",17,20,18,21,1,1,"first","L2","Ramp Colormap", "0","12.04", expectedReturn)
}

func TestMiddlePoints20LogColormapMiddle(t *testing.T) {
	expectedReturn := make([]byte,4) 
	expectedReturn[0] = 0 //Input Data 2. 20*log10(2*2) = 12.04. By setting the zmax to 0, 24.0824, the colormap should return the middle value which is 0,204,0. 
	expectedReturn[1] = 204 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",17,20,18,21,1,1,"first","L2","Ramp Colormap", "0","24.0824", expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmax(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 0 // Should return lowest value in colormap "Ramp Colormap"
	expectedReturn[1] = 0 
	expectedReturn[2] = 38 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","Ramp Colormap", "skip","skip", expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmaxGreyscale(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 0 // Should return lowest value in colormap "Ramp Colormap"
	expectedReturn[1] = 0 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","Greyscale", "skip","skip", expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxColorWheel(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 255 // Should return lowest value in colormap "Color Wheel"
	expectedReturn[1] = 255 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","Color Wheel", "skip","skip", expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxSpectrum(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 0 // Should return lowest value in colormap "Spectrum"
	expectedReturn[1] = 191 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","Spectrum", "skip","skip", expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmaxcalewhite(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 255 // Should return lowest value in colormap "calewhite"
	expectedReturn[1] = 255 
	expectedReturn[2] = 255 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","calewhite", "skip","skip", expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmaxHotDesat(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 71 // Should return lowest value in colormap "HotDesat"
	expectedReturn[1] = 71 
	expectedReturn[2] = 219 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","HotDesat", "skip","skip", expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxSunset(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	expectedReturn[0] = 26 // Should return lowest value in colormap "Sunset"
	expectedReturn[1] = 0 
	expectedReturn[2] = 59 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"first","Re","Sunset", "skip","skip", expectedReturn)
}

func TestMeanMiddlePointsColormapNoZinZmax(t *testing.T) {

	expectedReturn := make([]byte,4) //Input Data is values of 0,1,2 in equal amounts, mean value will return 1. 
	expectedReturn[0] = 0 // Should 10% point from "Ramp Colormap"
	expectedReturn[1] = 0 
	expectedReturn[2] = 128 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,"mydata_SB_60_60.tmp",0,20,18,21,1,1,"mean","Re","Ramp Colormap", "skip","skip", expectedReturn)
}

func TestFullReducedMax(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,60,60,30,30,"max","Re","SB",expectedReturn)
}
func TestFullReducedMin(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,60,60,30,30,"min","Re","SB",expectedReturn)
}

func TestFullReducedFirst(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,60,60,30,30,"first","Re","SB",expectedReturn)
}
func TestFullReducedMean(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,60,60,30,30,"mean","Re","SB",expectedReturn)
}

func TestFullReducedMaxAbs(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,60,60,30,30,"maxabs","Re","SB",expectedReturn)
}

func TestFullSameSizeMean(t *testing.T) {
	expectedReturn := makeWholeExpectedData(60)
	BaseicRDSHandler(t,"mydata_SB_60_60.tmp",0,0,60,60,60,60,"mean","Re","SB",expectedReturn)
}

func TestFullISameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]int16,len(expectedResults))
	for i:=0; i< len(IntData); i++ {
		IntData[i] = int16(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t,"mydata_SI_60_60.tmp",0,0,60,60,60,60,"mean","Re","SI",byteData.Bytes())
}

func TestFullLSameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]int32,len(expectedResults))
	for i:=0; i< len(IntData); i++ {
		IntData[i] = int32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t,"mydata_SL_60_60.tmp",0,0,60,60,60,60,"mean","Re","SL",byteData.Bytes())
}

func TestFullFSameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32,len(expectedResults))
	for i:=0; i< len(IntData); i++ {
		IntData[i] = float32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t,"mydata_SF_60_60.tmp",0,0,60,60,60,60,"mean","Re","SF",byteData.Bytes())
}

func TestFullDSameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float64,len(expectedResults))
	for i:=0; i< len(IntData); i++ {
		IntData[i] = float64(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t,"mydata_SD_60_60.tmp",0,0,60,60,60,60,"mean","Re","SD",byteData.Bytes())
}

func TestFullCFSameSizeMeanReal(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32,len(expectedResults))
	for i:=0; i< len(IntData); i++ {
		IntData[i] = float32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t,"mydata_CF_60_60.tmp",0,0,60,60,60,60,"mean","Im","CF",byteData.Bytes())
}

func TestFirstPointSP(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 0 //SP file has the bits 0,0,1,1, as the first four
	BaseicRDSHandler(t,"mydata_SP_80_80.tmp",0,0,1,1,1,1,"first","Re", "SB",expectedReturn)
}

func TestFourPointsSP(t *testing.T) {
	expectedReturn := make([]byte,4)
	expectedReturn[0] = 0 //SP file has the bits 0,0,1,1, as the first four
	expectedReturn[1] = 0
	expectedReturn[2] = 1
	expectedReturn[3] = 1
	BaseicRDSHandler(t,"mydata_SP_80_80.tmp",1,0,4,1,4,1,"first","Re", "SB",expectedReturn)
}
