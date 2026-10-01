package image

import (
	"testing"
)

func TestGetColorControlPointsKnownMaps(t *testing.T) {
	tests := []struct {
		name     string
		colorMap string
		wantLen  int
	}{
		{"Greyscale", "Greyscale", 3},
		{"Ramp Colormap", "Ramp Colormap", 7},
		{"Color Wheel", "Color Wheel", 7},
		{"Spectrum", "Spectrum", 7},
		{"calewhite", "calewhite", 7},
		{"HotDesat", "HotDesat", 8},
		{"Sunset", "Sunset", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pts := GetColorControlPoints(tt.colorMap)
			if len(pts) != tt.wantLen {
				t.Errorf("GetColorControlPoints(%q) returned %d points, want %d", tt.colorMap, len(pts), tt.wantLen)
			}
		})
	}
}

func TestGetColorControlPointsUnknownFallsBackToRamp(t *testing.T) {
	unknown := GetColorControlPoints("nonexistent_colormap")
	ramp := GetColorControlPoints("Ramp Colormap")

	if len(unknown) != len(ramp) {
		t.Fatalf("unknown len = %d, Ramp len = %d", len(unknown), len(ramp))
	}
	for i := range ramp {
		if unknown[i] != ramp[i] {
			t.Errorf("unknown[%d] = %+v, want %+v", i, unknown[i], ramp[i])
		}
	}
}

func TestGetCachedPaletteNormalizesUnknownNames(t *testing.T) {
	paletteMu.Lock()
	paletteCache = make(map[paletteKey][]Pixel)
	paletteMu.Unlock()

	unknownA := GetCachedPalette("missing-a", 500)
	unknownB := GetCachedPalette("missing-b", 500)
	ramp := GetCachedPalette("Ramp Colormap", 500)

	if &unknownA[0] != &unknownB[0] || &unknownA[0] != &ramp[0] {
		t.Fatalf("unknown palettes were not normalized to cached Ramp palette")
	}
	paletteMu.RLock()
	defer paletteMu.RUnlock()
	if len(paletteCache) != 1 {
		t.Fatalf("palette cache entries = %d, want 1", len(paletteCache))
	}
}

func TestMakeColorPaletteLength(t *testing.T) {
	tests := []struct {
		name      string
		numColors int
	}{
		{"100 colors", 100},
		{"500 colors", 500},
		{"1000 colors", 1000},
	}

	controlColors := GetColorControlPoints("Greyscale")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := MakeColorPalette(controlColors, tt.numColors)
			if len(palette) != tt.numColors {
				t.Errorf("MakeColorPalette returned %d colors, want %d", len(palette), tt.numColors)
			}
		})
	}
}

func TestMakeColorPaletteTwoPointGradient(t *testing.T) {
	// Black to white gradient
	controlColors := []Pixel{
		{Position: 0, Red: 0, Green: 0, Blue: 0},
		{Position: 100, Red: 100, Green: 100, Blue: 100},
	}
	numColors := 100
	palette := MakeColorPalette(controlColors, numColors)

	// First color should be black (0,0,0)
	if palette[0].Red != 0 || palette[0].Green != 0 || palette[0].Blue != 0 {
		t.Errorf("first color = (%v,%v,%v), want (0,0,0)", palette[0].Red, palette[0].Green, palette[0].Blue)
	}

	// Colors should generally increase from start to end
	if palette[numColors-1].Red <= palette[0].Red {
		t.Errorf("last color Red (%v) should be > first color Red (%v)", palette[numColors-1].Red, palette[0].Red)
	}
}

func TestMakeRampColorPaletteExactSamples(t *testing.T) {
	palette := MakeColorPalette(GetColorControlPoints("Ramp Colormap"), 500)
	expected := map[int][3]int{
		0:   {0, 0, 38},
		50:  {0, 0, 128},
		155: {0, 166, 191},
		250: {0, 217, 0},
		350: {191, 204, 0},
		415: {255, 153, 0},
		499: {255, 0, 0},
	}
	for idx, want := range expected {
		got := [3]int{int(palette[idx].Red), int(palette[idx].Green), int(palette[idx].Blue)}
		if got != want {
			t.Errorf("palette[%d] = %v, want %v", idx, got, want)
		}
	}
}

func TestMakeColorPaletteValuesInRange(t *testing.T) {
	controlColors := GetColorControlPoints("Ramp Colormap")
	palette := MakeColorPalette(controlColors, 1000)

	for i, p := range palette {
		if p.Red < 0 || p.Red > 255 {
			t.Errorf("palette[%d].Red = %v, out of [0,255]", i, p.Red)
		}
		if p.Green < 0 || p.Green > 255 {
			t.Errorf("palette[%d].Green = %v, out of [0,255]", i, p.Green)
		}
		if p.Blue < 0 || p.Blue > 255 {
			t.Errorf("palette[%d].Blue = %v, out of [0,255]", i, p.Blue)
		}
	}
}
