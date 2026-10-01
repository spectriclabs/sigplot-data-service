package bluefile

import (
	"fmt"
	"testing"
)

func BenchmarkConvertFileData(b *testing.B) {
	type formatSpec struct {
		name         string
		format       string
		bytesPerElem int
	}

	formats := []formatSpec{
		{"SF", "SF", 4},
		{"SD", "SD", 8},
		{"SI", "SI", 2},
		{"SB", "SB", 1},
		{"SL", "SL", 4},
	}

	sizes := []struct {
		name  string
		bytes int
	}{
		{"1KB", 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, f := range formats {
		for _, s := range sizes {
			// Round down to nearest multiple of bytesPerElem
			dataLen := (s.bytes / f.bytesPerElem) * f.bytesPerElem
			data := make([]byte, dataLen)
			// Fill with sequential bytes
			for i := range data {
				data[i] = byte(i & 0xFF)
			}

			b.Run(fmt.Sprintf("%s/%s", f.name, s.name), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					ConvertFileData(data, f.format)
				}
			})
		}
	}
}
