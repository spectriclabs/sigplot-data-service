package image

import (
	"fmt"
	"math"
	"testing"
)

func BenchmarkTransform(b *testing.B) {
	strategies := []string{"mean", "max", "min", "maxabs", "first"}
	sizes := []int{100, 1000, 10000}

	for _, strategy := range strategies {
		for _, size := range sizes {
			data := make([]float64, size)
			for i := range data {
				data[i] = float64(i) * 0.1
			}

			b.Run(fmt.Sprintf("%s/%d", strategy, size), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					Transform(data, strategy)
				}
			})
		}
	}
}

func BenchmarkDownSampleLineInX(b *testing.B) {
	cases := []struct {
		inSize  int
		outSize int
	}{
		{1000, 100},
		{10000, 500},
		{100000, 1000},
	}

	for _, tc := range cases {
		data := make([]float64, tc.inSize)
		for i := range data {
			data[i] = float64(i) * 0.01
		}

		b.Run(fmt.Sprintf("%d_to_%d", tc.inSize, tc.outSize), func(b *testing.B) {
			outData := make([]float64, tc.outSize)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				DownSampleLineInX(data, tc.outSize, "mean", outData, 0)
			}
		})
	}
}

func BenchmarkDownSampleLineInY(b *testing.B) {
	cases := []struct {
		totalSize int
		outxsize  int
	}{
		{1000, 100},  // 10 lines of 100
		{10000, 500}, // 20 lines of 500
	}

	for _, tc := range cases {
		data := make([]float64, tc.totalSize)
		for i := range data {
			data[i] = float64(i) * 0.01
		}

		b.Run(fmt.Sprintf("%d_to_%d", tc.totalSize, tc.outxsize), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				DownSampleLineInY(data, tc.outxsize, "mean")
			}
		})
	}
}

func BenchmarkApplyCXmode(b *testing.B) {
	modes := []string{"Ma", "Re", "Im", "Ph"}
	numComplex := 1000
	// 2 float64 per complex element (real, imag)
	data := make([]float64, numComplex*2)
	for i := 0; i < numComplex; i++ {
		data[i*2] = math.Cos(float64(i) * 0.01)
		data[i*2+1] = math.Sin(float64(i) * 0.01)
	}

	for _, mode := range modes {
		b.Run(mode, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ApplyCXmode(data, mode, true)
			}
		})
	}
}

func BenchmarkApplyCXmodeRealLog(b *testing.B) {
	modes := []string{"Lo", "L2"}
	source := make([]float64, 1000)
	for i := range source {
		source[i] = 1 + float64(i%97)*0.125
	}

	for _, mode := range modes {
		b.Run(mode, func(b *testing.B) {
			data := make([]float64, len(source))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				copy(data, source)
				ApplyCXmode(data, mode, false)
			}
		})
	}
}

func BenchmarkCreateOutput(b *testing.B) {
	sizes := []int{1000, 10000}

	for _, size := range sizes {
		data := make([]float64, size)
		for i := range data {
			data[i] = float64(i) / float64(size)
		}

		b.Run(fmt.Sprintf("RGBA/%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				CreateOutput(data, "RGBA", 0.0, 1.0, "Ramp Colormap")
			}
		})
	}
}

func BenchmarkMakeColorPalette(b *testing.B) {
	controlColors := GetColorControlPoints("Ramp Colormap")

	b.Run("256_colors", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			MakeColorPalette(controlColors, 256)
		}
	})
}
