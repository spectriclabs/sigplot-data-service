package bluefile

import (
	"math"
	"testing"
	"unsafe"
)

func TestGetFileTypeInfo(t *testing.T) {
	tests := []struct {
		name          string
		format        string
		wantBPA       float64
		wantIsComplex bool
	}{
		{"SF", "SF", 4.0, false},
		{"CF", "CF", 4.0, true},
		{"SD", "SD", 8.0, false},
		{"CD", "CD", 8.0, true},
		{"SI", "SI", 2.0, false},
		{"CI", "CI", 2.0, true},
		{"SB", "SB", 1.0, false},
		{"CB", "CB", 1.0, true},
		{"SL", "SL", 4.0, false},
		{"CL", "CL", 4.0, true},
		{"SP", "SP", 0.125, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bpa, isComplex := GetFileTypeInfo(tt.format)
			if bpa != tt.wantBPA {
				t.Errorf("GetFileTypeInfo(%q) bytesPerAtom = %v, want %v", tt.format, bpa, tt.wantBPA)
			}
			if isComplex != tt.wantIsComplex {
				t.Errorf("GetFileTypeInfo(%q) isComplex = %v, want %v", tt.format, isComplex, tt.wantIsComplex)
			}
		})
	}
}

func TestGetFileTypeInfoInvalidFormat(t *testing.T) {
	for _, format := range []string{"", "S", "X", "XX", "CZ"} {
		bpa, isComplex := GetFileTypeInfo(format)
		if bpa != 0 || isComplex {
			t.Errorf("GetFileTypeInfo(%q) = (%v, %v), want (0, false)", format, bpa, isComplex)
		}
	}
}

func TestConvertFileDataSF(t *testing.T) {
	// Create 4 bytes representing float32(3.14)
	var f float32 = 3.14
	b := make([]byte, 4)
	*(*float32)(unsafe.Pointer(&b[0])) = f

	out := ConvertFileData(b, "SF")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if math.Abs(out[0]-float64(f)) > 1e-6 {
		t.Errorf("ConvertFileData SF = %v, want %v", out[0], f)
	}
}

func TestConvertFileDataSI(t *testing.T) {
	// Create 2 bytes representing int16(-1234)
	var v int16 = -1234
	b := make([]byte, 2)
	*(*int16)(unsafe.Pointer(&b[0])) = v

	out := ConvertFileData(b, "SI")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0] != float64(v) {
		t.Errorf("ConvertFileData SI = %v, want %v", out[0], v)
	}
}

func TestConvertFileDataSB(t *testing.T) {
	// Create 1 byte representing int8(42)
	b := []byte{42}

	out := ConvertFileData(b, "SB")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0] != 42.0 {
		t.Errorf("ConvertFileData SB = %v, want 42.0", out[0])
	}
}

func TestConvertFileDataSBNegative(t *testing.T) {
	// int8(-5) is 0xFB
	b := []byte{0xFB}

	out := ConvertFileData(b, "SB")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0] != -5.0 {
		t.Errorf("ConvertFileData SB negative = %v, want -5.0", out[0])
	}
}

func TestConvertFileDataSD(t *testing.T) {
	var f float64 = 2.718281828
	b := make([]byte, 8)
	*(*float64)(unsafe.Pointer(&b[0])) = f

	out := ConvertFileData(b, "SD")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0] != f {
		t.Errorf("ConvertFileData SD = %v, want %v", out[0], f)
	}
}

func TestConvertFileDataSL(t *testing.T) {
	var v int32 = 100000
	b := make([]byte, 4)
	*(*int32)(unsafe.Pointer(&b[0])) = v

	out := ConvertFileData(b, "SL")
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	if out[0] != float64(v) {
		t.Errorf("ConvertFileData SL = %v, want %v", out[0], v)
	}
}

func TestConvertFileDataSP(t *testing.T) {
	// Packed data: byte 0xA5 = 10100101 => [1,0,1,0,0,1,0,1]
	b := []byte{0xA5}
	out := ConvertFileData(b, "SP")
	if len(out) != 8 {
		t.Fatalf("len = %d, want 8", len(out))
	}
	expected := []float64{1, 0, 1, 0, 0, 1, 0, 1}
	for i, want := range expected {
		if out[i] != want {
			t.Errorf("out[%d] = %v, want %v", i, out[i], want)
		}
	}
}

func TestConvertFileDataMultipleValues(t *testing.T) {
	// Two float32 values
	b := make([]byte, 8)
	*(*float32)(unsafe.Pointer(&b[0])) = 1.0
	*(*float32)(unsafe.Pointer(&b[4])) = 2.0

	out := ConvertFileData(b, "SF")
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0] != 1.0 {
		t.Errorf("out[0] = %v, want 1.0", out[0])
	}
	if out[1] != 2.0 {
		t.Errorf("out[1] = %v, want 2.0", out[1])
	}
}

func TestConvertFileDataEmptyInput(t *testing.T) {
	out := ConvertFileData([]byte{}, "SF")
	if len(out) != 0 {
		t.Errorf("len = %d, want 0", len(out))
	}
}

func TestConvertFileDataInvalidFormat(t *testing.T) {
	for _, format := range []string{"", "S", "XX", "CZ"} {
		out := ConvertFileData([]byte{1, 2, 3, 4}, format)
		if len(out) != 0 {
			t.Errorf("ConvertFileData(%q) len = %d, want 0", format, len(out))
		}
	}
}

func TestConvertFileDataIgnoresTrailingPartialAtom(t *testing.T) {
	out := ConvertFileData([]byte{1, 0, 2}, "SI")
	if len(out) != 1 || out[0] != 1 {
		t.Fatalf("ConvertFileData trailing partial = %v, want [1]", out)
	}
}
