package main

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/http/httptest"

	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"testing"
)

// Tests use the data file, "mydata_SB_600_600.tmp". This file is a 600 by 600 scaler byte file where it is 0 for the first 100  lines and 10 for the last 100 lines.
// For lines between 101-500 it changes based on x value with 10 equal sized portions. Each section of 60 columns increases by 1 starting from 0 and going to 9.
// For example, lines 0-59, are 0, 60-119 are 1 ... 540-599 are 9.
// Test files of "mydata_XX_60_60.tmp" are the same format but on 60x 60 in size.

func FSHandler(t *testing.T, locationName string, expectedReturnCode int) []byte {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	// Create a request to pass to our handler.

	sdsurl := "/sds/fs/" + locationName
	//t.Log("url", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	if err != nil {
		t.Fatal(err)
	}

	SetupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	headerServer := &routerServer{}
	headerServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}
	return rr.Body.Bytes()
}
func TestBaddModeHandler(t *testing.T) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	// Create a request to pass to our handler.

	sdsurl := "/sds/bad/"
	//t.Log("url", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	if err != nil {
		t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	headerServer := &routerServer{}
	headerServer.ServeHTTP(rr, req)

	if rr.Code != 400 {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, 400)
	}
}

type fileObj struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
}

func checkfiles(t *testing.T, returnBytes []byte) {
	var fileDetails []fileObj
	marshalError := json.Unmarshal(returnBytes, &fileDetails)
	if marshalError != nil {
		t.Errorf("File List Returned did not unmarshal to the correct type")
	}
	files, err := ioutil.ReadDir("./tests")
	if err != nil {
		t.Errorf("Error Reading Testing directory")
	}
	var found bool
	for _, osfile := range files {
		found = false
		for _, jsonfile := range fileDetails {
			//log.Println("Looking for:",osfile.Name(), "found:", jsonfile.Filename )
			if osfile.Name() == jsonfile.Filename {
				found = true
				if (osfile.IsDir() || jsonfile.Type == "directory") && !(osfile.IsDir() && jsonfile.Type == "directory") {
					t.Errorf("File Type not correct. For file %v", osfile.Name())
				}
			}
		}
		if !found {
			t.Errorf("File %v not found in return data", osfile.Name())
		}

	}
}

func TestDirectoryHandler(t *testing.T) {
	locationName := "TestDir/"
	returnData := FSHandler(t, locationName, 200)
	checkfiles(t, returnData)

}

func TestDirectoryHandler2(t *testing.T) {
	locationName := "TestDir"
	returnData := FSHandler(t, locationName, 200)
	checkfiles(t, returnData)
}

func TestDirectoryHandler3(t *testing.T) {
	locationName := "ServiceDir/tests/"
	returnData := FSHandler(t, locationName, 200)
	checkfiles(t, returnData)
}

func TestDirectoryHandler4(t *testing.T) {
	locationName := "ServiceDir/tests"
	returnData := FSHandler(t, locationName, 200)
	checkfiles(t, returnData)
}

func TestDirectoryHandlerBad(t *testing.T) {
	locationName := "ServiceDir/bad"
	_ = FSHandler(t, locationName, 400)
}

func TestFileHandler(t *testing.T) {
	locationName := "TestDir/mydata_SB_60_60.tmp"
	returnData := FSHandler(t, locationName, 200)
	if len(returnData) != (60*60)+512 {
		t.Errorf("File URL Mode failed. Expecting %v bytes got %v", (60*60)+512, len(returnData))
	}
}

func TestLocationListHandler(t *testing.T) {
	locationName := ""
	returnData := FSHandler(t, locationName, 200)
	var locationDetails []Location
	marshalError := json.Unmarshal(returnData, &locationDetails)
	if marshalError != nil {
		t.Errorf("Error with Rturn Data")
	}
	for i := 0; i < len(configuration.LocationDetails); i++ {
		if configuration.LocationDetails[i] != locationDetails[i] {
			t.Errorf("Location Details Don't match Configuration.")
		}
	}
}

func HDRHandler(t *testing.T, locationName, filename string, expectedReturnCode int) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	// Create a request to pass to our handler.
	sdsurl := "/sds/hdr/" + locationName + "/" + filename
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
	headerServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}
	if rr.Code == 400 {
		return
	}

	var fileHeaderData BlueHeaderShortenedFields
	marshalError := json.Unmarshal(rr.Body.Bytes(), &fileHeaderData)
	if marshalError != nil {
		t.Errorf("Error unMarshaling JSON from hdr return: %v", marshalError)
	}
	//Check some of the header fields
	if fileHeaderData.Version != "BLUE" || fileHeaderData.Data_start != 512 ||
		fileHeaderData.Data_size != 3600 || fileHeaderData.File_type != 2000 ||
		fileHeaderData.Subsize != 60 || fileHeaderData.Xdelta != 1 {
		t.Errorf("Incorrect Header Data Returned")
	}

}

func TestHDRHandlerTestDir(t *testing.T) {
	filename := "mydata_SB_60_60.tmp"
	HDRHandler(t, "TestDir", filename, 200)
}
func TestHDRHandlerBad(t *testing.T) {
	filename := "mydata_SB_60_60.tmp"
	HDRHandler(t, "bad", filename, 400)
}
func TestHDRHandlerBadFileType(t *testing.T) {
	filename := "sdsTestConfig.json"
	HDRHandler(t, "TestDir", filename, 400)
}

func RDSTileHandler(t *testing.T, filename string, tileXsize, tileYsize, decX, decY, tileX, tileY int, outfmt string, expectedReturnCode int, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rdstile/" + strconv.Itoa(tileXsize) + "/" + strconv.Itoa(tileYsize) + "/" + strconv.Itoa(decX) + "/" + strconv.Itoa(decY) + "/" + strconv.Itoa(tileX) + "/" + strconv.Itoa(tileY) + "/" + locationName + "/" + filename + "?outfmt=" + outfmt

	req, err := http.NewRequest("GET", sdsurl, nil)

	if err != nil {
		t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	rdsServer := &routerServer{}
	rdsServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}
	for i := 0; i < len(expectedReturn); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v", i, rr.Body.Bytes()[i], expectedReturn[i])
		}
	}
}

func TestInvalidTileRequest(t *testing.T) {

	expectedReturn := make([]byte, 0)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 10, 100, 1, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 10, 1, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 1000, 100, 1, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 1000, 1, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 350, 100, 1, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 0, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 1, 0, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 1, 11, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 11, 1, 0, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 1, 1, 8, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 1, 1, 0, 8, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 1, 1, -1, 0, "SB", 400, expectedReturn)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", 100, 100, 1, 1, 0, -1, "SB", 400, expectedReturn)
}

func TestFirstTile(t *testing.T) {
	tileXsize := 100
	tileYsize := 100
	tileX := 0
	tileY := 0
	expectedReturn := makeTileExpectedData(600, tileXsize, tileYsize, tileX, tileY)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", tileXsize, tileYsize, 1, 1, tileX, tileY, "SB", 200, expectedReturn)
}
func TestMiddleTile(t *testing.T) {
	tileXsize := 100
	tileYsize := 100
	tileX := 1
	tileY := 1
	expectedReturn := makeTileExpectedData(600, tileXsize, tileYsize, tileX, tileY)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", tileXsize, tileYsize, 1, 1, tileX, tileY, "SB", 200, expectedReturn)
}
func TestMiddleLargeTile(t *testing.T) {
	tileXsize := 200
	tileYsize := 300
	tileX := 1
	tileY := 1
	expectedReturn := makeTileExpectedData(600, tileXsize, tileYsize, tileX, tileY)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", tileXsize, tileYsize, 1, 1, tileX, tileY, "SB", 200, expectedReturn)
}

func TestPartialTile(t *testing.T) {
	tileXsize := 400
	tileYsize := 400
	tileX := 1
	tileY := 1
	expectedReturn := makeTileExpectedData(600, tileXsize, tileYsize, tileX, tileY)
	RDSTileHandler(t, "mydata_SB_600_600.tmp", tileXsize, tileYsize, 1, 1, tileX, tileY, "SB", 200, expectedReturn)
}

func BaseicRDSHandlerColormap(t *testing.T, filename string, x1, y1, x2, y2, outxsize, outysize int, transform, cxmode, colormap, zmin, zmax string, expectedReturnCode int, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rds/" + strconv.Itoa(x1) + "/" + strconv.Itoa(y1) + "/" + strconv.Itoa(x2) + "/" + strconv.Itoa(y2) + "/" + strconv.Itoa(outxsize) + "/" + strconv.Itoa(outysize) + "/" + locationName + "/" + filename

	sdsurl = sdsurl + "?transform=" + transform + "&cxmode=" + cxmode + "&colormap=" + colormap + "&outfmt=RGBA"

	if zmin != "skip" {
		sdsurl = sdsurl + "&zmin=" + zmin
	}
	if zmax != "skip" {
		sdsurl = sdsurl + "&zmax=" + zmax
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
	rdsServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}

	for i := 0; i < len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v", i, rr.Body.Bytes()[i], expectedReturn[i])
		}
	}

}

func BaseicRDSHandler(t *testing.T, filename string, x1, y1, x2, y2, outxsize, outysize int, transform, cxmode, outfmt string, expectedReturnCode int, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rds/" + strconv.Itoa(x1) + "/" + strconv.Itoa(y1) + "/" + strconv.Itoa(x2) + "/" + strconv.Itoa(y2) + "/" + strconv.Itoa(outxsize) + "/" + strconv.Itoa(outysize) + "/" + locationName + "/" + filename
	sdsurl = sdsurl + "?transform=" + transform + "&cxmode=" + cxmode + "&outfmt=" + outfmt

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
	rdsServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}

	if len(rr.Body.Bytes()) != len(expectedReturn) {
		t.Errorf("Did not get correct length return. Got %v epected %v ", len(rr.Body.Bytes()), len(expectedReturn))
	}

	var testFail bool = false
	for i := 0; i < len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			testFail = true
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v", i, rr.Body.Bytes()[i], expectedReturn[i])
		}
	}

	if testFail {
		t.Errorf("Values did not match expected data")
	}

}

func BaseicRDSHandlerSubsize(t *testing.T, filename string, x1, y1, x2, y2, outxsize, outysize, subsize int, transform, cxmode, outfmt string, expectedReturnCode int, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/rds/" + strconv.Itoa(x1) + "/" + strconv.Itoa(y1) + "/" + strconv.Itoa(x2) + "/" + strconv.Itoa(y2) + "/" + strconv.Itoa(outxsize) + "/" + strconv.Itoa(outysize) + "/" + locationName + "/" + filename
	sdsurl = sdsurl + "?transform=" + transform + "&cxmode=" + cxmode + "&outfmt=" + outfmt + "&subsize=" + strconv.Itoa(subsize)

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
	rdsServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}

	if len(rr.Body.Bytes()) != len(expectedReturn) {
		t.Errorf("Did not get correct length return. Got %v epected %v ", len(rr.Body.Bytes()), len(expectedReturn))
	}

	var testFail bool = false
	for i := 0; i < len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			testFail = true
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v", i, rr.Body.Bytes()[i], expectedReturn[i])
		}
	}

	if testFail {
		t.Errorf("Values did not match expected data")
	}

}
func BaseicRDSxCutHandler(t *testing.T, filename, mode string, x1, y1, x2, y2, outxsize, outzsize int, cxmode string, expectedReturnCode int, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/" + mode + "/" + strconv.Itoa(x1) + "/" + strconv.Itoa(y1) + "/" + strconv.Itoa(x2) + "/" + strconv.Itoa(y2) + "/" + strconv.Itoa(outxsize) + "/" + strconv.Itoa(outzsize) + "/" + locationName + "/" + filename
	sdsurl = sdsurl + "?cxmode=" + cxmode

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
	rdsServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}

	if len(rr.Body.Bytes()) != len(expectedReturn) {
		t.Errorf("Did not get correct length return. Got %v epected %v ", len(rr.Body.Bytes()), len(expectedReturn))
	}

	var testFail bool = false
	for i := 0; i < len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			testFail = true
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v", i, rr.Body.Bytes()[i], expectedReturn[i])
		}
	}

	if testFail {
		t.Errorf("Values did not match expected data")
	}

}

func BaseicLDSHandler(t *testing.T, filename string, x1, x2, outxsize, outzsize int, cxmode string, expectedReturnCode int, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false", "-config=./tests/sdsTestConfig.json"}
	locationName := "TestDir"
	sdsurl := "/sds/lds/" + strconv.Itoa(x1) + "/" + strconv.Itoa(x2) + "/" + strconv.Itoa(outxsize) + "/" + strconv.Itoa(outzsize) + "/" + locationName + "/" + filename
	sdsurl = sdsurl + "?cxmode=" + cxmode

	t.Log("url:", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
	if err != nil {
		t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	rdsServer := &routerServer{}
	rdsServer.ServeHTTP(rr, req)

	if rr.Code != expectedReturnCode {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, expectedReturnCode)
	}

	if len(rr.Body.Bytes()) != len(expectedReturn) {
		t.Errorf("Did not get correct length return. Got %v epected %v ", len(rr.Body.Bytes()), len(expectedReturn))
	}

	var testFail bool = false
	for i := 0; i < len(rr.Body.Bytes()); i++ {
		if rr.Body.Bytes()[i] != expectedReturn[i] {
			testFail = true
			t.Errorf("Values Did not match expected for %v byte: got %v expected %v", i, rr.Body.Bytes()[i], expectedReturn[i])
		}
	}

	if testFail {
		t.Errorf("Values did not match expected data")
	}

}

func makeWholeExpectedData(size int) []byte {
	expectedReturn := make([]byte, size*size)
	for line := 0; line < size; line++ {
		if line < (size / 6) {
			//Lines all 0
			for column := 0; column < size; column++ {
				expectedReturn[line*size+column] = 0
			}

		} else if line >= (5 * size / 6) {
			//Lines all 10
			for column := 0; column < size; column++ {
				expectedReturn[line*size+column] = 10
			}

		} else {
			//Lines moving up from 0 to 9
			for column := 0; column < size; column++ {
				expectedReturn[line*size+column] = uint8(column / (size / 10))
			}
		}
	}
	return expectedReturn
}

func makeTileExpectedData(size, tileXsize, tileYsize, tileX, tileY int) []byte {
	var startLine, endLine, startColumn, endColumn int

	startLine = tileY * tileYsize
	endLine = int(math.Min(float64(startLine+tileYsize), float64(size)))
	startColumn = tileX * tileXsize
	endColumn = int(math.Min(float64(startColumn+tileXsize), float64(size)))
	expectedReturn := make([]byte, (endColumn-startColumn)*(endLine-startLine))
	for line := startLine; line < endLine; line++ {
		if line < (size / 6) {
			//Lines all 0
			for column := startColumn; column < endColumn; column++ {
				expectedReturn[(line-startLine)*(endColumn-startColumn)+(column-startColumn)] = 0
			}

		} else if line >= (5 * size / 6) {
			//Lines all 10
			for column := startColumn; column < endColumn; column++ {
				expectedReturn[(line-startLine)*(endColumn-startColumn)+(column-startColumn)] = 10
			}

		} else {
			//Lines moving up from 0 to 9
			for column := startColumn; column < endColumn; column++ {
				expectedReturn[(line-startLine)*(endColumn-startColumn)+(column-startColumn)] = uint8(column / (size / 10))
			}
		}
	}
	return expectedReturn
}

func make1DExpectedData(mode string, size, line, outxsize, outzsize, zmin, zmax int) []byte {

	var wholefile []byte
	if mode == "xcut" {
		wholefile = makeWholeExpectedData(size)
	} else if mode == "line" {
		wholefile = makeLineExpectedData()
		line = 0
		size = 500
	} else if mode == "ycut" {
		wholefile = makeYcutExpectedData(size, line)
		line = 0
	}
	xslice := make([]int16, 0, size*2)
	zslice := make([]int16, 0, size*2)
	xratio := float64(size) / float64(outxsize-1)
	zratio := float64(zmax-zmin) / float64(outzsize-1)
	for x := line * size; x < (line+1)*size; x++ {

		xpixel := int16(math.Round((float64(x) - float64(line*size)) / xratio))
		zpixel := int16(math.Round((float64(zmax) - float64(wholefile[x])) / zratio))
		// If the thinned array does not already have two values in it then append this value.
		if len(xslice) >= 1 {
			//If this value is not duplicate to the last then append it.
			if !(xslice[len(xslice)-1] == xpixel && zslice[len(zslice)-1] == zpixel) {
				xslice = append(xslice, xpixel)
				zslice = append(zslice, zpixel)
			}
		} else {
			xslice = append(xslice, xpixel)
			zslice = append(zslice, zpixel)
		}
	}
	xslice = append(xslice, zslice...)
	//const SIZEOF_INT16 = 2 // bytes

	outData := new(bytes.Buffer)
	_ = binary.Write(outData, binary.LittleEndian, &xslice)

	return outData.Bytes()

}

func makeLineExpectedData() []byte {
	expectedReturn := make([]byte, 500)
	for i := 0; i < 5; i++ {
		for j := 0; j < 100; j++ {
			switch i {
			case 0:
				expectedReturn[i*100+j] = 0
			case 1:
				expectedReturn[i*100+j] = 3
			case 2:
				expectedReturn[i*100+j] = 5
			case 3:
				expectedReturn[i*100+j] = 8
			case 4:
				expectedReturn[i*100+j] = 10
			}

		}
	}
	return expectedReturn
}

func makeYcutExpectedData(size, line int) []byte {
	expectedReturn := make([]byte, size)

	var middleValue byte = uint8(line / (size / 10))

	for i := 0; i < 10; i++ {
		expectedReturn[i] = 0
	}
	for i := 10; i < 50; i++ {
		expectedReturn[i] = middleValue
	}
	for i := 50; i < 60; i++ {
		expectedReturn[i] = 10
	}

	return expectedReturn
}
func TestInvalidTransform(t *testing.T) {
	// An unknown transform should result in defaulting to "first."
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 10
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 59, 59, 60, 60, 1, 1, "bad", "Re", "SB", 200, expectedReturn)
}
func TestInvalidCxMode(t *testing.T) {
	// An unknown csmode should result in defaulting to "Real"
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 10
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 59, 59, 60, 60, 1, 1, "first", "bad", "SB", 200, expectedReturn)
}
func TestInvalidCxModeComplex(t *testing.T) {
	// An unknown cxmode should result in defaulting to "Real"
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 10
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 60, 1, 1, "first", "bad", "SB", 200, expectedReturn)
}

func TestInvalidOutfmt(t *testing.T) {
	// A bad outfmt will result in no return data, but not break the service so the the next request should still work
	expectedReturn1 := make([]byte, 0)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 60, 1, 1, "first", "Re", "bad", 200, expectedReturn1)

	expectedReturn2 := make([]byte, 1)
	expectedReturn2[0] = 10
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 60, 1, 1, "first", "Re", "SB", 200, expectedReturn2)
	//Test again with X1 and X2 in reverse order. Should still work.
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 60, 59, 59, 60, 1, 1, "first", "Re", "SB", 200, expectedReturn2)
}

func TestInvalidXYParams(t *testing.T) {

	expectedReturn := make([]byte, 0)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 61, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 59, 61, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", -1, 59, 61, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 61, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 0, 60, 61, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, -1, 60, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 60, -1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 60, 1, -1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 60, 59, 60, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 60, 60, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, -1, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, -1, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 70, 60, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 59, 59, 60, 70, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 70, 60, 59, 59, 1, 1, "first", "Re", "SB", 400, expectedReturn)
	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 60, 70, 59, 59, 1, 1, "first", "Re", "SB", 400, expectedReturn)
}

func TestFirstPoint(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 1, 1, 1, 1, "first", "Re", "SB", 200, expectedReturn)
}

func TestLastPoint(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 10
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 59, 59, 60, 60, 1, 1, "first", "Re", "SB", 200, expectedReturn)
}

func TestAverageFirst10Point(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 10, 10, 1, 1, "mean", "Re", "SB", 200, expectedReturn)
}

func TestAverageMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 1 //Input Data is values of 0,1,2 in equal amounts, averaged together be 1
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "mean", "Re", "SB", 200, expectedReturn)
}
func TestFirstMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0 //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "SB", 200, expectedReturn)
}
func TestMaxMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 2 //Input Data is values of 0,1,2 in equal amounts, max value will return 2.
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "max", "Re", "SB", 200, expectedReturn)
}

func TestMiddlePoint20Log(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 6 //Input Data 2. 20*log10(2) = 6.02 which will return 6
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "L2", "SB", 200, expectedReturn)
}

func TestMiddlePoint10Log(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 3 //Input Data 2. 10*log10(2) = 3.01 which will return 3
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "Lo", "SB", 200, expectedReturn)
}
func TestMiddlePointMag(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 2 //Input Data 2. sqrt(4+0) = 2
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "Ma", "SB", 200, expectedReturn)
}
func TestMiddlePointPh(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = byte(math.Atan2(float64(2), float64(2))) //Input Data 2. Phase is atan(2,2)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "Ph", "SB", 200, expectedReturn)
}
func TestMiddlePointIm(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0 //Imaginary mode for real data returns 0
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "Im", "SB", 200, expectedReturn)
}

func TestMiddlePoints20LogColormap(t *testing.T) {
	expectedReturn := make([]byte, 4)
	expectedReturn[0] = 0 //Input Data 2. 20*log10(2) = 6.02. By setting the zmin to 6.02, the colormap should return the first value which is 0,0,38.
	expectedReturn[1] = 0
	expectedReturn[2] = 38
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "L2", "Ramp Colormap", "6.02", "50", 200, expectedReturn)
}

func TestMiddlePoints20LogColormapMax(t *testing.T) {
	expectedReturn := make([]byte, 4)
	expectedReturn[0] = 255 //Input Data 2. 20*log10(2*2) = 6.02. By setting the zmax to 6.02, the colormap should return the last value which is 255,0,0.
	expectedReturn[1] = 0
	expectedReturn[2] = 0
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "L2", "Ramp Colormap", "0", "6.02", 200, expectedReturn)
}

func TestMiddlePoints20LogColormapMiddle(t *testing.T) {
	expectedReturn := make([]byte, 4)
	expectedReturn[0] = 0 //Input Data 2. 20*log10(2*2) = 6.02. By setting the zmax to 0, "12.041199826559", the colormap should return the middle value which is 0,204,0.
	expectedReturn[1] = 204
	expectedReturn[2] = 0
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 17, 20, 18, 21, 1, 1, "first", "L2", "Ramp Colormap", "0", "12.041199826559", 200, expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmax(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 0             // Should return lowest value in colormap "Ramp Colormap"
	expectedReturn[1] = 0
	expectedReturn[2] = 38
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "Ramp Colormap", "skip", "skip", 200, expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxBadColorMap(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 0             // Should return lowest value in default colormap "Ramp Colormap"
	expectedReturn[1] = 0
	expectedReturn[2] = 38
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "Bad", "skip", "skip", 200, expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmaxGreyscale(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 0             // Should return lowest value in colormap "Ramp Colormap"
	expectedReturn[1] = 0
	expectedReturn[2] = 0
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "Greyscale", "skip", "skip", 200, expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxColorWheel(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 255           // Should return lowest value in colormap "Color Wheel"
	expectedReturn[1] = 255
	expectedReturn[2] = 0
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "Color Wheel", "skip", "skip", 200, expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxSpectrum(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 0             // Should return lowest value in colormap "Spectrum"
	expectedReturn[1] = 191
	expectedReturn[2] = 0
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "Spectrum", "skip", "skip", 200, expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmaxcalewhite(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 255           // Should return lowest value in colormap "calewhite"
	expectedReturn[1] = 255
	expectedReturn[2] = 255
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "calewhite", "skip", "skip", 200, expectedReturn)
}

func TestFirstMiddlePointsColormapNoZinZmaxHotDesat(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 71            // Should return lowest value in colormap "HotDesat"
	expectedReturn[1] = 71
	expectedReturn[2] = 219
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "HotDesat", "skip", "skip", 200, expectedReturn)
}
func TestFirstMiddlePointsColormapNoZinZmaxSunset(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, first value will return 0.
	expectedReturn[0] = 26            // Should return lowest value in colormap "Sunset"
	expectedReturn[1] = 0
	expectedReturn[2] = 59
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "first", "Re", "Sunset", "skip", "skip", 200, expectedReturn)
}

func TestMeanMiddlePointsColormapNoZinZmax(t *testing.T) {

	expectedReturn := make([]byte, 4) //Input Data is values of 0,1,2 in equal amounts, mean value will return 1.
	expectedReturn[0] = 0             // Should 10% point from "Ramp Colormap"
	expectedReturn[1] = 0
	expectedReturn[2] = 128
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, "mean", "Re", "Ramp Colormap", "skip", "skip", 200, expectedReturn)
}

func TestFullReducedMax(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 60, 60, 30, 30, "max", "Re", "SB", 200, expectedReturn)
}
func TestFullReducedMin(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 60, 60, 30, 30, "min", "Re", "SB", 200, expectedReturn)
}

func TestFullReducedFirst(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 60, 60, 30, 30, "first", "Re", "SB", 200, expectedReturn)
}
func TestFullReducedMean(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 60, 60, 30, 30, "mean", "Re", "SB", 200, expectedReturn)
}

func TestFullReducedMaxAbs(t *testing.T) {
	expectedReturn := makeWholeExpectedData(30)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 60, 60, 30, 30, "maxabs", "Re", "SB", 200, expectedReturn)
}

func TestFullSameSizeMean(t *testing.T) {
	expectedReturn := makeWholeExpectedData(60)
	BaseicRDSHandler(t, "mydata_SB_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Re", "SB", 200, expectedReturn)
}

func TestFullISameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]int16, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = int16(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_SI_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Re", "SI", 200, byteData.Bytes())
}

func TestFullLSameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]int32, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = int32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_SL_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Re", "SL", 200, byteData.Bytes())
}

func TestFullFSameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = float32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_SF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Re", "SF", 200, byteData.Bytes())
}

func TestFullDSameSizeMean(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float64, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = float64(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_SD_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Re", "SD", 200, byteData.Bytes())
}

func TestFullCFSameSizeMeanReal(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = float32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Re", "CF", 200, byteData.Bytes())
}
func TestFullCFSameSizeMeanIm(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = float32(expectedResults[i])
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Im", "CF", 200, byteData.Bytes())
}
func TestFullCFSameSizeMeanMag(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = float32(math.Sqrt(float64(expectedResults[i] * expectedResults[i] * 2)))
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Ma", "CF", 200, byteData.Bytes())
}
func TestFullCFSameSizeMeanPh(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	for i := 0; i < len(IntData); i++ {
		IntData[i] = float32(math.Atan2(float64(expectedResults[i]), float64(expectedResults[i])))
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Ph", "CF", 200, byteData.Bytes())
}

func TestFullCFSameSizeMeanLo(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	loThresh := 1.0e-20
	for i := 0; i < len(IntData); i++ {
		mag2 := float64(expectedResults[i]*expectedResults[i] + expectedResults[i]*expectedResults[i])
		mag2 = math.Max(mag2, loThresh)
		IntData[i] = float32(10 * math.Log10(mag2))
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "Lo", "CF", 200, byteData.Bytes())
}
func TestFullCFSameSizeMeanL2(t *testing.T) {
	expectedResults := makeWholeExpectedData(60)
	IntData := make([]float32, len(expectedResults))
	loThresh := 1.0e-20
	for i := 0; i < len(IntData); i++ {
		mag2 := float64(expectedResults[i]*expectedResults[i] + expectedResults[i]*expectedResults[i])
		mag2 = math.Max(mag2, loThresh)
		IntData[i] = float32(20 * math.Log10(mag2))
	}
	byteData := new(bytes.Buffer)
	_ = binary.Write(byteData, binary.LittleEndian, &IntData)

	BaseicRDSHandler(t, "mydata_CF_60_60.tmp", 0, 0, 60, 60, 60, 60, "mean", "L2", "CF", 200, byteData.Bytes())
}

func TestFirstPointSP(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0 //SP file has the bits 0,0,1,1, as the first four
	BaseicRDSHandler(t, "mydata_SP_80_80.tmp", 0, 0, 1, 1, 1, 1, "first", "Re", "SB", 200, expectedReturn)
}

func TestFourPointsSP(t *testing.T) {
	expectedReturn := make([]byte, 4)
	expectedReturn[0] = 0 //SP file has the bits 0,0,1,1, as the first four
	expectedReturn[1] = 0
	expectedReturn[2] = 1
	expectedReturn[3] = 1
	BaseicRDSHandler(t, "mydata_SP_80_80.tmp", 1, 0, 4, 1, 4, 1, "first", "Re", "SB", 200, expectedReturn)
}
func TestSPOutput(t *testing.T) {
	expectedReturn := make([]byte, 2)
	expectedReturn[0] = 48 //SP file has the bits 0,0,1,1,0,0,0,0 as the first 8 which is decimal 48
	expectedReturn[1] = 48
	BaseicRDSHandler(t, "mydata_SP_80_80.tmp", 0, 0, 16, 1, 16, 1, "first", "Re", "SP", 200, expectedReturn)
}

func TestFirstPointSubsize(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0
	BaseicRDSHandlerSubsize(t, "mydata_SB_60_60.tmp", 0, 0, 1, 1, 1, 1, 60, "first", "Re", "SB", 200, expectedReturn)
}

func TestFirstPointSubsize2(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 0
	BaseicRDSHandlerSubsize(t, "mydata_SB_60_60.tmp", 0, 0, 30, 20, 1, 1, 30, "mean", "Re", "SB", 200, expectedReturn)
}

func TestFirstPointSubsize3(t *testing.T) {
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 5
	BaseicRDSHandlerSubsize(t, "mydata_SB_60_60.tmp", 0, 20, 30, 22, 1, 1, 30, "mean", "Re", "SB", 200, expectedReturn)
}
func TestAverageMiddlePointsBadSubsize(t *testing.T) { //Badsubsize should be ignored and result should be the same as if it was not specified.
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 1 //Input Data is values of 0,1,2 in equal amounts, averaged together be 1
	BaseicRDSHandlerSubsize(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, 0, "mean", "Re", "SB", 200, expectedReturn)
}
func TestAverageMiddlePointsBadSubsize2(t *testing.T) { //Badsubsize should be ignored and result should be the same as if it was not specified.
	expectedReturn := make([]byte, 1)
	expectedReturn[0] = 1 //Input Data is values of 0,1,2 in equal amounts, averaged together be 1
	BaseicRDSHandlerSubsize(t, "mydata_SB_60_60.tmp", 0, 20, 18, 21, 1, 1, -1, "mean", "Re", "SB", 200, expectedReturn)
}
func TestXCutFirstLine(t *testing.T) {
	expectedResults := make1DExpectedData("xcut", 60, 0, 60, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, 1, 60, 10, "Re", 200, expectedResults)
}

func TestXCutMiddleLine(t *testing.T) {
	expectedResults := make1DExpectedData("xcut", 60, 30, 60, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 30, 60, 31, 60, 10, "Re", 200, expectedResults)
}

func TestXCutMiddleLineXCompress(t *testing.T) {
	expectedResults := make1DExpectedData("xcut", 60, 30, 30, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 30, 60, 31, 30, 10, "Re", 200, expectedResults)
}

func TestXCutMiddleLineZCompress(t *testing.T) {
	expectedResults := make1DExpectedData("xcut", 60, 30, 60, 5, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 30, 60, 31, 60, 5, "Re", 200, expectedResults)
}

func TestYCutFirstLine(t *testing.T) {
	expectedResults := make1DExpectedData("ycut", 60, 0, 60, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 0, 0, 1, 60, 60, 10, "Re", 200, expectedResults)
}

func TestYCutMiddleLine(t *testing.T) {
	expectedResults := make1DExpectedData("ycut", 60, 20, 60, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 20, 0, 21, 60, 60, 10, "Re", 200, expectedResults)
}

func TestYCutMiddleLine2(t *testing.T) {
	expectedResults := make1DExpectedData("ycut", 60, 40, 60, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 40, 0, 41, 60, 60, 10, "Re", 200, expectedResults)
}
func TestYCutMiddleLineXCompress(t *testing.T) {
	expectedResults := make1DExpectedData("ycut", 60, 40, 30, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 40, 0, 41, 60, 30, 10, "Re", 200, expectedResults)
}
func TestYCutMiddleLineZCompress(t *testing.T) {
	expectedResults := make1DExpectedData("ycut", 60, 40, 60, 5, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 40, 0, 41, 60, 60, 5, "Re", 200, expectedResults)
}

func Test1DLine(t *testing.T) {
	outxsize := 500
	outysize := 10
	expectedResults := make1DExpectedData("line", 500, 0, outxsize, outysize, 0, 10)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, outysize, "Re", 200, expectedResults)
}

func Test1DLineYExpansion(t *testing.T) {
	outxsize := 500
	outysize := 20
	expectedResults := make1DExpectedData("line", 500, 0, outxsize, outysize, 0, 10)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, outysize, "Re", 200, expectedResults)
}

func Test1DLineYCompression(t *testing.T) {
	outxsize := 500
	outysize := 5
	expectedResults := make1DExpectedData("line", 500, 0, outxsize, outysize, 0, 10)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, outysize, "Re", 200, expectedResults)
}
func Test1DLineXCompression(t *testing.T) {
	outxsize := 100
	outysize := 10
	expectedResults := make1DExpectedData("line", 500, 0, outxsize, outysize, 0, 10)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, outysize, "Re", 200, expectedResults)
}

func Test1DLineXExpansion(t *testing.T) {
	outxsize := 1000
	outysize := 10
	expectedResults := make1DExpectedData("line", 500, 0, outxsize, outysize, 0, 10)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, outysize, "Re", 200, expectedResults)
}

func TestXCutInvalidRequests(t *testing.T) {

	// Valid Request

	expectedResults := make1DExpectedData("xcut", 60, 0, 60, 10, 0, 10)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, 1, 60, 10, "Re", 200, expectedResults)

	// Invalid Requests
	expectedResults = make([]byte, 0)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", -1, 0, 60, 1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, -1, 60, 1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, -1, 1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, -1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, 1, 0, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, 1, 60, 0, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, 0, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 0, 0, 0, 60, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 60, 2, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsycut", 0, 0, 2, 60, 60, 10, "Re", 400, expectedResults)

	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 0, 70, 1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 65, 0, 70, 1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 55, 0, 70, 1, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 60, 60, 61, 60, 10, "Re", 400, expectedResults)
	BaseicRDSxCutHandler(t, "mydata_SB_60_60.tmp", "rdsxcut", 0, 62, 60, 63, 60, 10, "Re", 400, expectedResults)
}

func Test1DLineInvalidRequests(t *testing.T) {

	// Valid Request
	outxsize := 500
	outysize := 10
	expectedResults := make1DExpectedData("line", 500, 0, outxsize, outysize, 0, 10)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, outysize, "Re", 200, expectedResults)

	// Invalid Requests
	expectedResults = make([]byte, 0)
	BaseicLDSHandler(t, "stairstep.tmp", -1, 500, outxsize, outysize, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 0, -100, outxsize, outysize, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, 0, outysize, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 500, outxsize, 0, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 0, outxsize, outysize, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 0, 1000, outxsize, outysize, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 400, 600, outxsize, outysize, "Re", 400, expectedResults)
	BaseicLDSHandler(t, "stairstep.tmp", 1000, 1100, outxsize, outysize, "Re", 400, expectedResults)
}
