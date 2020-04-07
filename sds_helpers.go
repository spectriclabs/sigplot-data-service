package main

import (

	"log"
	"net/http"
	"strconv"
	"strings"

)


func getURLQueryParamFloat(r *http.Request, keyname string) (float64, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return 0.0, false
	}
	retval, err := strconv.ParseFloat(keys[0], 64)
	if err != nil {
		log.Println("Url Param ", keyname, "  is invalid")
		return 0.0, false
	}
	return retval, true
}

func getURLQueryParamInt(r *http.Request, keyname string) (int, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return 0, false
	}
	retval, err := strconv.Atoi(keys[0])
	if err != nil {
		log.Println("Url Param ", keyname, "  is invalid")
		return 0, false
	}
	return retval, true
}

func getURLQueryParamString(r *http.Request, keyname string) (string, bool) {
	keys, ok := r.URL.Query()[keyname]

	if !ok || len(keys[0]) < 1 {
		return "", false
	}
	return keys[0], true
}

func getURLArgumentFloat(url string, positionNum int) (float64, bool) {
	pathData := strings.Split(url, "/")
	param := pathData[positionNum]
	retval, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return 0.0, false
	}
	return retval, true
}

func getURLArgumentInt(url string, positionNum int) (int, bool) {
	pathData := strings.Split(url, "/")
	param := pathData[positionNum]
	retval, err := strconv.Atoi(param)
	if err != nil {
		return 0, false
	}
	return retval, true
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func intInSlice(a int, list []int) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func getURLArgumentSting(url string, positionNum int) (string, bool) {
	pathData := strings.Split(url, "/")
	param := pathData[positionNum]
	if param !="" {
		return param, true
	}
	return "",false
}
