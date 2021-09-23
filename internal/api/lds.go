package api

type ldsServer struct{}

//func (s *ldsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	var data []byte
//	var inCache bool
//	var ok bool
//	var rdsRequest sds.RdsRequest
//
//	//Get URL Parameters
//	//url - /sds/lds/x1/x2/outxsize/outzsize
//
//	rdsRequest.X1, ok = sds.GetURLArgumentInt(r.URL.Path, 3)
//	if !ok || rdsRequest.X1 < 0 {
//		log.Println("X1 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//	rdsRequest.X2, ok = sds.GetURLArgumentInt(r.URL.Path, 4)
//	if !ok || rdsRequest.X2 < 0 {
//		log.Println("X2 Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.Outxsize, ok = sds.GetURLArgumentInt(r.URL.Path, 5)
//	if !ok || rdsRequest.Outxsize < 1 {
//		log.Println("outxsize Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.Outzsize, ok = sds.GetURLArgumentInt(r.URL.Path, 6)
//	if !ok || rdsRequest.Outzsize < 1 {
//		log.Println("outzsize Missing or Bad. Required Field")
//		w.WriteHeader(400)
//		return
//	}
//
//	rdsRequest.GetQueryParams(r)
//
//	rdsRequest.ComputeRequestSizes()
//
//	rdsRequest.Ystart = 0
//	rdsRequest.Ysize = 1
//
//	if rdsRequest.Xsize < 1 {
//		log.Println("Bad Xsize: ", rdsRequest.Xsize)
//		w.WriteHeader(400)
//		return
//	}
//
//	log.Println("LDS Request params xstart, xsize, outxsize, outzsize:", rdsRequest.Xstart, rdsRequest.Xsize, rdsRequest.Outxsize, rdsRequest.Outzsize)
//
//	start := time.Now()
//	cacheFileName := cache.UrlToCacheFileName(r.URL.Path, r.URL.RawQuery)
//	// Check if request has been previously processed and is in cache. If not process Request.
//	if *useCache {
//		data, inCache = cache.GetDataFromCache(cacheFileName, "outputFiles/")
//	} else {
//		inCache = false
//	}
//
//	if !inCache { // If the output is not already in the cache then read the data file and do the processing.
//		log.Println("RDS Request not in Cache, computing result")
//		rdsRequest.Reader, rdsRequest.FileName, ok = OpenDataSource(r.URL.Path, 7)
//		if !ok {
//			w.WriteHeader(400)
//			return
//		}
//
//		if strings.Contains(rdsRequest.FileName, ".tmp") || strings.Contains(rdsRequest.FileName, ".prm") {
//			rdsRequest.ProcessBlueFileHeader()
//			if rdsRequest.FileType != 1000 {
//				log.Println("Line Plots only support Type 100 files.")
//				w.WriteHeader(400)
//				return
//			}
//			rdsRequest.FileXSize = int(float64(rdsRequest.FileDataSize) / bluefile.BytesPerAtomMap[string(rdsRequest.FileFormat[1])])
//			rdsRequest.FileYSize = 1
//		} else {
//			log.Println("Invalid File Type")
//			w.WriteHeader(400)
//			return
//		}
//		// Check Request against File Size
//		if rdsRequest.Xsize > rdsRequest.FileXSize {
//			log.Println("Invalid Request. Requested X size greater than file X size")
//			w.WriteHeader(400)
//			return
//		}
//		if rdsRequest.X1 > rdsRequest.FileXSize {
//			log.Println("Invalid Request. Requested X1 greater than file X size")
//			w.WriteHeader(400)
//			return
//		}
//		if rdsRequest.X2 > rdsRequest.FileXSize {
//			log.Println("Invalid Request. Requested X2 greater than file X size")
//			w.WriteHeader(400)
//			return
//		}
//
//		//If Zmin and Zmax were not explitily given then compute
//		if !rdsRequest.Zset {
//			rdsRequest.FindZminMax()
//		}
//
//		data = processLineRequest(rdsRequest, "lds")
//
//		if *useCache {
//			go cache.PutItemInCache(cacheFileName, "outputFiles/", data)
//		}
//
//		// Store MetaData of request off in cache
//		var fileMData sds.FileMetaData
//		fileMData.Outxsize = rdsRequest.Outxsize
//		fileMData.Outysize = rdsRequest.Outysize
//		fileMData.Outzsize = rdsRequest.Outzsize
//		fileMData.Filexstart = rdsRequest.Filexstart
//		fileMData.Filexdelta = rdsRequest.Filexdelta
//		fileMData.Fileystart = rdsRequest.Fileystart
//		fileMData.Fileydelta = rdsRequest.Fileydelta
//		fileMData.Xstart = rdsRequest.Xstart
//		fileMData.Ystart = rdsRequest.Ystart
//		fileMData.Xsize = rdsRequest.Xsize
//		fileMData.Ysize = rdsRequest.Ysize
//		fileMData.Zmin = rdsRequest.Zmin
//		fileMData.Zmax = rdsRequest.Zmax
//
//		//var marshalError error
//		fileMDataJSON, marshalError := json.Marshal(fileMData)
//		if marshalError != nil {
//			log.Println("Error Encoding metadata file to cache", marshalError)
//			w.WriteHeader(400)
//			return
//		}
//		cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", fileMDataJSON)
//
//	}
//	elapsed := time.Since(start)
//	log.Println("Length of Output Data ", len(data), " processed in: ", elapsed)
//
//	// Get the metadata for this request to put into the return header.
//	fileMetaDataJSON, metaInCache := cache.GetDataFromCache(cacheFileName+"meta", "outputFiles/")
//	if !metaInCache {
//		log.Println("Error reading the metadata file from cache")
//		w.WriteHeader(400)
//		return
//	}
//	var fileMDataCache sds.FileMetaData
//	marshalError := json.Unmarshal(fileMetaDataJSON, &fileMDataCache)
//	if marshalError != nil {
//		log.Println("Error Decoding metadata file from cache", marshalError)
//		w.WriteHeader(400)
//		return
//	}
//	// Create a Return header with some metadata in it.
//	outxsizeStr := strconv.Itoa(fileMDataCache.Outxsize)
//	outysizeStr := strconv.Itoa(fileMDataCache.Outysize)
//	outzsizeStr := strconv.Itoa(fileMDataCache.Outzsize)
//
//	w.Header().Add("Access-Control-Allow-Origin", "*")
//	w.Header().Add("Access-Control-Expose-Headers", "outxsize,outysize,zmin,zmax,filexstart,filexdelta,fileystart,fileydelta,xmin,xmax,ymin,ymax")
//	w.Header().Add("outxsize", outxsizeStr)
//	w.Header().Add("outysize", outysizeStr)
//	w.Header().Add("outzsize", outzsizeStr)
//	w.Header().Add("zmin", fmt.Sprintf("%f", fileMDataCache.Zmin))
//	w.Header().Add("zmax", fmt.Sprintf("%f", fileMDataCache.Zmax))
//	w.Header().Add("filexstart", fmt.Sprintf("%f", fileMDataCache.Filexstart))
//	w.Header().Add("filexdelta", fmt.Sprintf("%f", fileMDataCache.Filexdelta))
//	w.Header().Add("fileystart", fmt.Sprintf("%f", fileMDataCache.Fileystart))
//	w.Header().Add("fileydelta", fmt.Sprintf("%f", fileMDataCache.Fileydelta))
//	w.Header().Add("xmin", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart)))
//	w.Header().Add("xmax", fmt.Sprintf("%f", fileMDataCache.Filexstart+fileMDataCache.Filexdelta*float64(fileMDataCache.Xstart+fileMDataCache.Xsize)))
//	w.Header().Add("ymin", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart)))
//	w.Header().Add("ymax", fmt.Sprintf("%f", fileMDataCache.Fileystart+fileMDataCache.Fileydelta*float64(fileMDataCache.Ystart+fileMDataCache.Ysize)))
//	w.WriteHeader(http.StatusOK)
//
//	w.Write(data)
//
//}
