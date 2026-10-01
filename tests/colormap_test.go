package main

import (
	"testing"

	"github.com/spectriclabs/sigplot-data-service/internal/image"
)

func TestGetColorControlPoints(t *testing.T) {
	pixel := func(pos, red, green, blue float64) image.Pixel {
		return image.Pixel{Position: pos, Red: red, Green: green, Blue: blue}
	}
	expected := []struct {
		Input    string
		NumElems int
		Points   []image.Pixel
	}{
		{Input: "Greyscale", NumElems: 3, Points: []image.Pixel{pixel(0, 0, 0, 0), pixel(60, 50, 50, 50), pixel(100, 100, 100, 100)}},
		{Input: "Ramp Colormap", NumElems: 7, Points: []image.Pixel{pixel(0, 0, 0, 15), pixel(10, 0, 0, 50), pixel(31, 0, 65, 75), pixel(50, 0, 85, 0), pixel(70, 75, 80, 0), pixel(83, 100, 60, 0), pixel(100, 100, 0, 0)}},
		{Input: "Color Wheel", NumElems: 7, Points: []image.Pixel{pixel(0, 100, 100, 0), pixel(20, 0, 80, 40), pixel(30, 0, 100, 100), pixel(50, 10, 10, 0), pixel(65, 100, 0, 0), pixel(88, 100, 40, 0), pixel(100, 100, 100, 0)}},
		{Input: "Spectrum", NumElems: 7, Points: []image.Pixel{pixel(0, 0, 75, 0), pixel(22, 0, 90, 90), pixel(37, 0, 0, 85), pixel(49, 90, 0, 85), pixel(68, 90, 0, 0), pixel(80, 90, 90, 0), pixel(100, 95, 95, 95)}},
		{Input: "calewhite", NumElems: 7, Points: []image.Pixel{pixel(0, 100, 100, 100), pixel(16.666, 0, 0, 100), pixel(33.333, 0, 100, 100), pixel(50, 0, 100, 0), pixel(66.666, 100, 100, 0), pixel(83.333, 100, 0, 0), pixel(100, 100, 0, 100)}},
		{Input: "HotDesat", NumElems: 8, Points: []image.Pixel{pixel(0, 27.84, 27.84, 85.88), pixel(14.2857, 0, 0, 35.69), pixel(28.571, 0, 100, 100), pixel(42.857, 0, 49.8, 0), pixel(57.14286, 100, 100, 0), pixel(71.42857, 100, 37.65, 0), pixel(85.7143, 41.96, 0, 0), pixel(100, 87.84, 29.8, 29.8)}},
		{Input: "Sunset", NumElems: 7, Points: []image.Pixel{pixel(0, 10, 0, 23), pixel(18, 34, 0, 60), pixel(36, 58, 20, 47), pixel(55, 74, 20, 28), pixel(72, 90, 43, 0), pixel(87, 100, 72, 0), pixel(100, 100, 100, 76)}},
	}
	for _, exp := range expected {
		result := image.GetColorControlPoints(exp.Input)
		if len(result) != exp.NumElems {
			t.Errorf(
				"GetColorControlPoints(%s) returned %d elements, expected %d",
				exp.Input,
				len(result),
				exp.NumElems,
			)
		}
		for i := range exp.Points {
			if result[i] != exp.Points[i] {
				t.Errorf("GetColorControlPoints(%s)[%d] = %+v, expected %+v", exp.Input, i, result[i], exp.Points[i])
			}
		}
	}

	// Unknown colormap should fall back to default (Ramp Colormap = 7 points)
	result := image.GetColorControlPoints("nonexistent")
	if len(result) != 7 {
		t.Errorf("GetColorControlPoints(nonexistent) returned %d elements, expected 7 (default)", len(result))
	}
}

func TestMakeColorPalette(t *testing.T) {
	controlColors := image.GetColorControlPoints("Greyscale")
	numColors := 10
	result := image.MakeColorPalette(controlColors, numColors)
	if len(result) != numColors {
		t.Errorf("MakeColorPalette returned %d colors, expected %d", len(result), numColors)
	}
}

func TestMakeRampColorPaletteExactSamples(t *testing.T) {
	palette := image.MakeColorPalette(image.GetColorControlPoints("Ramp Colormap"), 500)
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
