package main

import (
	"reflect"
	"testing"

	"sigplot-data-service/internal/image"
)

func TestGetColorControlPoints(t *testing.T) {
	expected := []struct {
		Input  string
		Output []image.ColorPoint
	}{
		{
			Input: "Greyscale",
			Output: []image.ColorPoint{
				image.ColorPoint{0, 0, 0, 0},
				image.ColorPoint{60, 50, 50, 50},
				image.ColorPoint{100, 100, 100, 100},
			},
		},
		{
			Input: "RampColormap",
			Output: []image.ColorPoint{
				image.ColorPoint{0, 0, 0, 15},
				image.ColorPoint{10, 0, 0, 50},
				image.ColorPoint{31, 0, 65, 75},
				image.ColorPoint{50, 0, 80, 0},
				image.ColorPoint{70, 75, 80, 0},
				image.ColorPoint{83, 100, 60, 0},
				image.ColorPoint{100, 100, 0, 0},
			},
		},
		{
			Input: "ColorWheel",
			Output: []image.ColorPoint{
				image.ColorPoint{0, 100, 100, 0},
				image.ColorPoint{20, 0, 80, 40},
				image.ColorPoint{30, 0, 100, 100},
				image.ColorPoint{50, 10, 10, 0},
				image.ColorPoint{65, 100, 0, 0},
				image.ColorPoint{88, 100, 40, 0},
				image.ColorPoint{100, 100, 100, 0},
			},
		},
		{
			Input: "Spectrum",
			Output: []image.ColorPoint{
				image.ColorPoint{0, 0, 75, 0},
				image.ColorPoint{22, 0, 90, 90},
				image.ColorPoint{37, 0, 0, 85},
				image.ColorPoint{49, 90, 0, 85},
				image.ColorPoint{68, 90, 0, 0},
				image.ColorPoint{80, 90, 90, 0},
				image.ColorPoint{100, 95, 95, 95},
			},
		},
	}
	for _, exp := range expected {
		result := image.GetColorControlPoints(exp.Input)
		if !reflect.DeepEqual(result, exp.Output) {
			t.Errorf(
				"GetColorControlPoints(%s) returned %v instead of %v",
				exp.Input,
				result,
				exp.Output,
			)
		}
	}
}

func TestMakeColorPalette(t *testing.T) {
	expected := []struct {
		InputControlColors []image.ColorPoint
		InputNumColors     int
		Output             []image.ColorPoint
	}{
		{
			InputControlColors: []image.ColorPoint{
				image.ColorPoint{0, 0, 0, 0},
				image.ColorPoint{60, 50, 50, 50},
				image.ColorPoint{100, 100, 100, 100},
			},
			InputNumColors: 6,
			Output: []image.ColorPoint{
				image.ColorPoint{0, 0, 0, 0},
				image.ColorPoint{60, 50, 50, 50},
				image.ColorPoint{100, 100, 100, 100},
			},
		},
	}
	for _, exp := range expected {
		result := image.MakeColorPalette(exp.InputControlColors, exp.InputNumColors)
		if !reflect.DeepEqual(result, exp.Output) {
			t.Errorf(
				"MakeColorPalette(%v, %d) returned %v instead of %v",
				exp.InputControlColors,
				exp.InputNumColors,
				result,
				exp.Output,
			)
		}
	}
}
