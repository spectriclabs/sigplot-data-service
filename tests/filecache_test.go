package main

import (
	"fmt"
	"testing"

	"github.com/spectriclabs/sigplot-data-service/internal/cache"
)

func TestUrlToCacheFileName(t *testing.T) {
	expected := []struct {
		InputURL            string
		OutputCacheFileName string
	}{
		{
			InputURL:            "TestDir/foo.tmp?x1=0&y1=0&x2=127&y2=127&outxsize=320&outysize=316&outfmt=RGBA&colormap=RampColormap&cxmode=Re&transform=first",
			OutputCacheFileName: "TestDirfootmp_x10y10x2127y2127outxsize320outysize316outfmtRGBAcolormapRampColormapcxmodeRetransformfirst",
		},
		{
			InputURL:            "TestDir/foo.tmp",
			OutputCacheFileName: "TestDirfootmp",
		},
	}

	for _, exp := range expected {
		result := cache.UrlToCacheFileName(exp.InputURL)
		fmt.Println(result)
		if result != exp.OutputCacheFileName {
			t.Errorf(
				"UrlToCacheFileName(%s) returned %s instead of %s",
				exp.InputURL,
				result,
				exp.OutputCacheFileName,
			)
		}
	}
}
