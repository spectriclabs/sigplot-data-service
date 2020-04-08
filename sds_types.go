package main

import (
	"io"
	"encoding/binary"
	"math"
)

type Zminzmax struct {
	Zmin float64
	Zmax float64
}

type rdsRequest struct {
	TileRequest bool
	FileFormat string
	FileName string
	FileType,FileSubsize int
	FileXSize,FileYSize int
	FileDataSize float64
	FileDataOffset int
	TileXSize,TileYSize,DecXMode,DecYMode,TileX,TileY int 
	DecX,DecY int
	Zset bool
	Subsize int
	SubsizeSet bool
	Transform string
	ColorMap string
	Reader io.ReadSeeker
	Cxmode string
	CxmodeSet bool
	OutputFmt string
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
	X1,X2,Y1,Y2 int

}
func (request *rdsRequest) computeYSize () {
	request.FileYSize = int(float64(request.FileDataSize) / bytesPerAtomMap[string(request.FileFormat[1])])/(request.FileXSize)
	if string(request.FileFormat[0]) == "C" {
		request.FileYSize = request.FileYSize/2
	}
}

func (request *rdsRequest) computeTileSizes () {
	request.DecX = decimationLookup[request.DecXMode]
	request.DecY = decimationLookup[request.DecYMode]

	request.Xstart = request.TileX*request.TileXSize* request.DecX
	request.Ystart =  request.TileY* request.TileYSize* request.DecY
	request.Xsize = int(request.TileXSize* request.DecX)
	request.Ysize = int(request.TileYSize* request.DecY)
	request.Outxsize = request.TileXSize
	request.Outysize = request.TileYSize
}
func (request *rdsRequest) computeRequestSizes () {
	request.Ystart = int(math.Min(float64(request.Y1), float64(request.Y2)))
	request.Xstart = int(math.Min(float64(request.X1), float64(request.X2)))
	request.Xsize = int(math.Abs(float64(request.X2) - float64(request.X1)))
	request.Ysize = int(math.Abs(float64(request.Y2) - float64(request.Y1)))
}

func (request *rdsRequest) processBlueFileHeader() {

	//TODO - Convert to just work on the rdsRequest struct and store the new values back into it. 
	var bluefileheader BlueHeader
	binary.Read(request.Reader, binary.LittleEndian, &bluefileheader)

	request.FileFormat = string(bluefileheader.Format[:])
	request.FileType = int(bluefileheader.File_type)
	request.FileXSize = int(bluefileheader.Subsize)
	request.Filexstart = bluefileheader.Xstart
	request.Filexdelta = bluefileheader.Xdelta
	request.Fileystart = bluefileheader.Ystart
	request.Fileydelta = bluefileheader.Ydelta
	request.FileDataOffset = int(bluefileheader.Data_start)
	request.FileDataSize = bluefileheader.Data_size

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
	MaxBytesZminZmax int 	   `json:"maxBytesZminZmax"`
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
