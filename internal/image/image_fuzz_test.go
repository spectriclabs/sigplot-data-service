package image

import (
	"encoding/binary"
	"math"
	"testing"
)

func FuzzTransform(f *testing.F) {
	// Encode float64 slices as bytes for fuzzer
	encode := func(vals ...float64) []byte {
		buf := make([]byte, 8*len(vals))
		for i, v := range vals {
			binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
		}
		return buf
	}

	transforms := []string{"mean", "max", "min", "maxabs", "first"}
	for _, tr := range transforms {
		f.Add(encode(1.0, 2.0, 3.0), tr)
	}
	f.Add(encode(0.0), "max")

	f.Fuzz(func(t *testing.T, rawData []byte, transform string) {
		// Decode bytes back to float64 slice
		n := len(rawData) / 8
		if n == 0 {
			return
		}
		data := make([]float64, n)
		for i := range data {
			data[i] = math.Float64frombits(binary.LittleEndian.Uint64(rawData[i*8:]))
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Transform panicked with transform=%q len(data)=%d: %v", transform, n, r)
			}
		}()
		Transform(data, transform)
	})
}
