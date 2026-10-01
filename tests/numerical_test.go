package main

import (
	"math"
	"testing"

	"github.com/spectriclabs/sigplot-data-service/internal/image"
)

func TestTransform(t *testing.T) {
	expected := []struct {
		DataIn    []float64
		Transform string
		Output    float64
	}{
		{
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "mean",
			Output:    4.3875,
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
			// Unknown transform defaults to "first", returning dataIn[0]
			DataIn:    []float64{3.0, 4.5, 6.4, 1.1, 8.6, 9.3, -3.3, 5.5},
			Transform: "foo",
			Output:    3.0,
		},
	}

	for _, exp := range expected {
		result := image.Transform(exp.DataIn, exp.Transform)
		if math.Abs(result-exp.Output) > 1e-10 {
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
	datain := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	outxsize := 3
	outData := make([]float64, outxsize)
	image.DownSampleLineInX(datain, outxsize, "mean", outData, 0)
	expected := []float64{1.5, 3.5, 5.5}
	for i, v := range expected {
		if outData[i] != v {
			t.Errorf("DownSampleLineInX mean: index %d got %f expected %f", i, outData[i], v)
		}
	}

	// Test expansion (outxsize > input length)
	smallIn := []float64{1.0, 2.0, 3.0}
	expandOut := make([]float64, 6)
	image.DownSampleLineInX(smallIn, 6, "first", expandOut, 0)
	expandExpected := []float64{1.0, 1.0, 2.0, 2.0, 3.0, 3.0}
	for i, v := range expandExpected {
		if expandOut[i] != v {
			t.Errorf("DownSampleLineInX expand: index %d got %f expected %f", i, expandOut[i], v)
		}
	}
}

func TestDownSampleLineInY(t *testing.T) {
	// 2 lines of 3 elements each
	datain := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	outxsize := 3
	result := image.DownSampleLineInY(datain, outxsize, "mean")
	expected := []float64{2.5, 3.5, 4.5}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("DownSampleLineInY mean: index %d got %f expected %f", i, result[i], v)
		}
	}

	// Test with "max" transform
	result = image.DownSampleLineInY(datain, outxsize, "max")
	expectedMax := []float64{4.0, 5.0, 6.0}
	for i, v := range expectedMax {
		if result[i] != v {
			t.Errorf("DownSampleLineInY max: index %d got %f expected %f", i, result[i], v)
		}
	}
}

func TestApplyCXmode(t *testing.T) {
	// Test non-complex Re mode (passthrough)
	datain := []float64{1.0, 2.0, 3.0}
	result := image.ApplyCXmode(datain, "Re", false)
	for i, v := range datain {
		if result[i] != v {
			t.Errorf("ApplyCXmode Re non-complex: index %d got %f expected %f", i, result[i], v)
		}
	}

	// Test complex Ma mode: sqrt(re^2 + im^2)
	complexIn := []float64{3.0, 4.0, 0.0, 5.0}
	result = image.ApplyCXmode(complexIn, "Ma", true)
	expectedMa := []float64{5.0, 5.0}
	for i, v := range expectedMa {
		if math.Abs(result[i]-v) > 1e-10 {
			t.Errorf("ApplyCXmode Ma complex: index %d got %f expected %f", i, result[i], v)
		}
	}

	// Test complex Re mode: extract real parts
	result = image.ApplyCXmode(complexIn, "Re", true)
	expectedRe := []float64{3.0, 0.0}
	for i, v := range expectedRe {
		if result[i] != v {
			t.Errorf("ApplyCXmode Re complex: index %d got %f expected %f", i, result[i], v)
		}
	}

	// Test complex Im mode: extract imaginary parts
	result = image.ApplyCXmode(complexIn, "Im", true)
	expectedIm := []float64{4.0, 5.0}
	for i, v := range expectedIm {
		if result[i] != v {
			t.Errorf("ApplyCXmode Im complex: index %d got %f expected %f", i, result[i], v)
		}
	}
}
