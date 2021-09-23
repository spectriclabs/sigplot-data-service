package sds

import (
	"encoding/binary"
	"gonum.org/v1/gonum/floats"
	"io"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/spectriclabs/sigplot-data-service/internal/bluefile"
)

type Zminzmax struct {
	Zmin float64
	Zmax float64
}

type RdsRequest struct {
	TileRequest    bool
	FileFormat     string
	FileName       string
	FileType       int
	FileSubsize    int
	FileXSize      int
	FileYSize      int
	FileDataSize   float64
	FileDataOffset int
	TileXSize      int `param:"tileXsize"`
	TileYSize      int `param:"tileYsize"`
	DecXMode       int `param:"decXmode"`
	DecYMode       int `param:"dexYmode"`
	TileX          int `param:"tileX"`
	TileY          int `param:"tileY"`
	DecX           int
	DecY           int
	Zset           bool
	Subsize        int `query:"subsize,omitempty"`
	SubsizeSet     bool
	Transform      string `query:"transform"`
	ColorMap       string `query:"colormap,omitempty"`
	Reader         io.ReadSeeker
	Cxmode         string `query:"cxmode,omitempty"`
	CxmodeSet      bool
	OutputFmt      string  `query:"outfmt,omitempty"`
	Outxsize       int     `json:"outxsize"`
	Outysize       int     `json:"outysize"`
	Outzsize       int     `json:"outzsize"`
	Zmin           float64 `json:"zmin" query:"zmin,omitempty"`
	Zmax           float64 `json:"zmax" query:"zmax,omitempty"`
	Filexstart     float64 `json:"filexstart"`
	Filexdelta     float64 `json:"filexdelta"`
	Fileystart     float64 `json:"fileystart"`
	Fileydelta     float64 `json:"fileydelta"`
	Xstart         int     `json:"xstart"`
	Xsize          int     `json:"xsize"`
	Ystart         int     `json:"ystart"`
	Ysize          int     `json:"ysize"`
	X1, X2, Y1, Y2 int
}

func (request *RdsRequest) ComputeYSize() {
	request.FileYSize = int(request.FileDataSize/bluefile.BytesPerAtomMap[string(request.FileFormat[1])]) / (request.FileXSize)
	if string(request.FileFormat[0]) == "C" {
		request.FileYSize = request.FileYSize / 2
	}
}

func (request *RdsRequest) ComputeTileSizes() {
	request.DecX = DecimationLookup[request.DecXMode]
	request.DecY = DecimationLookup[request.DecYMode]

	request.Xstart = request.TileX * request.TileXSize * request.DecX
	request.Ystart = request.TileY * request.TileYSize * request.DecY
	request.Xsize = request.TileXSize * request.DecX
	request.Ysize = request.TileYSize * request.DecY
	request.Outxsize = request.TileXSize
	request.Outysize = request.TileYSize
}

func (request *RdsRequest) ComputeRequestSizes() {
	request.Ystart = int(math.Min(float64(request.Y1), float64(request.Y2)))
	request.Xstart = int(math.Min(float64(request.X1), float64(request.X2)))
	request.Xsize = int(math.Abs(float64(request.X2) - float64(request.X1)))
	request.Ysize = int(math.Abs(float64(request.Y2) - float64(request.Y1)))
}

func (request *RdsRequest) ProcessBlueFileHeader() {
	var bluefileheader bluefile.BlueHeader
	binary.Read(request.Reader, binary.LittleEndian, &bluefileheader)

	request.FileFormat = string(bluefileheader.Format[:])
	request.FileType = int(bluefileheader.FileType)
	request.FileXSize = int(bluefileheader.Subsize)
	request.Filexstart = bluefileheader.Xstart
	request.Filexdelta = bluefileheader.Xdelta
	request.Fileystart = bluefileheader.Ystart
	request.Fileydelta = bluefileheader.Ydelta
	request.FileDataOffset = int(bluefileheader.DataStart)
	request.FileDataSize = bluefileheader.DataSize
}

func (request *RdsRequest) GetQueryParams(r *http.Request) {
	var ok bool
	// Get URL Query Params
	request.Transform, ok = GetURLQueryParamString(r, "transform")
	if !ok {
		request.Transform = "first"
	}
	request.SubsizeSet = true
	request.Subsize, ok = GetURLQueryParamInt(r, "subsize")
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
	request.Cxmode, ok = GetURLQueryParamString(r, "cxmode")
	if !ok {
		request.Cxmode = "Re"
		request.CxmodeSet = false
	}
	var zminSet, zmaxSet bool
	request.Zmin, zminSet = GetURLQueryParamFloat(r, "zmin")
	if !zminSet {
		request.Zmin = 0
	}
	request.Zmax, zmaxSet = GetURLQueryParamFloat(r, "zmax")
	if !zmaxSet {
		request.Zmax = 0
	}
	request.Zset = (zmaxSet && zminSet)
	request.ColorMap, ok = GetURLQueryParamString(r, "colormap")
	if !ok {
		log.Println("colorMap Not Specified.Defaulting to RampColormap")
		request.ColorMap = "RampColormap"
	}
	request.OutputFmt, ok = GetURLQueryParamString(r, "outfmt")
	if !ok {
		log.Println("Outformat Not Specified. Setting Equal to Input Format")
		request.OutputFmt = "RGBA"

	}
}

var ZminzmaxFileMap map[string]Zminzmax

var DecimationLookup = map[int]int{
	1:  1,
	2:  2,
	3:  4,
	4:  8,
	5:  16,
	6:  32,
	7:  64,
	8:  128,
	9:  256,
	10: 512,
}

type File struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
}

type FileMetaData struct {
	Outxsize   int     `json:"outxsize"`
	Outysize   int     `json:"outysize"`
	Outzsize   int     `json:"outzsize"`
	Zmin       float64 `json:"zmin"`
	Zmax       float64 `json:"zmax"`
	Filexstart float64 `json:"filexstart"`
	Filexdelta float64 `json:"filexdelta"`
	Fileystart float64 `json:"fileystart"`
	Fileydelta float64 `json:"fileydelta"`
	Xstart     int     `json:"xstart"`
	Xsize      int     `json:"xsize"`
	Ystart     int     `json:"ystart"`
	Ysize      int     `json:"ysize"`
}

var IoMutex = &sync.Mutex{}
var ZMinMaxTileMutex = &sync.Mutex{}

func (request *RdsRequest) FindZminMax(maxBytesZminZmax int) {
	start := time.Now()
	ZMinMaxTileMutex.Lock()
	zminmax, ok := ZminzmaxFileMap[request.FileName+request.Cxmode]
	if ok {
		request.Zmin = zminmax.Zmin
		request.Zmax = zminmax.Zmax
	} else {
		var zminmaxRequest RdsRequest
		zminmaxRequest = *request
		zminmaxRequest.Ysize = 1
		zminmaxRequest.Xsize = zminmaxRequest.FileXSize
		zminmaxRequest.Xstart = 0
		zminmaxRequest.Outysize = 1
		zminmaxRequest.Outxsize = 1
		zminmaxRequest.OutputFmt = "SD"
		bytesPerAtom, complexFlag := bluefile.GetFileTypeInfo(request.FileFormat)
		bytesPerElement := bytesPerAtom
		if complexFlag {
			bytesPerElement = bytesPerElement * 2
		}
		log.Println("Computing Zminmax", bytesPerElement, request.FileXSize, request.FileYSize, maxBytesZminZmax)
		// File is small enough, look at entire file for Zmax/Zmin

		if (int(float64(request.FileXSize*request.FileYSize) * (bytesPerElement))) < maxBytesZminZmax {
			log.Println("Computing Zmax/Zmin on whole file, not previously computed")
			min := make([]float64, request.FileYSize)
			max := make([]float64, request.FileYSize)
			done := make(chan bool, 1)
			for line := 0; line < request.FileYSize; line++ {
				zminmaxRequest.Ystart = line
				zminmaxRequest.Transform = "min"
				go ProcessLine(min, line, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go ProcessLine(max, line, done, zminmaxRequest)
			}
			for i := 0; i < request.FileYSize*2; i++ {
				<-done
			}
			request.Zmin = floats.Min(min)
			request.Zmax = floats.Max(max)
			ZminzmaxFileMap[request.FileName+request.Cxmode] = Zminzmax{request.Zmin, request.Zmax}
		} else if request.FileYSize == 1 { //If the file is large but only has one line then we need to break it into section in the x direction.
			log.Println("Computing Zmax/Zmin on section of 1D file, not previously computed")
			numSubSections := 4
			min := make([]float64, numSubSections)
			max := make([]float64, numSubSections)
			done := make(chan bool, 1)
			spaceBytes := (float64(request.FileXSize) * bytesPerElement) - float64(maxBytesZminZmax)
			elementsPerSpace := int(spaceBytes/bytesPerElement) / (numSubSections - 1)
			elementsPerSection := maxBytesZminZmax / numSubSections

			zminmaxRequest.Xsize = elementsPerSection
			// First section of the file
			zminmaxRequest.Xstart = 0
			zminmaxRequest.Transform = "min"
			go ProcessLine(min, 0, done, zminmaxRequest)
			zminmaxRequest.Transform = "max"
			go ProcessLine(max, 0, done, zminmaxRequest)
			// Middle Sections of the file
			for section := 1; section < numSubSections-1; section++ {
				zminmaxRequest.Xstart = section * (elementsPerSection + elementsPerSpace)
				zminmaxRequest.Transform = "min"
				go ProcessLine(min, section, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go ProcessLine(max, section, done, zminmaxRequest)

			}

			// Last Section of the file
			zminmaxRequest.Xstart = request.FileXSize - elementsPerSection
			zminmaxRequest.Transform = "min"
			go ProcessLine(min, numSubSections-1, done, zminmaxRequest)
			zminmaxRequest.Transform = "max"
			go ProcessLine(max, numSubSections-1, done, zminmaxRequest)
			for i := 0; i < numSubSections*2; i++ {
				<-done
			}
			request.Zmin = floats.Min(min)
			request.Zmax = floats.Max(max)
			ZminzmaxFileMap[request.FileName+request.Cxmode] = Zminzmax{request.Zmin, request.Zmax}

		} else { // If file is large and has multiple lines then CheckError the first, last, and a number of middles lines
			numMiddlesLines := int(math.Max(float64((maxBytesZminZmax/request.FileXSize)-2), 0))
			log.Println("Computing Zmax/Zmin on sampling of file, not previously computed. Number of middle lines:", numMiddlesLines)
			min := make([]float64, 2+numMiddlesLines)
			max := make([]float64, 2+numMiddlesLines)
			done := make(chan bool, 1)
			numRequested := 0
			// Process Min and Max of first line
			zminmaxRequest.Ystart = 0
			zminmaxRequest.Transform = "min"
			go ProcessLine(min, 0, done, zminmaxRequest)
			zminmaxRequest.Transform = "max"
			go ProcessLine(max, 0, done, zminmaxRequest)
			numRequested += 2

			//Process Min and Max of last line
			zminmaxRequest.Ystart = request.FileYSize - 1
			if zminmaxRequest.Ystart != 0 { // If the last line is the first line, don't do it again.
				zminmaxRequest.Transform = "min"
				go ProcessLine(min, 1, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go ProcessLine(max, 1, done, zminmaxRequest)
				numRequested += 2
			}

			//Process Min and Max from lines evenly spaced in the middle
			for i := 0; i < numMiddlesLines; i++ {
				zminmaxRequest.Ystart = ((request.FileYSize) / numMiddlesLines) * i
				zminmaxRequest.Transform = "min"
				go ProcessLine(min, i+2, done, zminmaxRequest)
				zminmaxRequest.Transform = "max"
				go ProcessLine(max, i+2, done, zminmaxRequest)
				numRequested += 2
			}
			for i := 0; i < numRequested; i++ {
				<-done
			}
			request.Zmin = floats.Min(min)
			request.Zmax = floats.Max(max)
			ZminzmaxFileMap[request.FileName+request.Cxmode] = Zminzmax{request.Zmin, request.Zmax}

		}
		elapsed := time.Since(start)
		log.Println("Found Zmin, Zmax to be", request.Zmin, request.Zmax, " in ", elapsed)

	}
	ZMinMaxTileMutex.Unlock()
}
