package bluefile

import (
	"testing"
)

func FuzzConvertFileData(f *testing.F) {
	// Seed with valid format codes and some data
	formats := []string{"SF", "CF", "SD", "CD", "SI", "CI", "SL", "CL", "SB", "CB", "SP"}
	for _, fmt := range formats {
		f.Add([]byte{0x00, 0x00, 0x80, 0x3F}, fmt) // 1.0 as float32 LE
	}
	f.Add([]byte{}, "SF")     // empty input
	f.Add([]byte{0xFF}, "XX") // unknown format

	f.Fuzz(func(t *testing.T, data []byte, format string) {
		// Should not panic on any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ConvertFileData panicked with format=%q len(data)=%d: %v", format, len(data), r)
			}
		}()
		ConvertFileData(data, format)
	})
}
