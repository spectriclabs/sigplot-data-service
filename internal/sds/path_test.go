package sds

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spectriclabs/sigplot-data-service/internal/cache"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
)

func TestResolvePath(t *testing.T) {
	base := filepath.Join("testdata", "root")
	tests := []struct {
		name     string
		userPath string
		wantErr  bool
	}{
		{name: "empty", userPath: ""},
		{name: "nested", userPath: "dir/file.tmp"},
		{name: "clean", userPath: "dir//file.tmp"},
		{name: "parent", userPath: "../secret", wantErr: true},
		{name: "nested parent", userPath: "dir/../file.tmp", wantErr: true},
		{name: "absolute", userPath: "/etc/hosts", wantErr: true},
		{name: "backslash parent", userPath: `dir\..\secret`, wantErr: true},
		{name: "nul", userPath: "file\x00.tmp", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePath(base, tt.userPath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolvePath(%q) returned nil error", tt.userPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePath(%q) error = %v", tt.userPath, err)
			}
			rel, err := filepath.Rel(mustAbs(t, base), got)
			if err != nil {
				t.Fatalf("filepath.Rel error = %v", err)
			}
			if rel == ".." || rel == "."+string(filepath.Separator)+".." {
				t.Fatalf("ResolvePath(%q) escaped base: %q", tt.userPath, got)
			}
		})
	}
}

func TestObjectListPrefix(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		userPath string
		want     string
		wantErr  bool
	}{
		{name: "root", want: ""},
		{name: "base only", base: "data", want: "data/"},
		{name: "subdir", userPath: "subdir", want: "subdir/"},
		{name: "base subdir", base: "data", userPath: "subdir", want: "data/subdir/"},
		{name: "trailing slash", userPath: "subdir/", want: "subdir/"},
		{name: "sibling prefix", userPath: "foo", want: "foo/"},
		{name: "parent rejected", userPath: "../secret", wantErr: true},
		{name: "nested parent rejected", userPath: "foo/../bar", wantErr: true},
		{name: "absolute rejected", userPath: "/secret", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ObjectListPrefix(tt.base, tt.userPath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ObjectListPrefix(%q, %q) returned nil error", tt.base, tt.userPath)
				}
				return
			}
			if err != nil {
				t.Fatalf("ObjectListPrefix(%q, %q) error = %v", tt.base, tt.userPath, err)
			}
			if got != tt.want {
				t.Fatalf("ObjectListPrefix(%q, %q) = %q, want %q", tt.base, tt.userPath, got, tt.want)
			}
		})
	}
}

func BenchmarkResolvePath(b *testing.B) {
	base := b.TempDir()
	for i := 0; i < b.N; i++ {
		if _, err := ResolvePath(base, "dir/file.tmp"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOpenDataSourceLocal(b *testing.B) {
	prevLog := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(prevLog)

	root := b.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.tmp"), []byte("blue data"), 0644); err != nil {
		b.Fatal(err)
	}
	cfg := &config.Config{
		LocationDetails: []config.Location{
			{LocationName: "local", LocationType: "localFile", Path: root},
		},
	}
	sdsCache := &cache.Cache{Location: filepath.Join(root, "cache") + string(os.PathSeparator)}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		reader, err := OpenDataSource(cfg, sdsCache, "local", "file.tmp")
		if err != nil {
			b.Fatal(err)
		}
		if closer, ok := reader.(io.Closer); ok {
			_ = closer.Close()
		}
	}
}

func TestOpenDataSourceMinioNoCache(t *testing.T) {
	body := []byte("blue data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("location") {
			_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/root/file.tmp") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", "9")
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.Header().Set("ETag", `"etag"`)
		if r.Method == http.MethodHead {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()

	cfg := &config.Config{
		UseCache: false,
		LocationDetails: []config.Location{
			{
				LocationName:   "minio",
				LocationType:   "minio",
				Path:           "root",
				MinioBucket:    "bucket",
				Location:       server.Listener.Addr().String(),
				MinioAccessKey: "access",
				MinioSecretKey: "secret",
			},
		},
	}

	reader, err := OpenDataSource(cfg, &cache.Cache{Location: "cache/"}, "minio", "file.tmp")
	if err != nil {
		t.Fatalf("OpenDataSource error = %v", err)
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("OpenDataSource data = %q, want %q", got, body)
	}
}

func TestOpenDataSourceMinioMissingDoesNotCache(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	cacheDir := t.TempDir() + string(os.PathSeparator)
	cfg := &config.Config{
		UseCache:      true,
		CacheLocation: cacheDir,
		LocationDetails: []config.Location{
			{
				LocationName:   "minio-missing",
				LocationType:   "minio",
				MinioBucket:    "bucket",
				Location:       server.Listener.Addr().String(),
				MinioAccessKey: "access",
				MinioSecretKey: "secret",
			},
		},
	}

	reader, err := OpenDataSource(cfg, &cache.Cache{Location: cacheDir}, "minio-missing", "missing.tmp")
	if err == nil {
		t.Fatalf("OpenDataSource returned reader %v and nil error", reader)
	}
	entries, readErr := os.ReadDir(filepath.Join(cacheDir, "miniocache"))
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("ReadDir cache error = %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("cache entries = %d, want 0", len(entries))
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
