package sds

import (
	"bytes"
	"math"
	"os"
	"sync"
	"testing"
)

func TestComputeRequestSizes(t *testing.T) {
	tests := []struct {
		name                   string
		x1, x2, y1, y2         int
		wantXstart, wantYstart int
		wantXsize, wantYsize   int
	}{
		{
			name: "normal order",
			x1:   10, x2: 50, y1: 5, y2: 25,
			wantXstart: 10, wantYstart: 5,
			wantXsize: 40, wantYsize: 20,
		},
		{
			name: "reversed order",
			x1:   50, x2: 10, y1: 25, y2: 5,
			wantXstart: 10, wantYstart: 5,
			wantXsize: 40, wantYsize: 20,
		},
		{
			name: "zero range",
			x1:   0, x2: 0, y1: 0, y2: 0,
			wantXstart: 0, wantYstart: 0,
			wantXsize: 0, wantYsize: 0,
		},
		{
			name: "single pixel",
			x1:   5, x2: 6, y1: 3, y2: 4,
			wantXstart: 5, wantYstart: 3,
			wantXsize: 1, wantYsize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RdsRequest{X1: tt.x1, X2: tt.x2, Y1: tt.y1, Y2: tt.y2}
			req.ComputeRequestSizes()

			if req.Xstart != tt.wantXstart {
				t.Errorf("Xstart = %d, want %d", req.Xstart, tt.wantXstart)
			}
			if req.Ystart != tt.wantYstart {
				t.Errorf("Ystart = %d, want %d", req.Ystart, tt.wantYstart)
			}
			if req.Xsize != tt.wantXsize {
				t.Errorf("Xsize = %d, want %d", req.Xsize, tt.wantXsize)
			}
			if req.Ysize != tt.wantYsize {
				t.Errorf("Ysize = %d, want %d", req.Ysize, tt.wantYsize)
			}
		})
	}
}

func TestComputeYSize(t *testing.T) {
	tests := []struct {
		name         string
		fileDataSize float64
		fileFormat   string
		fileXSize    int
		wantYSize    int
	}{
		{
			name:         "SF 400 bytes, xsize 10",
			fileDataSize: 400,
			fileFormat:   "SF",
			fileXSize:    10,
			wantYSize:    10, // 400/4 = 100 atoms, 100/10 = 10
		},
		{
			name:         "SD 800 bytes, xsize 10",
			fileDataSize: 800,
			fileFormat:   "SD",
			fileXSize:    10,
			wantYSize:    10, // 800/8 = 100 atoms, 100/10 = 10
		},
		{
			name:         "SI 200 bytes, xsize 10",
			fileDataSize: 200,
			fileFormat:   "SI",
			fileXSize:    10,
			wantYSize:    10, // 200/2 = 100 atoms, 100/10 = 10
		},
		{
			name:         "CF 800 bytes, xsize 10",
			fileDataSize: 800,
			fileFormat:   "CF",
			fileXSize:    10,
			wantYSize:    10, // 800/4 = 200 atoms, 200/10 = 20, complex /2 = 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RdsRequest{
				FileDataSize: tt.fileDataSize,
				FileFormat:   tt.fileFormat,
				FileXSize:    tt.fileXSize,
			}
			req.ComputeYSize()
			if req.FileYSize != tt.wantYSize {
				t.Errorf("FileYSize = %d, want %d", req.FileYSize, tt.wantYSize)
			}
		})
	}
}

func TestComputeTileSizes(t *testing.T) {
	tests := []struct {
		name                   string
		tileX, tileY           int
		tileXSize, tileYSize   int
		decXMode, decYMode     int
		wantXstart, wantYstart int
		wantXsize, wantYsize   int
		wantOutx, wantOuty     int
	}{
		{
			name:  "tile 0,0 dec 1",
			tileX: 0, tileY: 0,
			tileXSize: 256, tileYSize: 256,
			decXMode: 1, decYMode: 1,
			wantXstart: 0, wantYstart: 0,
			wantXsize: 256, wantYsize: 256,
			wantOutx: 256, wantOuty: 256,
		},
		{
			name:  "tile 1,2 dec 1",
			tileX: 1, tileY: 2,
			tileXSize: 128, tileYSize: 128,
			decXMode: 1, decYMode: 1,
			wantXstart: 128, wantYstart: 256,
			wantXsize: 128, wantYsize: 128,
			wantOutx: 128, wantOuty: 128,
		},
		{
			name:  "tile 0,0 dec 3 (4x)",
			tileX: 0, tileY: 0,
			tileXSize: 100, tileYSize: 100,
			decXMode: 3, decYMode: 3,
			wantXstart: 0, wantYstart: 0,
			wantXsize: 400, wantYsize: 400,
			wantOutx: 100, wantOuty: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RdsRequest{
				TileX:     tt.tileX,
				TileY:     tt.tileY,
				TileXSize: tt.tileXSize,
				TileYSize: tt.tileYSize,
				DecXMode:  tt.decXMode,
				DecYMode:  tt.decYMode,
			}
			req.ComputeTileSizes()

			if req.Xstart != tt.wantXstart {
				t.Errorf("Xstart = %d, want %d", req.Xstart, tt.wantXstart)
			}
			if req.Ystart != tt.wantYstart {
				t.Errorf("Ystart = %d, want %d", req.Ystart, tt.wantYstart)
			}
			if req.Xsize != tt.wantXsize {
				t.Errorf("Xsize = %d, want %d", req.Xsize, tt.wantXsize)
			}
			if req.Ysize != tt.wantYsize {
				t.Errorf("Ysize = %d, want %d", req.Ysize, tt.wantYsize)
			}
			if req.Outxsize != tt.wantOutx {
				t.Errorf("Outxsize = %d, want %d", req.Outxsize, tt.wantOutx)
			}
			if req.Outysize != tt.wantOuty {
				t.Errorf("Outysize = %d, want %d", req.Outysize, tt.wantOuty)
			}
		})
	}
}

func TestDecimationLookup(t *testing.T) {
	expected := map[int]int{
		1: 1, 2: 2, 3: 4, 4: 8, 5: 16,
		6: 32, 7: 64, 8: 128, 9: 256, 10: 512,
	}

	for k, want := range expected {
		got, ok := DecimationLookup[k]
		if !ok {
			t.Errorf("DecimationLookup[%d] missing", k)
			continue
		}
		if got != want {
			t.Errorf("DecimationLookup[%d] = %d, want %d", k, got, want)
		}
	}

	// Verify powers of 2 pattern
	for k := 1; k <= 10; k++ {
		want := int(math.Pow(2, float64(k-1)))
		if DecimationLookup[k] != want {
			t.Errorf("DecimationLookup[%d] = %d, want 2^%d = %d", k, DecimationLookup[k], k-1, want)
		}
	}
}

func TestApplyBindDefaults(t *testing.T) {
	tests := []struct {
		name           string
		req            RdsRequest
		wantSubsize    int
		wantSubsizeSet bool
		wantCxmode     string
		wantCxmodeSet  bool
		wantZset       bool
		wantZmin       float64
		wantZmax       float64
		wantTransform  string
		wantColorMap   string
	}{
		{
			name:          "defaults",
			req:           RdsRequest{},
			wantSubsize:   1,
			wantCxmode:    "Re",
			wantTransform: "first",
			wantColorMap:  "RampColormap",
		},
		{
			name:           "subsize cxmode transform colormap",
			req:            RdsRequest{Subsize: 8, Cxmode: "Im", Transform: "mean", ColorMap: "Greyscale"},
			wantSubsize:    8,
			wantSubsizeSet: true,
			wantCxmode:     "Im",
			wantCxmodeSet:  true,
			wantTransform:  "mean",
			wantColorMap:   "Greyscale",
		},
		{
			name:          "only zmin is not zset",
			req:           RdsRequest{Zmin: 1.25, ZminSet: true},
			wantSubsize:   1,
			wantCxmode:    "Re",
			wantZmin:      1.25,
			wantTransform: "first",
			wantColorMap:  "RampColormap",
		},
		{
			name:          "only zmax is not zset",
			req:           RdsRequest{Zmax: 2.5, ZmaxSet: true},
			wantSubsize:   1,
			wantCxmode:    "Re",
			wantZmax:      2.5,
			wantTransform: "first",
			wantColorMap:  "RampColormap",
		},
		{
			name:          "both zmin zmax are zset",
			req:           RdsRequest{Zmin: -1, Zmax: 3, ZminSet: true, ZmaxSet: true},
			wantSubsize:   1,
			wantCxmode:    "Re",
			wantZset:      true,
			wantZmin:      -1,
			wantZmax:      3,
			wantTransform: "first",
			wantColorMap:  "RampColormap",
		},
		{
			name:          "explicit zero z range is zset",
			req:           RdsRequest{ZminSet: true, ZmaxSet: true},
			wantSubsize:   1,
			wantCxmode:    "Re",
			wantZset:      true,
			wantTransform: "first",
			wantColorMap:  "RampColormap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			req.ApplyBindDefaults()
			if req.Subsize != tt.wantSubsize || req.SubsizeSet != tt.wantSubsizeSet {
				t.Fatalf("subsize = %d/%v, want %d/%v", req.Subsize, req.SubsizeSet, tt.wantSubsize, tt.wantSubsizeSet)
			}
			if req.Cxmode != tt.wantCxmode || req.CxmodeSet != tt.wantCxmodeSet {
				t.Fatalf("cxmode = %q/%v, want %q/%v", req.Cxmode, req.CxmodeSet, tt.wantCxmode, tt.wantCxmodeSet)
			}
			if req.Zset != tt.wantZset || req.Zmin != tt.wantZmin || req.Zmax != tt.wantZmax {
				t.Fatalf("z = %v/%v/%v, want %v/%v/%v", req.Zset, req.Zmin, req.Zmax, tt.wantZset, tt.wantZmin, tt.wantZmax)
			}
			if req.Transform != tt.wantTransform {
				t.Fatalf("transform = %q, want %q", req.Transform, tt.wantTransform)
			}
			if req.ColorMap != tt.wantColorMap {
				t.Fatalf("colormap = %q, want %q", req.ColorMap, tt.wantColorMap)
			}
		})
	}
}

func TestFindZminMaxConcurrent(t *testing.T) {
	ResetZminzmaxFileMap()
	fileData, err := os.ReadFile("../../tests/data/mydata_SB_60_60.tmp")
	if err != nil {
		t.Fatal(err)
	}

	const workers = 16
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := RdsRequest{
				FileName:    "mydata_SB_60_60.tmp",
				Cxmode:      "Re",
				Transform:   "first",
				OutputFmt:   "SD",
				Reader:      bytes.NewReader(fileData),
				ReaderMutex: &sync.Mutex{},
			}
			req.ProcessBlueFileHeader()
			req.ComputeYSize()
			req.FindZminMax(1000000)
			if req.Zmin != 0 || req.Zmax != 10 {
				t.Errorf("zmin/zmax = %v/%v, want 0/10", req.Zmin, req.Zmax)
			}
		}()
	}
	wg.Wait()
}

func TestProcessLineRequestYCutReadFailure(t *testing.T) {
	req := RdsRequest{
		FileFormat:     "SB",
		FileXSize:      60,
		FileDataOffset: 0,
		Xstart:         0,
		Ystart:         0,
		Ysize:          2,
		Outxsize:       2,
		Outzsize:       2,
		Zmin:           0,
		Zmax:           10,
		Reader:         bytes.NewReader([]byte{1}),
		ReaderMutex:    &sync.Mutex{},
	}
	if _, err := ProcessLineRequest(req, "rdsycut"); err == nil {
		t.Fatal("ProcessLineRequest returned nil error for short y-cut read")
	}
}

func TestApplyBindDefaultsOutputFmt(t *testing.T) {
	for in, want := range map[string]string{"": "RGBA", "SD": "SD", "RGBA": "RGBA"} {
		req := RdsRequest{OutputFmt: in}
		req.ApplyBindDefaults()
		if req.OutputFmt != want {
			t.Errorf("OutputFmt %q -> %q, want %q", in, req.OutputFmt, want)
		}
	}
}
