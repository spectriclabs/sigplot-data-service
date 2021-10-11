package main

import (
	"reflect"
	"testing"

	"github.com/spectriclabs/sigplot-data-service/internal/image"
)

func TestGetColorControlPoints(t *testing.T) {
	expected := []struct {
		Input  string
		Output []image.Pixel
	}{
		{
			Input: "Greyscale",
			Output: []image.Pixel{
				{0, 0, 0, 0},
				{60, 50, 50, 50},
				{100, 100, 100, 100},
			},
		},
		{
			Input: "RampColormap",
			Output: []image.Pixel{
				{0, 0, 0, 15},
				{10, 0, 0, 50},
				{31, 0, 65, 75},
				{50, 0, 80, 0},
				{70, 75, 80, 0},
				{83, 100, 60, 0},
				{100, 100, 0, 0},
			},
		},
		{
			Input: "ColorWheel",
			Output: []image.Pixel{
				{0, 100, 100, 0},
				{20, 0, 80, 40},
				{30, 0, 100, 100},
				{50, 10, 10, 0},
				{65, 100, 0, 0},
				{88, 100, 40, 0},
				{100, 100, 100, 0},
			},
		},
		{
			Input: "Spectrum",
			Output: []image.Pixel{
				{0, 0, 75, 0},
				{22, 0, 90, 90},
				{37, 0, 0, 85},
				{49, 90, 0, 85},
				{68, 90, 0, 0},
				{80, 90, 90, 0},
				{100, 95, 95, 95},
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
		InputControlColors []image.Pixel
		InputNumColors     int
		Output             []image.Pixel
	}{
		{
			InputControlColors: []image.Pixel{
				{0, 0, 0, 0},
				{60, 50, 50, 50},
				{100, 100, 100, 100},
			},
			InputNumColors: 6,
			Output: []image.Pixel{
				{0, 0, 0, 0},
				{60, 50, 50, 50},
				{100, 100, 100, 100},
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
