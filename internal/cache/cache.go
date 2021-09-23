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

type Cache struct {
	Location string
}

// UrlToCacheFileName uses a url and query string
// to form SigPlot Data Services' cached file name.
func UrlToCacheFileName(url string) string {
	response := strings.Replace(url, "?", "_", 1)
	replacer := strings.NewReplacer("&", "", "=", "", ".", "", "/", "")
	cacheFileName := replacer.Replace(response)
	return cacheFileName
}

// GetDataFromCache retrieves data from a provided `cacheFileName`
// within a `subDir` directory
func (c *Cache) GetDataFromCache(cacheFileName string, subDir string) ([]byte, error) {
	fullPath := fmt.Sprintf("%s%s%s", c.Location, subDir, cacheFileName)
	outData, err := ioutil.ReadFile(fullPath)
	return outData, err
}

// GetItemFromCache retrieves a file from a `cacheFileName`
// within a `subDir` directory and returns an `io.ReadSeeker`
func (c *Cache) GetItemFromCache(cacheFileName string, subDir string) (io.ReadSeeker, error) {
	fullPath := fmt.Sprintf("%s%s%s", c.Location, subDir, cacheFileName)
	file, err := os.Open(fullPath)
	return file, err
}

// PutItemInCache places `data` into file denoted by `cacheFileName`
// within `subDir`
func (c *Cache) PutItemInCache(cacheFileName string, subDir string, data []byte) error {
	fullPath := fmt.Sprintf("%s%s%s", c.Location, subDir, cacheFileName)
	fullPathDirectory := filepath.Dir(fullPath)
	if _, err := os.Stat(fullPathDirectory); os.IsNotExist(err) {
		mkdirErr := os.Mkdir(fullPathDirectory, 0755)
		return mkdirErr
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	num, err := file.Write(data)
	if err != nil || num != len(data) {
		return err
	}

	return nil
}

// CheckCache runs a check every `checkInterval` seconds
// and purges if the current cache size exceeds `maxBytes`
func CheckCache(cachePath string, checkInterval int, maxBytes int64) {
	// duration expressed in nano seconds
	nextRun := time.Now()
	for {
		if nextRun.Before(time.Now()) {

			files, err := ioutil.ReadDir(cachePath)
			if err != nil {
				log.Println("CheckCache Error: ", err)
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
				nextRun = nextRun.Add(time.Second * time.Duration(checkInterval))
			}
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}
