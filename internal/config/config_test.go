package config

import (
	"encoding/json"
	"testing"
)

func TestConfigUnmarshalValid(t *testing.T) {
	raw := `{
		"host": "localhost",
		"port": 8080,
		"debug": true,
		"config_file": "/etc/sds.json",
		"use_cache": true,
		"cache_location": "/tmp/cache",
		"cache_polling_interval": 60,
		"cache_max_bytes": 1048576,
		"max_bytes_zmin_zmax": 4096,
		"location_details": [
			{
				"location_name": "local",
				"location_type": "filesystem",
				"path": "/data",
				"minio_bucket": "mybucket",
				"location": "minio.example.com",
				"minio_access_key": "access",
				"minio_secret_key": "secret",
				"minio_use_ssl": true
			}
		]
	}`

	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
	if cfg.CacheLocation != "/tmp/cache" {
		t.Errorf("CacheLocation = %q, want %q", cfg.CacheLocation, "/tmp/cache")
	}
	if cfg.CachePollingInterval != 60 {
		t.Errorf("CachePollingInterval = %d, want %d", cfg.CachePollingInterval, 60)
	}
	if cfg.CacheMaxBytes != 1048576 {
		t.Errorf("CacheMaxBytes = %d, want %d", cfg.CacheMaxBytes, int64(1048576))
	}
	if cfg.MaxBytesZminZmax != 4096 {
		t.Errorf("MaxBytesZminZmax = %d, want %d", cfg.MaxBytesZminZmax, 4096)
	}
	if len(cfg.LocationDetails) != 1 {
		t.Fatalf("len(LocationDetails) = %d, want 1", len(cfg.LocationDetails))
	}

	loc := cfg.LocationDetails[0]
	if loc.LocationName != "local" {
		t.Errorf("LocationName = %q, want %q", loc.LocationName, "local")
	}
	if loc.LocationType != "filesystem" {
		t.Errorf("LocationType = %q, want %q", loc.LocationType, "filesystem")
	}
	if loc.Path != "/data" {
		t.Errorf("Path = %q, want %q", loc.Path, "/data")
	}
	if loc.MinioBucket != "mybucket" {
		t.Errorf("MinioBucket = %q, want %q", loc.MinioBucket, "mybucket")
	}
	if loc.MinioAccessKey != "access" {
		t.Errorf("MinioAccessKey = %q, want %q", loc.MinioAccessKey, "access")
	}
	if loc.MinioSecretKey != "secret" {
		t.Errorf("MinioSecretKey = %q, want %q", loc.MinioSecretKey, "secret")
	}
	if !loc.MinioUseSSL {
		t.Error("MinioUseSSL = false, want true")
	}
}

func TestConfigUnmarshalMissingOptionalFields(t *testing.T) {
	raw := `{}`
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if cfg.Host != "" {
		t.Errorf("Host = %q, want empty", cfg.Host)
	}
	if cfg.Port != 0 {
		t.Errorf("Port = %d, want 0", cfg.Port)
	}
	if cfg.Debug {
		t.Error("Debug = true, want false")
	}
	if cfg.LocationDetails != nil {
		t.Errorf("LocationDetails = %v, want nil", cfg.LocationDetails)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	original := Config{
		Host:                 "0.0.0.0",
		Port:                 9090,
		Debug:                true,
		ConfigFile:           "config.json",
		UseCache:             true,
		CacheLocation:        "/var/cache",
		CachePollingInterval: 30,
		CacheMaxBytes:        2048,
		MaxBytesZminZmax:     512,
		LocationDetails: []Location{
			{
				LocationName:   "minio1",
				LocationType:   "minio",
				MinioBucket:    "bucket1",
				Location:       "minio.local:9000",
				MinioAccessKey: "key",
				MinioSecretKey: "secret",
				MinioUseSSL:    true,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Host != original.Host {
		t.Errorf("Host = %q, want %q", decoded.Host, original.Host)
	}
	if decoded.Port != original.Port {
		t.Errorf("Port = %d, want %d", decoded.Port, original.Port)
	}
	if decoded.CacheMaxBytes != original.CacheMaxBytes {
		t.Errorf("CacheMaxBytes = %d, want %d", decoded.CacheMaxBytes, original.CacheMaxBytes)
	}
	if len(decoded.LocationDetails) != 1 {
		t.Fatalf("len(LocationDetails) = %d, want 1", len(decoded.LocationDetails))
	}
	if decoded.LocationDetails[0].MinioUseSSL != true {
		t.Error("MinioUseSSL round-trip failed")
	}
	if decoded.LocationDetails[0].LocationName != "minio1" {
		t.Errorf("LocationName = %q, want %q", decoded.LocationDetails[0].LocationName, "minio1")
	}
}

func TestLocationMinioUseSSLOmitEmpty(t *testing.T) {
	loc := Location{
		LocationName: "test",
		LocationType: "fs",
	}
	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw := string(data)
	// MinioUseSSL is false (zero value) and has omitempty, so it should not appear
	if containsString(raw, "minio_use_ssl") {
		t.Errorf("expected minio_use_ssl to be omitted, got %s", raw)
	}

	loc.MinioUseSSL = true
	data, err = json.Marshal(loc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	raw = string(data)
	if !containsString(raw, "minio_use_ssl") {
		t.Errorf("expected minio_use_ssl to be present, got %s", raw)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
