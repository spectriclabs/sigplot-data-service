package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/// urlToCacheFileName concatenates a `url` and `query`
/// to form SigPlot Data Services' cached file name.
func urlToCacheFileName(url string, query string) string {
	response := fmt.Sprintf("%s_%s", url, query)
	replacer := strings.NewReplacer("&", "", "=", "", ".", "", "/", "")
	cacheFileName := replacer.Replace(response)
	return cacheFileName
}

func getDataFromCache(cacheFileName string, subDir string) ([]byte, bool) {
	fullPath := fmt.Sprintf("%s%s%s", configuration.CacheLocation, subDir, cacheFileName)
	outData, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return outData, false
	}
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
	fullPathDirectory := filepath.Dir(fullPath)
	if _, err := os.Stat(fullPathDirectory); os.IsNotExist(err) {
		os.Mkdir(fullPathDirectory, 0755)
	}
	file, err := os.Create(fullPath)
	if err != nil {
		log.Println("Error creating Cache File", err)
		return
	}
	num, err := file.Write(data)
	if err != nil || num != len(data) {
		log.Println("Error creating Cache File", err)
		return
	}
}

func checkCache(cachePath string, every int, maxBytes int64) {
	// duration expressed in nano seconds
	nextRun := time.Now()
	for {
		if nextRun.Before(time.Now()) {

			files, err := ioutil.ReadDir(cachePath)
			if err != nil {
				log.Println("checkCache Error: ", err)
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

				if strings.Contains(oldestFile.Name(), "sds") && (strings.Contains(oldestFile.Name(), "rds") || strings.Contains(oldestFile.Name(), "lds")) {
					log.Println("Cache over Maximum. Removing Old File", oldestFile.Name())
					err = os.Remove(path)
					if err != nil {
						log.Println("Error remove cache file", err)
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
