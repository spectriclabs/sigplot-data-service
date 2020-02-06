package main

import (
   "log"
	"fmt"
	"os"
	"strings"
	"io/ioutil"
	"time"
)
func urlToCacheFileName(url string,query string) string {

	pathData := strings.Split(url, "/")
	fileName :=pathData[2]
	response:=fmt.Sprintf("%s_%s", fileName,query)
	response = strings.ReplaceAll(response, "&", "") 
	response = strings.ReplaceAll(response, "=", "") 
	response = strings.ReplaceAll(response, ".", "") 
	
	return response
}

func getItemFromCache(cacheFileName string) ([]byte,bool) {
	var outData []byte

	path:=fmt.Sprintf("%s%s", configuration.CacheLocation,cacheFileName)
	outData, err := ioutil.ReadFile(path)
    if err != nil {
		log.Println("Request not in Cache")
		return outData, false
	} 
	log.Println("Found File in Cache. FileName: " , cacheFileName)
	return outData, true
}

func putItemInCache(cacheFileName string, data []byte) {

	path:=fmt.Sprintf("%s%s", configuration.CacheLocation,cacheFileName)
    file, err := os.Create(path)
    if err != nil {
        log.Println("Error creating Cache File" , err)
        return
	}
	num,err := file.Write(data)
	if err!=nil || num!=len(data) {
		log.Println("Error creating Cache File" , err)
        return
	}
	log.Println("Cached Data to File " , cacheFileName)
}

func checkCache(cachePath string, every int, maxBytes int64) {

	// duration expressed in nano seconds

	nextRun := time.Now()
	for  {
		if nextRun.Before(time.Now()){

			files, err := ioutil.ReadDir(cachePath)
			if err != nil {
				log.Fatal(err)
			}

			var currentBytes int64 =0
			var oldestFile os.FileInfo 
			if len(files) >0 {
				oldestFile = files[0]
			}
			for _, file := range files {
				if !(file.IsDir()) {
					currentBytes+=file.Size()
					if file.ModTime().Before(oldestFile.ModTime()) {
						oldestFile = file
					}

				}
				
			}
			if currentBytes > maxBytes {
				path:=fmt.Sprintf("%s%s", cachePath,oldestFile.Name())

				if strings.Contains(oldestFile.Name(),"x1") && strings.Contains(oldestFile.Name(),"x2")&& 
				strings.Contains(oldestFile.Name(),"y1")&& strings.Contains(oldestFile.Name(),"y2")&& 
				strings.Contains(oldestFile.Name(),"outxsize")&& strings.Contains(oldestFile.Name(),"outysize") {
					log.Println("Cache over Maximum. Removing Old File" , oldestFile.Name())
					err = os.Remove(path)
					if err != nil {
						log.Println("Error remove cache file" , err)
					}
				} else {
					log.Println("Almost Removed a file that was in the cache directory but doesn't appear to be in the format of sds. Don't put non-SDS files in the cache dir")
				}

			} else {
			nextRun = nextRun.Add(time.Second * time.Duration(every))
			}
		} else {
			time.Sleep(5*time.Second)
		}
	}
}


