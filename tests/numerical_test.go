package main

import (
	"github.com/spectriclabs/sigplot-data-service/internal/image"
	"math"
	"testing"
)

func TestSuppressNaN(t *testing.T) {
	expected := []struct {
		Input  float64
		Output float64
	}{
		{Input: 0.0, Output: 0.0},
		{Input: 13000000.5, Output: 13000000.5},
		{Input: math.NaN(), Output: 0},
		{Input: math.Inf(1), Output: math.Inf(1)},
	}

	for _, exp := range expected {
		result := numerical.SuppressNaN(exp.Input)
		if result != exp.Output {
			t.Errorf(
				"SuppressNaN(%f) returned %f instead of %f",
				exp.Input,
				result,
				exp.Output,
			)
		}
	}
}

func TestTransform(t *testing.T) {
	expected := []struct {
		DataIn    []float64
		Transform string
		Output    float64
	}{
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "mean",
			Output:    4.387499999999999,
		},
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "max",
			Output:    9.3,
		},
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "min",
			Output:    -3.3,
		},
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "absmax",
			Output:    3.0,
		},
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "first",
			Output:    3.0,
		},
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "foo",
			Output:    0,
		},
	}

	for _, exp := range expected {
		result := image.Transform(exp.DataIn, exp.Transform)
		if result != exp.Output {
			t.Errorf(
				"Transform(%v, %s) returned %f instead of %f",
				exp.DataIn,
				exp.Transform,
				result,
				exp.Output,
			)
		}
	}
}

func TestDownSampleLineInX(t *testing.T) {

}

func TestDownSampleLineInY(t *testing.T) {

}

func TestApplyCXmode(t *testing.T) {

}
