package sds

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync"
	"testing"
)

func BenchmarkGetBytesFromReader(b *testing.B) {
	data := make([]byte, 64*1024) // 64KB buffer
	for i := range data {
		data[i] = byte(i & 0xFF)
	}

	b.Run("sequential_4KB", func(b *testing.B) {
		reader := bytes.NewReader(data)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			getBytesFromReaderMu(reader, 0, 4096, IoMutex)
		}
	})

	b.Run("concurrent_4KB", func(b *testing.B) {
		reader := bytes.NewReader(data)
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				getBytesFromReaderMu(reader, 0, 4096, IoMutex)
			}
		})
	})
}

func BenchmarkProcessLine(b *testing.B) {
	// Create synthetic float32 data (1000 elements)
	numElements := 1000
	buf := new(bytes.Buffer)
	for i := 0; i < numElements; i++ {
		binary.Write(buf, binary.LittleEndian, float32(math.Sin(float64(i)*0.01)))
	}
	rawData := buf.Bytes()

	b.Run("SF_1000_to_100", func(b *testing.B) {
		reader := bytes.NewReader(rawData)
		outData := make([]float64, 100)
		done := make(chan bool, 1)

		req := RdsRequest{
			FileFormat:     "SF",
			FileXSize:      numElements,
			FileDataOffset: 0,
			Xstart:         0,
			Xsize:          numElements,
			Ystart:         0,
			Outxsize:       100,
			Transform:      "mean",
			Reader:         reader,
			ReaderMutex:    &sync.Mutex{},
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ProcessLine(outData, 0, done, req)
			<-done
		}
	})
}

func BenchmarkComputeRequestSizes(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &RdsRequest{
			X1: 10, X2: 500,
			Y1: 5, Y2: 250,
		}
		req.ComputeRequestSizes()
	}
}

func BenchmarkComputeYSize(b *testing.B) {
	cases := []struct {
		name   string
		format string
		size   float64
		xsize  int
	}{
		{"SF", "SF", 40000, 100},
		{"SD", "SD", 80000, 100},
		{"CF", "CF", 80000, 100},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req := &RdsRequest{
					FileDataSize: tc.size,
					FileFormat:   tc.format,
					FileXSize:    tc.xsize,
				}
				req.ComputeYSize()
			}
		})
	}
}
