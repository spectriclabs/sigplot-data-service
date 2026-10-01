package api

func (a *API) getCachedOutput(cacheFileName string) ([]byte, []byte, bool) {
	if !a.Cfg.UseCache {
		return nil, nil, false
	}

	data, err := a.Cache.GetDataFromCache(cacheFileName, "outputFiles/")
	if err != nil {
		return nil, nil, false
	}
	meta, err := a.Cache.GetDataFromCache(cacheFileName+"meta", "outputFiles/")
	if err != nil {
		return nil, nil, false
	}
	return data, meta, true
}

func (a *API) putCachedOutput(cacheFileName string, data []byte, meta []byte) error {
	if !a.Cfg.UseCache {
		return nil
	}
	if err := a.Cache.PutItemInCache(cacheFileName+"meta", "outputFiles/", meta); err != nil {
		return err
	}
	go a.Cache.PutItemInCache(cacheFileName, "outputFiles/", data)
	return nil
}
