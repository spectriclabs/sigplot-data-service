package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

var benchCacheDirSeq uint64

func BenchmarkUrlToCacheFileName(b *testing.B) {
	urls := []struct {
		name string
		url  string
	}{
		{"short", "data/file.tmp"},
		{"with_params", "data/file.tmp?x=1&y=2&z=3"},
		{"long_path", "/api/v1/locations/local/files/very/deep/nested/path/to/data.blue?subsize=1024&outfmt=RGBA&colormap=Ramp%20Colormap"},
	}

	for _, tc := range urls {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				UrlToCacheFileName(tc.url)
			}
		})
	}
}

func BenchmarkGetDataFromCacheHitMemory(b *testing.B) {
	memCache.clear()
	defer memCache.clear()

	tmpDir := benchCacheDir(b)

	c := &Cache{Location: tmpDir}
	data := make([]byte, 64*1024)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}
	if err := c.PutItemInCache("sds_hit", "bench/", data); err != nil {
		b.Fatal(err)
	}
	if _, err := c.GetDataFromCache("sds_hit", "bench/"); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := c.GetDataFromCache("sds_hit", "bench/")
		if err != nil {
			b.Fatal(err)
		}
		if len(got) != len(data) {
			b.Fatalf("len(got) = %d, want %d", len(got), len(data))
		}
	}
}

func BenchmarkGetDataFromCacheMissDisk(b *testing.B) {
	memCache.clear()
	defer memCache.clear()

	tmpDir := benchCacheDir(b)

	c := &Cache{Location: tmpDir}
	data := make([]byte, 64*1024)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "bench"), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "bench", "sds_miss"), data, 0644); err != nil {
		b.Fatal(err)
	}
	memCache.setMaxBytes(0)
	defer memCache.setMaxBytes(defaultMemCacheMaxSize)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := c.GetDataFromCache("sds_miss", "bench/")
		if err != nil {
			b.Fatal(err)
		}
		if len(got) != len(data) {
			b.Fatalf("len(got) = %d, want %d", len(got), len(data))
		}
	}
}

func BenchmarkConcurrentGetPut(b *testing.B) {
	memCache.clear()
	defer memCache.clear()

	tmpDir := benchCacheDir(b)

	c := &Cache{Location: tmpDir}
	data := make([]byte, 64*1024)
	for i := range data {
		data[i] = byte(i & 0xFF)
	}
	if err := c.PutItemInCache("sds_concurrent", "bench/", data); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var total int64
		for pb.Next() {
			if err := c.PutItemInCache("sds_concurrent", "bench/", data); err != nil {
				b.Errorf("PutItemInCache: %v", err)
				return
			}
			got, err := c.GetDataFromCache("sds_concurrent", "bench/")
			if err != nil {
				b.Errorf("GetDataFromCache: %v", err)
				return
			}
			total += int64(len(got))
		}
		atomic.AddInt64(&benchConcurrentBytes, total)
	})
}

var benchConcurrentBytes int64

func BenchmarkPutGetRoundTrip(b *testing.B) {
	sizes := []struct {
		name  string
		bytes int
	}{
		{"1KB", 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, s := range sizes {
		tmpDir := benchCacheDir(b)

		c := &Cache{Location: tmpDir}
		data := make([]byte, s.bytes)
		for i := range data {
			data[i] = byte(i & 0xFF)
		}
		fileName := fmt.Sprintf("bench_%s", s.name)

		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				c.PutItemInCache(fileName, "bench/", data)
				c.GetDataFromCache(fileName, "bench/")
			}
		})
	}
}

func benchCacheDir(b *testing.B) string {
	b.Helper()
	base := filepath.Join(".", ".cache-bench")
	name := strings.NewReplacer("/", "_", " ", "_").Replace(b.Name())
	dir := filepath.Join(base, fmt.Sprintf("%s-%d-%d", name, os.Getpid(), atomic.AddUint64(&benchCacheDirSeq, 1)))
	if err := os.MkdirAll(dir, 0755); err != nil {
		b.Fatalf("failed to create cache bench dir: %v", err)
	}
	b.Cleanup(func() {
		_ = os.RemoveAll(dir)
		_ = os.Remove(base)
	})
	abs, err := filepath.Abs(dir)
	if err != nil {
		b.Fatalf("failed to resolve cache bench dir: %v", err)
	}
	return abs
}
