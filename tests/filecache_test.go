package main

import (
	"fmt"
	"testing"

	"github.com/spectriclabs/sigplot-data-service/internal/cache"
)

func TestUrlToCacheFileName(t *testing.T) {
	expected := []struct {
		InputFileLocation   string
		InputFileName       string
		InputQuery          string
		OutputCacheFileName string
	}{
		{
			InputFileLocation:   "TestDir",
			InputFileName:       "foo.tmp",
			InputQuery:          "x1=0&y1=0&x2=127&y2=127&outxsize=320&outysize=316&outfmt=RGBA&colormap=RampColormap&cxmode=Re&transform=first",
			OutputCacheFileName: "TestDir_footmp_x10y10x2127y2127outxsize320outysize316outfmtRGBAcolormapRampColormapcxmodeRetransformfirst",
		},
		{
			InputFileLocation:   "TestDir",
			InputFileName:       "foo.tmp",
			InputQuery:          "",
			OutputCacheFileName: "TestDir_footmp_",
		},
	}

	for _, exp := range expected {
		result := cache.UrlToCacheFileName(
			exp.InputFileName,
			exp.InputQuery,
		)
		fmt.Println(result)
		if result != exp.OutputCacheFileName {
			t.Errorf(
				"UrlToCacheFileName(%s, %s, %s) returned %s instead of %s",
				exp.InputFileLocation,
				exp.InputFileName,
				exp.InputQuery,
				result,
				exp.OutputCacheFileName,
			)
		}
	}
}
