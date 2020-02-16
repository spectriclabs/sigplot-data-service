package main

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/fasthttp/router"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"os"
	"runtime/pprof"
	"sigplot-data-service/internal/cache"
	"sigplot-data-service/internal/config"
	"sigplot-data-service/internal/datasource"
	"sigplot-data-service/internal/handlers"
)

// RunProfile kicks off a profiling job and writes
// the output to the requested profileFile (set by --cpuprofile)
func RunProfile(
	configuration config.Configuration,
	logger *zap.Logger,
	profileFile string,
) {
	f, err := os.Create(profileFile)
	if err != nil {
		logger.Fatal(
			"An error creating a file occurred",
			zap.Error(err),
			zap.String("profile_file", profileFile),
		)
	}

	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	req := handlers.ProcessRequest{
		FileFormat:     "SI",
		FileDataOffset: 0,
		FileXSize:      8192,
		Xstart:         0,
		Ystart:         0,
		Xsize:          8192,
		Ysize:          20000,
		Outxsize:       300,
		Outysize:       700,
		Transform:      "mean",
		Cxmode:         "Lo",
		OutputFmt:      "RGBA",
		Zmin:           -20000,
		Zmax:           8192,
		Zset:           true,
		CxmodeSet:      true,
		ColorMap:       "RampColormap",
	}

	start := time.Now()
	reader, _, succeed := datasource.OpenDataSource(
		configuration,
		logger,
		"TestDir",
		"profile_test.tmp",
	)
	if !succeed {
		logger.Fatal(
			"An error has occurred reading the data source",
			zap.String("filename", "profile_test.tmp"),
			zap.Bool("succeed", succeed),
		)
	}
	data := handlers.HandleProcessRequest(reader, req)
	elapsed := time.Since(start)
	logger.Info(
		"Successfully processed request",
		zap.Int("data_len", len(data)),
		zap.String("filename", "profile_test.tmp"),
		zap.Duration("elapsed", elapsed),
	)
}

// SetupLogger sets up the zap.Logger structured logger.
func SetupLogger(debug bool) *zap.Logger {
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}
	logger, logErr := zap.Config{
		Encoding:    "json",
		Level:       zap.NewAtomicLevelAt(level),
		OutputPaths: []string{"stdout"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:  "message",
			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}.Build()
	if logErr != nil {
		log.Fatalf("Couldn't setup logger: %v", logErr)
	}

	return logger
}

// LoadConfig reads the configuration file and unmarshals
// it into a config.Configuration struct.
func LoadConfig(logger *zap.Logger, configFile string) *config.Configuration {
	// Load Configuration File
	configuration := &config.Configuration{}
	viper.SetConfigName(configFile)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal(
			"Error reading config file",
			zap.String("config_file", configFile),
			zap.Error(err),
		)
	}

	err = viper.Unmarshal(configuration)
	if err != nil {
		logger.Fatal(
			"Error decoding config file",
			zap.String("config_file", configFile),
			zap.Error(err),
		)
	}
	return configuration
}

// SetupCache kicks off two cache monitors for minio and localFiles.
// These two monitors will execute on the interval set in the config file.
func SetupCache(configuration config.Configuration, logger *zap.Logger) {
	// Launch a seperate routine to monitor the cache size
	outputPath := filepath.Join(configuration.CacheLocation, "outputFiles/")
	minioPath := filepath.Join(configuration.CacheLocation, "miniocache/")
	go cache.CheckCache(
		outputPath,
		configuration.CheckCacheEvery,
		configuration.CacheMaxBytes,
	)
	go cache.CheckCache(
		minioPath,
		configuration.CheckCacheEvery,
		configuration.CacheMaxBytes,
	)
}

// SetupFlags sets up the minimal set of CLI flags.
//
// * cpuprofile - location of file to write profile output
//                (default: "")
// * config - location of SDS config file
//            (default: ./sds_config.yml)
// * debug - whether or not to enable debug logging
//           (default: false)
//
// Note: we're using the pflags module, which is a
// drop-in replacement for the built-in flags module.
func SetupFlags() (*string, *string, *bool) {
	cpuprofile := flag.String(
		"cpuprofile",
		"",
		"Profile SDS and write to file",
	)
	configFile := flag.String(
		"config",
		"./sdsConfig.json",
		"Location of SigPlot Data Service configuration file",
	)
	debugFlag := flag.Bool(
		"debug",
		false,
		"Whether or not to enable debug logging",
	)
	flag.Parse()

	return cpuprofile, configFile, debugFlag
}

// StartServer kicks off the server on the port provided
// in the configuration file (config.Configuration.Port)
// and binds the handlers.ServeHTTP handler to
// /sds/:location/:filename.
func StartServer(configuration config.Configuration, logger *zap.Logger) {
	port := fmt.Sprintf(":%d", configuration.Port)
	logger.Info("Starting server", zap.Int("port", configuration.Port))
	r := router.New()
	r.GET("/sds/:location/:filename", handlers.ServeHTTP(logger, configuration))
	logger.Fatal(
		"Stopping server due to error",
		zap.Error(fasthttp.ListenAndServe(port, r.Handler)),
	)
}

func main() {
	cpuprofile, configFile, debugFlag := SetupFlags()

	logger := SetupLogger(*debugFlag)
	defer logger.Sync()

	configuration := LoadConfig(logger, *configFile)

	// Used to profile speed
	if *cpuprofile != "" {
		RunProfile(*configuration, logger, *cpuprofile)
	}

	SetupCache(*configuration, logger)
	StartServer(*configuration, logger)
}
