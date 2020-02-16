package cache

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

func UrlToCacheFileName(fileLocation string, fileName string, query string) string {
	response := fmt.Sprintf("%s_%s_%s", fileLocation, fileName, query)
	response = strings.ReplaceAll(response, "&", "")
	response = strings.ReplaceAll(response, "=", "")
	response = strings.ReplaceAll(response, ".", "")
	return response
}

func GetDataFromCache(cacheLocation string, cacheFileName string, subDir string) ([]byte, bool) {
	fullPath := filepath.Join(cacheLocation, subDir, cacheFileName)
	outData, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Println("Request not in Cache")
		return outData, false
	}
	return outData, true
}

func GetItemFromCache(cacheLocation string, cacheFileName string, subDir string) (io.ReadSeeker, bool) {
	fullPath := filepath.Join(cacheLocation, subDir, cacheFileName)
	file, err := os.Open(fullPath)
	if err != nil {
		log.Println("Request not in Cache", err)
		return nil, false
	}
	return file, true
}

func PutItemInCache(cacheLocation string, cacheFileName string, subDir string, data []byte) {
	fullPath := filepath.Join(cacheLocation, subDir, cacheFileName)
	fullPathDirectory := filepath.Join(cacheLocation, subDir)
	if _, err := os.Stat(fullPathDirectory); os.IsNotExist(err) {
		os.MkdirAll(fullPathDirectory, 0700)
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

func CheckCache(cachePath string, every int, maxBytes int64) {
	// duration expressed in nano seconds
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		os.MkdirAll(cachePath, 0700)
	}
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
				path := filepath.Join(cachePath, oldestFile.Name())

				if strings.Contains(oldestFile.Name(), "x1") && strings.Contains(oldestFile.Name(), "x2") &&
					strings.Contains(oldestFile.Name(), "y1") && strings.Contains(oldestFile.Name(), "y2") &&
					strings.Contains(oldestFile.Name(), "outxsize") && strings.Contains(oldestFile.Name(), "outysize") {
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
