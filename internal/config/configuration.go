package config

type Location struct {
	LocationName   string `mapstructure:"location_name"`
	LocationType   string `mapstructure:"location_type"`
	Path           string `mapstructure:"path"`
	MinioBucket    string `mapstructure:"minio_bucket"`
	Location       string `mapstructure:"location"`
	MinioAccessKey string `mapstructure:"minio_access_key"`
	MinioSecretKey string `mapstructure:"minio_secret_key"`
}

// Configuration Struct for Configuraion File
type Configuration struct {
	Port            int        `mapstructure:"port"`
	CacheLocation   string     `mapstructure:"cache_location"`
	Logfile         string     `mapstructure:"logfile"`
	CacheMaxBytes   int64      `mapstructure:"cache_max_bytes"`
	CheckCacheEvery int        `mapstructure:"check_cache_every"`
	LocationDetails []Location `mapstructure:"location_details"`
}

type FileMetaData struct {
	Outxsize   int     `json:"outxsize"`
	Outysize   int     `json:"outysize"`
	Zmin       float64 `json:"zmin"`
	Zmax       float64 `json:"zmax"`
	Filexstart float64 `json:"filexstart"`
	Filexdelta float64 `json:"filexdelta"`
	Fileystart float64 `json:"fileystart"`
	Fileydelta float64 `json:"fileydelta"`
}
