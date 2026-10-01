package cache

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

var testCacheDirSeq uint64

func TestUrlToCacheFileName(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "simple URL no query",
			url:  "data/file.tmp",
			want: "datafiletmp",
		},
		{
			name: "URL with query params",
			url:  "data/file.tmp?x=1&y=2",
			want: "datafiletmp_x1y2",
		},
		{
			name: "URL with dots and slashes",
			url:  "path/to/some.file.dat",
			want: "pathtosomefiledat",
		},
		{
			name: "URL with multiple special chars",
			url:  "a/b/c.d?key=val&foo=bar",
			want: "abcd_keyvalfoobar",
		},
		{
			name: "empty URL",
			url:  "",
			want: "",
		},
		{
			name: "URL with no slashes or dots",
			url:  "simplename",
			want: "simplename",
		},
		{
			name: "URL with only question mark",
			url:  "file?",
			want: "file_",
		},
		{
			name: "URL with equals and ampersand",
			url:  "f?a=1&b=2&c=3",
			want: "f_a1b2c3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UrlToCacheFileName(tt.url)
			if got != tt.want {
				t.Errorf("UrlToCacheFileName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestPutAndGetDataFromCache(t *testing.T) {
	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	subDir := "/"
	fileName := "testfile"
	data := []byte("hello cache data")

	if err := c.PutItemInCache(fileName, subDir, data); err != nil {
		t.Fatalf("PutItemInCache error: %v", err)
	}

	got, err := c.GetDataFromCache(fileName, subDir)
	if err != nil {
		t.Fatalf("GetDataFromCache error: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("GetDataFromCache = %q, want %q", string(got), string(data))
	}
}

func TestPutAndGetUseJoinedPathWithoutTrailingSlash(t *testing.T) {
	resetMemCache(t)

	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	subDir := "outputFiles/"
	fileName := "sds_rds_joined_path"
	data := []byte("joined path data")

	if err := c.PutItemInCache(fileName, subDir, data); err != nil {
		t.Fatalf("PutItemInCache error: %v", err)
	}

	wantPath := filepath.Join(tmpDir, "outputFiles", fileName)
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected cache file at joined path: %v", err)
	}
	if _, err := os.Stat(tmpDir + subDir + fileName); !os.IsNotExist(err) {
		t.Fatalf("cache file written to concatenated path, err=%v", err)
	}

	got, err := c.GetDataFromCache(fileName, subDir)
	if err != nil {
		t.Fatalf("GetDataFromCache error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("GetDataFromCache = %q, want %q", got, data)
	}

	reader, err := c.GetItemFromCache(fileName, subDir)
	if err != nil {
		t.Fatalf("GetItemFromCache error: %v", err)
	}
	if closer, ok := reader.(interface{ Close() error }); ok {
		defer closer.Close()
	}
}

func TestGetDataFromCacheDoesNotCacheEmptyFile(t *testing.T) {
	resetMemCache(t)

	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	fileName := "sds_empty"
	fullPath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(fullPath, nil, 0644); err != nil {
		t.Fatalf("WriteFile empty: %v", err)
	}

	got, err := c.GetDataFromCache(fileName, "/")
	if err != nil {
		t.Fatalf("GetDataFromCache empty error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetDataFromCache empty len = %d, want 0", len(got))
	}

	want := []byte("now populated")
	if err := os.WriteFile(fullPath, want, 0644); err != nil {
		t.Fatalf("WriteFile populated: %v", err)
	}
	got, err = c.GetDataFromCache(fileName, "/")
	if err != nil {
		t.Fatalf("GetDataFromCache populated error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("GetDataFromCache after empty = %q, want %q", got, want)
	}
}

func TestPutItemInCacheRefreshesMemoryCache(t *testing.T) {
	resetMemCache(t)

	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	fileName := "sds_refresh"
	if err := c.PutItemInCache(fileName, "/", []byte("old")); err != nil {
		t.Fatalf("PutItemInCache old error: %v", err)
	}
	if _, err := c.GetDataFromCache(fileName, "/"); err != nil {
		t.Fatalf("GetDataFromCache old error: %v", err)
	}

	want := []byte("new")
	if err := c.PutItemInCache(fileName, "/", want); err != nil {
		t.Fatalf("PutItemInCache new error: %v", err)
	}
	got, err := c.GetDataFromCache(fileName, "/")
	if err != nil {
		t.Fatalf("GetDataFromCache new error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("GetDataFromCache after put = %q, want %q", got, want)
	}
}

func TestConcurrentPutDoesNotExposePartialData(t *testing.T) {
	resetMemCache(t)

	tmpDir := testCacheDir(t)
	c := &Cache{Location: tmpDir}
	fileName := "sds_atomic"
	subDir := "atomic/"
	oldData := bytes.Repeat([]byte("a"), 32*1024)
	newData := bytes.Repeat([]byte("b"), 128*1024)
	if err := c.PutItemInCache(fileName, subDir, oldData); err != nil {
		t.Fatalf("PutItemInCache initial error: %v", err)
	}

	errCh := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			if err := c.PutItemInCache(fileName, subDir, newData); err != nil {
				errCh <- err
				return
			}
		}
	}()

	key := cachePath(tmpDir, subDir, fileName)
	for i := 0; i < 100; i++ {
		memCache.deletePath(key)
		got, err := c.GetDataFromCache(fileName, subDir)
		if err != nil {
			t.Fatalf("GetDataFromCache error: %v", err)
		}
		if !bytes.Equal(got, oldData) && !bytes.Equal(got, newData) {
			t.Fatalf("observed partial cache data of length %d", len(got))
		}
	}
	<-done
	select {
	case err := <-errCh:
		t.Fatalf("PutItemInCache concurrent error: %v", err)
	default:
	}
}

func TestMemoryCacheKeyIncludesFullPath(t *testing.T) {
	resetMemCache(t)

	dir1 := testCacheDir(t)
	dir2 := testCacheDir(t)

	c1 := &Cache{Location: dir1}
	c2 := &Cache{Location: dir2}
	fileName := "sds_same_name"
	if err := c1.PutItemInCache(fileName, "outputFiles/", []byte("one")); err != nil {
		t.Fatalf("PutItemInCache c1 error: %v", err)
	}
	if err := c2.PutItemInCache(fileName, "outputFiles/", []byte("two")); err != nil {
		t.Fatalf("PutItemInCache c2 error: %v", err)
	}

	got1, err := c1.GetDataFromCache(fileName, "outputFiles/")
	if err != nil {
		t.Fatalf("GetDataFromCache c1 error: %v", err)
	}
	got2, err := c2.GetDataFromCache(fileName, "outputFiles/")
	if err != nil {
		t.Fatalf("GetDataFromCache c2 error: %v", err)
	}
	if string(got1) != "one" || string(got2) != "two" {
		t.Fatalf("cache collision: got %q and %q", got1, got2)
	}
}

func TestMemoryCacheEvictsLRUByByteSize(t *testing.T) {
	resetMemCache(t)
	memCache.setMaxBytes(5)

	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	if err := c.PutItemInCache("sds_a", "/", []byte("aaa")); err != nil {
		t.Fatalf("PutItemInCache a error: %v", err)
	}
	if err := c.PutItemInCache("sds_b", "/", []byte("bbb")); err != nil {
		t.Fatalf("PutItemInCache b error: %v", err)
	}

	if _, ok := memCache.getPath(cachePath(tmpDir, "/", "sds_a")); ok {
		t.Fatal("least recently used entry was not evicted")
	}
	if got, ok := memCache.getPath(cachePath(tmpDir, "/", "sds_b")); !ok || string(got) != "bbb" {
		t.Fatalf("new entry missing from memory cache: ok=%v got=%q", ok, got)
	}

	if err := c.PutItemInCache("sds_b", "/", []byte("bb")); err != nil {
		t.Fatalf("PutItemInCache b replacement error: %v", err)
	}
	if err := c.PutItemInCache("sds_c", "/", []byte("cc")); err != nil {
		t.Fatalf("PutItemInCache c error: %v", err)
	}
	if _, ok := memCache.getPath(cachePath(tmpDir, "/", "sds_b")); !ok {
		t.Fatal("replacement size was double-counted")
	}
	if _, ok := memCache.getPath(cachePath(tmpDir, "/", "sds_c")); !ok {
		t.Fatal("new entry missing after replacement")
	}
}

func TestCheckCacheOnceRemovesMinioAndInvalidatesMemory(t *testing.T) {
	resetMemCache(t)

	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	fileName := "sds_minio_object"
	if err := c.PutItemInCache(fileName, "miniocache/", []byte("cached minio data")); err != nil {
		t.Fatalf("PutItemInCache error: %v", err)
	}
	if _, err := c.GetDataFromCache(fileName, "miniocache/"); err != nil {
		t.Fatalf("GetDataFromCache priming error: %v", err)
	}

	fullPath := filepath.Join(tmpDir, "miniocache", fileName)
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(fullPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes error: %v", err)
	}
	if err := checkCacheOnce(filepath.Join(tmpDir, "miniocache"), 0); err != nil {
		t.Fatalf("checkCacheOnce error: %v", err)
	}

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Fatalf("expected minio cache file removal, err=%v", err)
	}
	if _, err := c.GetDataFromCache(fileName, "miniocache/"); err == nil {
		t.Fatal("expected cache miss after on-disk eviction invalidated memory")
	}
}

func TestPutAndGetItemFromCache(t *testing.T) {
	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	subDir := "/"
	fileName := "testitem"
	data := []byte("seekable cache data")

	if err := c.PutItemInCache(fileName, subDir, data); err != nil {
		t.Fatalf("PutItemInCache error: %v", err)
	}

	reader, err := c.GetItemFromCache(fileName, subDir)
	if err != nil {
		t.Fatalf("GetItemFromCache error: %v", err)
	}

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("GetItemFromCache content = %q, want %q", string(content), string(data))
	}

	if closer, ok := reader.(interface{ Close() error }); ok {
		closer.Close()
	}
}

func TestGetDataFromCacheMissingFile(t *testing.T) {
	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	_, err := c.GetDataFromCache("nonexistent", "/")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestGetItemFromCacheMissingFile(t *testing.T) {
	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	_, err := c.GetItemFromCache("nonexistent", "/")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestPutItemInCacheCreatesSubDir(t *testing.T) {
	tmpDir := testCacheDir(t)

	c := &Cache{Location: tmpDir}
	subDir := "/newsubdir/"
	fileName := "testfile"
	data := []byte("data in subdir")

	if err := c.PutItemInCache(fileName, subDir, data); err != nil {
		t.Fatalf("PutItemInCache error: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmpDir, "newsubdir"))
	if err != nil {
		t.Fatalf("subdirectory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func resetMemCache(t *testing.T) {
	t.Helper()
	memCache.clear()
	memCache.setMaxBytes(defaultMemCacheMaxSize)
	t.Cleanup(func() {
		memCache.clear()
		memCache.setMaxBytes(defaultMemCacheMaxSize)
	})
}

func testCacheDir(t testing.TB) string {
	t.Helper()
	base := filepath.Join(".", ".cache-test")
	name := fmt.Sprintf("%d-%d", os.Getpid(), atomic.AddUint64(&testCacheDirSeq, 1))
	dir := filepath.Join(base, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create cache test dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
		_ = os.Remove(base)
	})
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve cache test dir: %v", err)
	}
	return abs
}
