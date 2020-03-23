package main

import (
    "net/http"
	"net/http/httptest"
//	"net/url"
	"testing"
	"encoding/json"
	"strconv"
	"os"
//	"fmt"
)

// Tests use the data file, "mydata_SB_600_600.tmp". This file is a 600 by 600 scaler byte file where it is 0 for the first 100  lines and 10 for the last 100 lines. 
// For lines between 101-500 it changes based on x value with 10 equal sized portions. Each section of 60 columns increases by 1 starting from 0 and going to 9.
// For example, lines 0-59, are 0, 60-119 are 1 ... 540-599 are 9. 


func TestHDRHandler(t *testing.T) {
    // Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	location_name := "ServiceDir/"
	filename := "mydata_SB_600_600.tmp"
	sdsurl := "/sds/" + location_name + filename +"?mode=hdr"
	//t.Log("url", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	headerServer := &fileHeaderServer{}
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
	   fileHeaderData.Data_size != 360000 ||  fileHeaderData.File_type != 2000 || 
	   fileHeaderData.Subsize != 600 || fileHeaderData.Xdelta != 1  {
		t.Errorf("Incorrect Header Data Returned")
	}

}



func BaseicRDSHandlerColormap(t *testing.T,x1,y1,x2,y2,outxsize,outysize int, transform, cxmode, colormap , zmin,zmax string, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false"}
	location_name := "ServiceDir/"
	filename := "mydata_SB_600_600.tmp"
	sdsurl := "/sds/" + location_name + filename +"?mode=rds" + "&x1=" + strconv.Itoa(x1)+ "&y1="+ strconv.Itoa(y1)+ "&x2="+ strconv.Itoa(x2)+ "&y2="+ strconv.Itoa(y2) + "&outxsize="+ strconv.Itoa(outxsize) +"&outysize="+ strconv.Itoa(outysize) + "&transform=" + transform + "&cxmode=" + cxmode +"&colormap=" + colormap + "&zmin=" +  zmin+"&zmax=" + zmax+ "&outfmt=RGBA"

	t.Log("url:", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	rdsServer := &rdsServer{}
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


func BaseicRDSHandler(t *testing.T,x1,y1,x2,y2,outxsize,outysize int, transform, cxmode string, expectedReturn []byte) {
	os.Args = []string{"cmd", "-usecache=false"}
	location_name := "ServiceDir/"
	filename := "mydata_SB_600_600.tmp"
	sdsurl := "/sds/" + location_name + filename +"?mode=rds" + "&x1=" + strconv.Itoa(x1)+ "&y1="+ strconv.Itoa(y1)+ "&x2="+ strconv.Itoa(x2)+ "&y2="+ strconv.Itoa(y2) + "&outxsize="+ strconv.Itoa(outxsize) +"&outysize="+ strconv.Itoa(outysize) + "&transform=" + transform + "&cxmode=" + cxmode
	t.Log("url:", sdsurl)
	req, err := http.NewRequest("GET", sdsurl, nil)
	//req, err := http.NewRequest("GET", sdsurl, url.Values{"mode": {"hdr"}})
    if err != nil {
        t.Fatal(err)
	}

	setupConfigLogCache()

	rr := httptest.NewRecorder()
	//handler := http.HandlerFunc(fileHeaderServer)
	rdsServer := &rdsServer{}
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
func TestFirstPoint(t *testing.T) {
	expectedReturn := make([]byte,1)
	expectedReturn[0] = 0 
	BaseicRDSHandler(t,0,0,1,1,1,1,"first","Re", expectedReturn)
}

func TestLastPoint(t *testing.T) {
	expectedReturn := make([]byte,1)
	expectedReturn[0] = 10 
	BaseicRDSHandler(t,599,599,600,600,1,1,"first","Re",expectedReturn)
}

func TestAverageFirst10Point(t *testing.T) {
	expectedReturn := make([]byte,1)
	expectedReturn[0] = 0 
	BaseicRDSHandler(t,0,0,10,10,1,1,"mean","Re",expectedReturn)
}

func TestAverageMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 1 //Input Data is values of 0,1,2 in equal amounts, averaged together be 1 
	BaseicRDSHandler(t,0,200,180,201,1,1,"mean","Re",expectedReturn)
}
func TestFirstMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 0 //Input Data is values of 0,1,2 in equal amounts, first value will return 0. 
	BaseicRDSHandler(t,0,200,180,201,1,1,"first","Re",expectedReturn)
}
func TestMaxMiddlePoints(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 2 //Input Data is values of 0,1,2 in equal amounts, max value will return 2. 
	BaseicRDSHandler(t,0,200,180,201,1,1,"max","Re",expectedReturn)
}

func TestMiddlePoint20Log(t *testing.T) {
	expectedReturn := make([]byte,1) 
	expectedReturn[0] = 12 //Input Data 2. 20*log10(2*2) = 12.04 which will return 12 
	BaseicRDSHandler(t,179,200,180,201,1,1,"first","L2",expectedReturn)
}

func TestMiddlePoints20LogColormap(t *testing.T) {
	expectedReturn := make([]byte,4) 
	expectedReturn[0] = 0 //Input Data 2. 20*log10(2*2) = 12.04. By setting the zmin to 12.04, the colormap should return the first value which is 0,0,0. 
	expectedReturn[1] = 0 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,179,200,180,201,1,1,"first","L2","RampColormap", "12.04","50", expectedReturn)
}

func TestMiddlePoints20LogColormapMax(t *testing.T) {
	expectedReturn := make([]byte,4) 
	expectedReturn[0] = 255 //Input Data 2. 20*log10(2*2) = 12.04. By setting the zmax to 12.04, the colormap should return the last value which is 255,0,0. 
	expectedReturn[1] = 0 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,179,200,180,201,1,1,"first","L2","RampColormap", "0","12.04", expectedReturn)
}

func TestMiddlePoints20LogColormapMiddle(t *testing.T) {
	expectedReturn := make([]byte,4) 
	expectedReturn[0] = 0 //Input Data 2. 20*log10(2*2) = 12.04. By setting the zmax to 0, 24.0824, the colormap should return the middle value which is 0,204,0. 
	expectedReturn[1] = 204 
	expectedReturn[2] = 0 
	expectedReturn[3] = 255 //Alpha is always 255
	BaseicRDSHandlerColormap(t,179,200,180,201,1,1,"first","L2","RampColormap", "0","24.0824", expectedReturn)
}