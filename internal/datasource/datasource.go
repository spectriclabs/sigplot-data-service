package datasource

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sigplot-data-service/internal/cache"
	"sigplot-data-service/internal/config"
	"sync"
	"time"

	"github.com/minio/minio-go/v6"
	"go.uber.org/zap"
)

var ioMutex = &sync.Mutex{}

func GetBytesFromReader(reader io.ReadSeeker, firstByte int, numbytes int) ([]byte, bool) {
	outData := make([]byte, numbytes)
	// Multiple Concurrent goroutines will use this function with the same reader.
	ioMutex.Lock()
	reader.Seek(int64(firstByte), io.SeekStart)
	numRead, err := reader.Read(outData)
	ioMutex.Unlock()

	if numRead != numbytes || err != nil {
		log.Println("Failed to Read Requested Bytes", err, numRead, numbytes)
		return outData, false
	}
	return outData, true
}

func OpenDataSource(
	configuration config.Configuration,
	logger *zap.Logger,
	locationName string,
	fileName string,
) (io.ReadSeeker, string, bool) {
	var currentLocation config.Location
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
		fullFilepath := filepath.Join(currentLocation.Path, fileName)
		logger.Info(
			"Reading local file",
			zap.String("location_name", locationName),
			zap.String("filename", fileName),
			zap.String("path", fullFilepath),
		)
		file, err := os.Open(fullFilepath)
		if err != nil {
			logger.Error("Error opening File,", zap.Error(err))
			return nil, "", false
		}
		reader := io.ReadSeeker(file)
		return reader, fileName, true
	case "minio":
		start := time.Now()
		fullFilepath := filepath.Join(currentLocation.Path, fileName)
		cacheFileName := filepath.Join(currentLocation.MinioBucket, fullFilepath, "x1y1x2y2outxsizeoutysize")
		file, inCache := cache.GetItemFromCache(configuration.CacheLocation, cacheFileName, "miniocache/")
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

			cache.PutItemInCache(configuration.CacheLocation, cacheFileName, "miniocache/", fileData)
			cacheFileFullpath := filepath.Join(configuration.CacheLocation, "miniocache/", cacheFileName)
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
		logger.Error(
			"Unsupported location type",
			zap.String("location_type", currentLocation.LocationType),
		)
		return nil, "", false
	}
}
