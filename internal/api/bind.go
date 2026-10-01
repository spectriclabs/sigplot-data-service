package api

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/spectriclabs/sigplot-data-service/internal/sds"
)

func bindRDSRequest(c echo.Context, request *sds.RdsRequest) error {
	request.FileName = c.Param("*")

	if err := bindIntParam(c, "tileXsize", &request.TileXSize); err != nil {
		return err
	}
	if err := bindIntParam(c, "tileYsize", &request.TileYSize); err != nil {
		return err
	}
	if err := bindIntParam(c, "decXMode", &request.DecXMode); err != nil {
		return err
	}
	if err := bindIntParam(c, "decYMode", &request.DecYMode); err != nil {
		return err
	}
	if err := bindIntParam(c, "tileX", &request.TileX); err != nil {
		return err
	}
	if err := bindIntParam(c, "tileY", &request.TileY); err != nil {
		return err
	}
	if err := bindIntParam(c, "outxsize", &request.Outxsize); err != nil {
		return err
	}
	if err := bindIntParam(c, "outysize", &request.Outysize); err != nil {
		return err
	}
	if err := bindIntParam(c, "outzsize", &request.Outzsize); err != nil {
		return err
	}
	if err := bindIntParam(c, "x1", &request.X1); err != nil {
		return err
	}
	if err := bindIntParam(c, "x2", &request.X2); err != nil {
		return err
	}
	if err := bindIntParam(c, "y1", &request.Y1); err != nil {
		return err
	}
	if err := bindIntParam(c, "y2", &request.Y2); err != nil {
		return err
	}

	if c.Request().URL.RawQuery != "" {
		query := c.QueryParams()
		if err := bindIntQuery(query, "subsize", &request.Subsize); err != nil {
			return err
		}
		var err error
		if request.ZminSet, err = bindFloatQuery(query, "zmin", &request.Zmin); err != nil {
			return err
		}
		if request.ZmaxSet, err = bindFloatQuery(query, "zmax", &request.Zmax); err != nil {
			return err
		}
		bindStringQuery(query, "transform", &request.Transform)
		bindStringQuery(query, "colormap", &request.ColorMap)
		bindStringQuery(query, "cxmode", &request.Cxmode)
		bindStringQuery(query, "outfmt", &request.OutputFmt)
	}

	return nil
}

func bindIntParam(c echo.Context, name string, target *int) error {
	value := c.Param(name)
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("invalid path parameter %s=%q: %w", name, value, err)
	}
	*target = parsed
	return nil
}

func bindIntQuery(query url.Values, name string, target *int) error {
	values, ok := query[name]
	if !ok || len(values) == 0 {
		return nil
	}
	parsed, err := strconv.Atoi(values[0])
	if err != nil {
		return fmt.Errorf("invalid query parameter %s=%q: %w", name, values[0], err)
	}
	*target = parsed
	return nil
}

func bindFloatQuery(query url.Values, name string, target *float64) (bool, error) {
	values, ok := query[name]
	if !ok || len(values) == 0 {
		return false, nil
	}
	parsed, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return false, fmt.Errorf("invalid query parameter %s=%q: %w", name, values[0], err)
	}
	*target = parsed
	return true, nil
}

func bindStringQuery(query url.Values, name string, target *string) {
	if values, ok := query[name]; ok && len(values) > 0 {
		*target = values[0]
	}
}
