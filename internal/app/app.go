package app

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spectriclabs/sigplot-data-service/internal/api"
	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/spectriclabs/sigplot-data-service/internal/sds"
	"github.com/spectriclabs/sigplot-data-service/ui"
	"github.com/spf13/pflag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

func Run() {
	cfg := ParseCLI()
	cfg.LocationDetails = ParseSDSConfigFile(cfg.ConfigFile)

	sds.ZminzmaxFileMap = make(map[string]sds.Zminzmax)

	if cfg.UseCache {
		SetupCache(
			cfg.CacheLocation,
			cfg.CachePollingInterval,
			cfg.CacheMaxBytes,
		)
	}

	// Setup API
	sdsapi := api.NewSDSAPI(&cfg)

	// Setup HTTP server
	e := SetupServer(sdsapi)

	// Run server
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	go func() {
		if err := e.Start(address); err != nil {
			e.Logger.Info("Shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func ParseCLI() config.Config {
	cfg := config.Config{}
	pflag.StringVarP(&cfg.Host, "host", "i", "0.0.0.0", "Host where the server will run")
	pflag.IntVarP(&cfg.Port, "port", "p", 5055, "Port where the server will run")
	pflag.BoolVarP(&cfg.Debug, "debug", "d", false, "Whether or not to enable debug logging")
	pflag.StringVarP(&cfg.ConfigFile, "config", "c", "./sdsConfig.json", "Location of SDS config file")
	pflag.BoolVarP(&cfg.UseCache, "use-cache", "u", true, "Use SDS Cache. Can be disabled for certain cases like testing.")
	pflag.StringVarP(&cfg.CacheLocation, "cache-location", "C", "./sdscache/", "Where the cache will be stored")
	pflag.IntVarP(&cfg.CachePollingInterval, "cache-polling-interval", "P", 60, "How often to check the cache (in seconds)")
	pflag.Int64VarP(&cfg.CacheMaxBytes, "cache-max-bytes", "m", 100000000, "How large to allow the cache to be")
	pflag.IntVarP(&cfg.MaxBytesZminZmax, "max-bytes-zmin-zmax", "z", 1000000, "")
	pflag.Parse()

	return cfg
}

func SetupServer(api *api.API) *echo.Echo {
	e := echo.New()

	e.Debug = api.Cfg.Debug

	// Setup Middleware
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// File-specific routes
	e.GET("/sds/fs", api.GetFileLocations)
	e.GET("/sds/fs/:location/*", api.GetFileOrDirectory)
	e.GET("/sds/hdr/:location/*", api.GetBluefileHeader)

	// Data-service routes
	e.GET(
		"/sds/rdstile/:location/:tileXsize/:tileYsize/:decimationXMode/:decimationYMode/:tileX/:tileY/*",
		api.GetRDSTile,
	)
	// e.GET("/sds/rdsxcut/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", api.GetRDSXYCut)
	// e.GET("/sds/rdsycut/:x1/:y1/:x2/:y2/:outxsize/:outysize/:location/*", api.GetRDSXYCut)
	// e.GET("/sds/lds/:x1/:x2/:outxsize/:outzsize/:location/*", api.GetLDS)

	// Setup SigPlot Data Service UI route
	webappFS := http.FileServer(ui.GetFileSystem())
	e.GET("/ui/", echo.WrapHandler(http.StripPrefix("/ui/", webappFS)))

	// Add Prometheus as middleware for metrics gathering
	p := prometheus.NewPrometheus("sigplot_data_service", nil)
	p.Use(e)

	return e
}

// SetupCache will setup a cache directory and kick off cache
// checking goroutines
func SetupCache(cacheLocation string, cachePollingInterval int, cacheMaxBytes int64) {
	// Create directories for cache if they don't exist
	err := os.MkdirAll(cacheLocation, 0755)
	if err != nil {
		log.Println("Error Creating Cache File Directory: ", cacheLocation, err)
		return
	}
	outputFilesDir := filepath.Join(cacheLocation, "outputFiles/")
	err = os.MkdirAll(outputFilesDir, 0755)
	if err != nil {
		log.Println("Error Creating Cache File/outputFiles Directory ", cacheLocation, err)
		return
	}

	miniocache := filepath.Join(cacheLocation, "miniocache/")
	err = os.MkdirAll(miniocache, 0755)
	if err != nil {
		log.Println("Error Creating Cache File/miniocache Directory ", cacheLocation, err)
		return
	}

	// Launch a seperate routine to monitor the cache size
	outputPath := filepath.Join(cacheLocation, "outputFiles/")
	minioPath := filepath.Join(cacheLocation, "miniocache/")
	go cache.CheckCache(outputPath, cachePollingInterval, cacheMaxBytes)
	go cache.CheckCache(minioPath, cachePollingInterval, cacheMaxBytes)
}

func ParseSDSConfigFile(cfgfile string) []config.Location {
	body, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		panic(err)
	}

	var cfg *config.Config
	err = json.Unmarshal(body, &cfg)
	if err != nil {
		panic(err)
	}

	return cfg.LocationDetails
}
