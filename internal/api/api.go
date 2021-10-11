package api

import (
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
)

type API struct {
	Cfg   *config.Config
	Cache *cache.Cache
}

func NewSDSAPI(cfg *config.Config) *API {
	return &API{
		Cfg:   cfg,
		Cache: &cache.Cache{Location: cfg.CacheLocation},
	}
}
