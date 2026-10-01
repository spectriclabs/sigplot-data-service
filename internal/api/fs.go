package api

import (
	"encoding/binary"
	"fmt"
	"github.com/labstack/echo/v4"
	minio "github.com/minio/minio-go/v7"
	"github.com/spectriclabs/sigplot-data-service/internal/bluefile"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
	"github.com/spectriclabs/sigplot-data-service/internal/sds"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type safeFileLocation struct {
	LocationName string `json:"location_name"`
	LocationType string `json:"location_type"`
}

func (a *API) GetBluefileHeader(c echo.Context) error {
	filePath := c.Param("*")
	locationName := c.Param("location")
	reader, err := sds.OpenDataSource(a.Cfg, a.Cache, locationName, filePath)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	if strings.Contains(filePath, ".tmp") || strings.Contains(filePath, ".prm") {
		c.Logger().Infof("Opening %s for file header mode", filePath)

		var bluefileheader bluefile.BlueHeader
		err := binary.Read(reader, binary.LittleEndian, &bluefileheader)
		if err != nil {
			return err
		}

		blueShort := bluefile.BlueHeaderShortenedFields{
			Version:   string(bluefileheader.Version[:]),
			HeadRep:   string(bluefileheader.HeadRep[:]),
			DataRep:   string(bluefileheader.DataRep[:]),
			Detached:  bluefileheader.Detached,
			Protected: bluefileheader.Protected,
			Pipe:      bluefileheader.Pipe,
			ExtStart:  bluefileheader.ExtStart,
			DataStart: bluefileheader.DataStart,
			DataSize:  bluefileheader.DataSize,
			FileType:  bluefileheader.FileType,
			Format:    string(bluefileheader.Format[:]),
			Flagmask:  bluefileheader.Flagmask,
			Timecode:  bluefileheader.Timecode,
			Xstart:    bluefileheader.Xstart,
			Xdelta:    bluefileheader.Xdelta,
			Xunits:    bluefileheader.Xunits,
			Subsize:   bluefileheader.Subsize,
			Ystart:    bluefileheader.Ystart,
			Ydelta:    bluefileheader.Ydelta,
			Yunits:    bluefileheader.Yunits,
		}

		blueShort.Spa = bluefile.SPA[string(blueShort.Format[0])]
		blueShort.Bps = bluefile.BPS[string(blueShort.Format[1])]
		blueShort.Bpa = float64(blueShort.Spa) * blueShort.Bps
		if blueShort.FileType == 1000 {
			blueShort.Ape = 1
		} else {
			blueShort.Ape = int(blueShort.Subsize)
		}

		blueShort.Bpe = float64(blueShort.Ape) * blueShort.Bpa
		log.Println("Computing Size", blueShort.DataSize, blueShort.Bpa, blueShort.Ape)
		blueShort.Size = int(blueShort.DataSize / (blueShort.Bpa * float64(blueShort.Ape)))

		return c.JSON(http.StatusOK, blueShort)
	} else {
		err := fmt.Errorf("can only return headers for bluefiles (.tmp or .prm)")
		return c.String(http.StatusBadRequest, err.Error())
	}
}

func (a *API) GetFileContents(c echo.Context, locationName string, filePath string) error {
	reader, err := sds.OpenDataSource(a.Cfg, a.Cache, locationName, filePath)
	if err != nil {
		c.Logger().Error(err)
		if sds.IsInvalidPath(err) {
			return c.String(http.StatusBadRequest, err.Error())
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}

	return c.Stream(http.StatusOK, sds.ContentTypeForPath(filePath), reader)
}

func (a *API) GetDirectoryContents(c echo.Context, directoryPath string) error {
	files, err := ioutil.ReadDir(directoryPath)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusBadRequest, err.Error())
	}
	filelist := make([]sds.File, len(files))

	for i, file := range files {
		filelist[i].Filename = file.Name()
		if file.IsDir() {
			filelist[i].Type = "directory"
		} else {
			filelist[i].Type = "file"
		}
	}

	return c.JSON(http.StatusOK, filelist)
}

func (a *API) GetFileLocations(c echo.Context) error {
	locations := make([]safeFileLocation, 0, len(a.Cfg.LocationDetails))
	for _, location := range a.Cfg.LocationDetails {
		locations = append(locations, safeFileLocation{
			LocationName: location.LocationName,
			LocationType: location.LocationType,
		})
	}
	return c.JSON(200, locations)
}

func (a *API) GetFileOrDirectory(c echo.Context) error {
	filePath := c.Param("*")
	locationName := c.Param("location")

	// Find the configured location corresponding
	// to `locationName`
	var currentLocation config.Location
	for i := range a.Cfg.LocationDetails {
		if a.Cfg.LocationDetails[i].LocationName == locationName {
			c.Logger().Debugf(
				"Found location %s in configured locations",
				locationName,
			)
			currentLocation = a.Cfg.LocationDetails[i]
		}
	}

	if currentLocation.LocationName != locationName {
		err := fmt.Errorf("couldn't find location %s", locationName)
		return c.String(http.StatusBadRequest, err.Error())
	}

	switch currentLocation.LocationType {
	case "localFile":
		joinedFilePath, err := sds.ResolvePath(currentLocation.Path, filePath)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		// Make sure the joined path exists
		fi, err := os.Stat(joinedFilePath)
		if err != nil {
			err := fmt.Errorf("error reading path %s: %s", joinedFilePath, err)
			return c.String(http.StatusBadRequest, err.Error())
		}

		// If the URL is to a file, use raw mode to return file contents
		mode := fi.Mode()
		if mode.IsRegular() {
			c.Logger().Info("Path is a file; returning contents in raw mode")
			return a.GetFileContents(c, locationName, filePath)
		} else {
			// Otherwise, it is likely a directory
			c.Logger().Info("Path is a directory; returning directory listing")
			return a.GetDirectoryContents(c, joinedFilePath)
		}

	case "minio":
		objectKey, err := sds.ResolveObjectKey(currentLocation.Path, filePath)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		prefix, err := sds.ObjectListPrefix(currentLocation.Path, filePath)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		minioClient, err := sds.MinioClientForLocation(currentLocation)
		if err != nil {
			log.Println("Error establishing connection to MinIO", err)
			return c.String(http.StatusInternalServerError, err.Error())
		}

		ctx := c.Request().Context()

		// If filePath is non-empty, try to get it as a file first
		if filePath != "" && !strings.HasSuffix(filePath, "/") {
			objectFd, err := minioClient.GetObject(ctx, currentLocation.MinioBucket, objectKey, minio.GetObjectOptions{})
			if err == nil {
				_, statErr := objectFd.Stat()
				if statErr == nil {
					defer objectFd.Close()
					return c.Stream(http.StatusOK, sds.ContentTypeForPath(filePath), objectFd)
				}
				objectFd.Close()
			}
		}
		// List directory contents
		objectCh := minioClient.ListObjects(ctx, currentLocation.MinioBucket, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: false,
		})

		filelist := []sds.File{}
		for object := range objectCh {
			if object.Err != nil {
				log.Println("Error listing MinIO objects", object.Err)
				return c.String(http.StatusInternalServerError, object.Err.Error())
			}
			if object.Key == prefix {
				continue
			}
			var f sds.File
			name := path.Base(strings.TrimSuffix(object.Key, "/"))
			f.Filename = name
			if strings.HasSuffix(object.Key, "/") {
				f.Type = "directory"
			} else {
				f.Type = "file"
			}
			filelist = append(filelist, f)
		}

		return c.JSON(http.StatusOK, filelist)

	default:
		err := fmt.Errorf("listing files not supported for location type: %s", currentLocation.LocationType)
		return c.String(http.StatusBadRequest, err.Error())
	}
}
