package sds

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spectriclabs/sigplot-data-service/internal/config"
)

var minioClients sync.Map

var ErrInvalidPath = errors.New("invalid path")

func IsInvalidPath(err error) bool {
	return errors.Is(err, ErrInvalidPath)
}

type minioClientKey struct {
	name      string
	endpoint  string
	accessKey string
	secretKey string
	useSSL    bool
}

func ResolvePath(base, userPath string) (string, error) {
	if strings.ContainsRune(userPath, '\x00') {
		return "", ErrInvalidPath
	}
	normalizedPath := strings.ReplaceAll(userPath, "\\", "/")
	normalizedUserPath := filepath.FromSlash(normalizedPath)
	if filepath.IsAbs(normalizedUserPath) || hasParentSegment(normalizedPath) {
		return "", fmt.Errorf("%w: path escapes location", ErrInvalidPath)
	}

	basePath := filepath.Clean(base)
	if !filepath.IsAbs(basePath) {
		var err error
		basePath, err = filepath.Abs(basePath)
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(basePath, normalizedUserPath), nil
}

func ResolveObjectKey(base, userPath string) (string, error) {
	if strings.ContainsRune(userPath, '\x00') {
		return "", ErrInvalidPath
	}
	normalizedUserPath := strings.ReplaceAll(userPath, "\\", "/")
	if path.IsAbs(normalizedUserPath) || hasParentSegment(normalizedUserPath) {
		return "", fmt.Errorf("%w: path escapes location", ErrInvalidPath)
	}
	normalizedBase := strings.ReplaceAll(base, "\\", "/")
	if hasParentSegment(normalizedBase) {
		return "", fmt.Errorf("%w: base path escapes location", ErrInvalidPath)
	}

	fullPath := path.Clean(path.Join(normalizedBase, normalizedUserPath))
	if fullPath == "." {
		return "", nil
	}
	return fullPath, nil
}

func ObjectListPrefix(base, userPath string) (string, error) {
	prefix, err := ResolveObjectKey(base, userPath)
	if err != nil || prefix == "" || strings.HasSuffix(prefix, "/") {
		return prefix, err
	}
	return prefix + "/", nil
}

func ContentTypeForPath(filePath string) string {
	if strings.Contains(filePath, ".tmp") || strings.Contains(filePath, ".prm") {
		return "application/bluefile"
	}
	return "application/binary"
}

func MinioClientForLocation(location config.Location) (*minio.Client, error) {
	key := minioClientKey{
		name:      location.LocationName,
		endpoint:  location.Location,
		accessKey: location.MinioAccessKey,
		secretKey: location.MinioSecretKey,
		useSSL:    location.MinioUseSSL,
	}
	if client, ok := minioClients.Load(key); ok {
		return client.(*minio.Client), nil
	}
	client, err := minio.New(location.Location, &minio.Options{
		Creds:  credentials.NewStaticV4(location.MinioAccessKey, location.MinioSecretKey, ""),
		Secure: location.MinioUseSSL,
	})
	if err != nil {
		return nil, err
	}
	actual, _ := minioClients.LoadOrStore(key, client)
	return actual.(*minio.Client), nil
}

func hasParentSegment(p string) bool {
	if p == ".." {
		return true
	}
	for p != "" {
		i := strings.IndexByte(p, '/')
		var part string
		if i < 0 {
			part = p
			p = ""
		} else {
			part = p[:i]
			p = p[i+1:]
		}
		if part == ".." {
			return true
		}
	}
	return false
}
