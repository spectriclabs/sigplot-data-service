package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func urlToCacheFileName(url string, query string) string {

	pathData := strings.Split(url, "/")
	fileLocation := pathData[2]
	fileName := pathData[len(pathData)-1]
	response := fmt.Sprintf("%s_%s_%s", fileLocation, fileName, query)
	response = strings.ReplaceAll(response, "&", "")
	response = strings.ReplaceAll(response, "=", "")
	response = strings.ReplaceAll(response, ".", "")

	return response
}

func getDataFromCache(cacheFileName string, subDir string) ([]byte, bool) {
	var outData []byte

	fullPath := fmt.Sprintf("%s%s%s", configuration.CacheLocation, subDir, cacheFileName)
	outData, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Error("Request not in Cache")
		return outData, false
	}
	log.Println("Found File in Cache. FileName: ", cacheFileName)
	return outData, true
}

func getItemFromCache(cacheFileName string, subDir string) (io.ReadSeeker, bool) {

	fullPath := fmt.Sprintf("%s%s%s", configuration.CacheLocation, subDir, cacheFileName)
	file, err := os.Open(fullPath)
	if err != nil {
		log.Println("Request not in Cache", err)
		return nil, false
	}
	return file, true
}

func putItemInCache(cacheFileName string, subDir string, data []byte) {

	fullPath := fmt.Sprintf("%s%s%s", configuration.CacheLocation, subDir, cacheFileName)
	fullPathDirectory := fmt.Sprintf("%s%s", configuration.CacheLocation, subDir)
	if _, err := os.Stat(fullPathDirectory); os.IsNotExist(err) {
		mkdirErr := os.MkdirAll(fullPathDirectory, 0700)
		if mkdirErr != nil {
			log.Fatalf("Error making directory: %s: %s", fullPathDirectory, mkdirErr)
		}
	}
	file, err := os.Create(fullPath)
	if err != nil {
		log.Errorf("Error creating cache file: %s", err)
		return
	}
	num, err := file.Write(data)
	if err != nil || num != len(data) {
		log.Errorf("Error creating cache file: %s", err)
		return
	}
	log.Printf("Cached Data to File %s", cacheFileName)
}

func checkCache(cachePath string, every int, maxBytes int64) {
	// make `cachePath` directory if it doesn't exist
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		log.Debugf("Making directory %s", cachePath)
		mkdirErr := os.MkdirAll(cachePath, 0700)
		if mkdirErr != nil {
			log.Fatalf("Error making directory: %s: %s", cachePath, mkdirErr)
		}
	}

	// duration expressed in nano seconds
	nextRun := time.Now()
	for {
		if nextRun.Before(time.Now()) {

			files, err := ioutil.ReadDir(cachePath)
			if err != nil {
				log.Errorf("checkCache Error: %s", err)
				time.Sleep(5 * time.Second)
				continue
			}

			var currentBytes int64 = 0
			var oldestFile os.FileInfo
			if len(files) > 0 {
				oldestFile = files[0]
			}
			for _, file := range files {
				if !(file.IsDir()) {
					currentBytes += file.Size()
					if file.ModTime().Before(oldestFile.ModTime()) {
						oldestFile = file
					}

				}

			}
			if currentBytes > maxBytes {
				path := fmt.Sprintf("%s%s", cachePath, oldestFile.Name())

				if strings.Contains(oldestFile.Name(), "x1") && strings.Contains(oldestFile.Name(), "x2") &&
					strings.Contains(oldestFile.Name(), "y1") && strings.Contains(oldestFile.Name(), "y2") &&
					strings.Contains(oldestFile.Name(), "outxsize") && strings.Contains(oldestFile.Name(), "outysize") {
					log.Println("Cache over Maximum. Removing Old File", oldestFile.Name())
					err = os.Remove(path)
					if err != nil {
						log.Errorf("Error removing cache file: %s", err)
					}
				} else {
					log.Println("Almost Removed a file that was in the cache directory but doesn't appear to be in the format of sds. Don't put non-SDS files in the cache dir")
				}

			} else {
				nextRun = nextRun.Add(time.Second * time.Duration(every))
			}
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}
