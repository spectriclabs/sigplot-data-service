package config

type Config struct {
	Host                 string     `json:"host,omitempty"`
	Port                 int        `json:"port,omitempty"`
	Debug                bool       `json:"debug,omitempty"`
	ConfigFile           string     `json:"config_file,omitempty"`
	UseCache             bool       `json:"use_cache,omitempty"`
	CacheLocation        string     `json:"cache_location,omitempty"`
	CachePollingInterval int        `json:"cache_polling_interval,omitempty"`
	CacheMaxBytes        int64      `json:"cache_max_bytes,omitempty"`
	MaxBytesZminZmax     int        `json:"max_bytes_zmin_zmax,omitempty"`
	LocationDetails      []Location `json:"location_details,omitempty"`
}

type Location struct {
	LocationName   string `json:"location_name"`
	LocationType   string `json:"location_type"`
	Path           string `json:"path,omitempty"`
	MinioBucket    string `json:"minio_bucket,omitempty"`
	Location       string `json:"location,omitempty"`
	MinioAccessKey string `json:"minio_access_key,omitempty"`
	MinioSecretKey string `json:"minio_secret_key,omitempty"`
}
