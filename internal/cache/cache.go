package cache

import (
	"container/list"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultMemCacheMaxSize int64 = 100 * 1024 * 1024

var (
	memCache     = newMemoryCache(defaultMemCacheMaxSize)
	cacheWriteMu sync.Mutex
	cacheTempSeq uint64
)

type memoryCache struct {
	mu       sync.Mutex
	items    map[cacheKey]*list.Element
	paths    map[string]*list.Element
	lru      *list.List
	size     int64
	maxBytes int64
}

type memoryCacheEntry struct {
	key  cacheKey
	path string
	data []byte
	size int64
}

type cacheKey struct {
	location string
	subDir   string
	name     string
}

func newMemoryCache(maxBytes int64) *memoryCache {
	return &memoryCache{
		items:    make(map[cacheKey]*list.Element),
		paths:    make(map[string]*list.Element),
		lru:      list.New(),
		maxBytes: maxBytes,
	}
}

func (m *memoryCache) get(key cacheKey) ([]byte, bool) {
	m.mu.Lock()

	elem, ok := m.items[key]
	if !ok {
		m.mu.Unlock()
		return nil, false
	}
	if elem != m.lru.Front() {
		m.lru.MoveToFront(elem)
	}
	data := elem.Value.(*memoryCacheEntry).data
	m.mu.Unlock()
	return data, true
}

func (m *memoryCache) getPath(path string) ([]byte, bool) {
	m.mu.Lock()

	elem, ok := m.paths[path]
	if !ok {
		m.mu.Unlock()
		return nil, false
	}
	if elem != m.lru.Front() {
		m.lru.MoveToFront(elem)
	}
	data := elem.Value.(*memoryCacheEntry).data
	m.mu.Unlock()
	return data, true
}

func (m *memoryCache) set(key cacheKey, path string, data []byte) {
	if len(data) == 0 {
		m.deleteKeyAndPath(key, path)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	size := int64(len(data))
	if size > m.maxBytes {
		m.removeKeyLocked(key)
		m.removePathLocked(path)
		return
	}

	if elem, ok := m.paths[path]; ok && elem.Value.(*memoryCacheEntry).key != key {
		m.removeElementLocked(elem)
	}
	if elem, ok := m.items[key]; ok {
		entry := elem.Value.(*memoryCacheEntry)
		m.size += size - entry.size
		delete(m.paths, entry.path)
		entry.path = path
		entry.data = data
		entry.size = size
		m.paths[path] = elem
		m.lru.MoveToFront(elem)
	} else {
		elem := m.lru.PushFront(&memoryCacheEntry{key: key, path: path, data: data, size: size})
		m.items[key] = elem
		m.paths[path] = elem
		m.size += size
	}

	for m.size > m.maxBytes {
		m.removeOldestLocked()
	}
}

func (m *memoryCache) deleteKeyAndPath(key cacheKey, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeKeyLocked(key)
	m.removePathLocked(path)
}

func (m *memoryCache) deletePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removePathLocked(path)
}

func (m *memoryCache) clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[cacheKey]*list.Element)
	m.paths = make(map[string]*list.Element)
	m.lru.Init()
	m.size = 0
}

func (m *memoryCache) setMaxBytes(maxBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxBytes = maxBytes
	for m.size > m.maxBytes {
		m.removeOldestLocked()
	}
}

func (m *memoryCache) removeOldestLocked() {
	elem := m.lru.Back()
	if elem != nil {
		m.removeElementLocked(elem)
	}
}

func (m *memoryCache) removeKeyLocked(key cacheKey) {
	elem, ok := m.items[key]
	if ok {
		m.removeElementLocked(elem)
	}
}

func (m *memoryCache) removePathLocked(path string) {
	elem, ok := m.paths[path]
	if ok {
		m.removeElementLocked(elem)
	}
}

func (m *memoryCache) removeElementLocked(elem *list.Element) {
	entry := elem.Value.(*memoryCacheEntry)
	delete(m.items, entry.key)
	delete(m.paths, entry.path)
	m.size -= entry.size
	m.lru.Remove(elem)
}

type Cache struct {
	Location string
}

// UrlToCacheFileName uses a url and query string
// to form SigPlot Data Services' cached file name.
func UrlToCacheFileName(url string) string {
	response := strings.Replace(url, "?", "_", 1)
	replacer := strings.NewReplacer("&", "", "=", "", ".", "", "/", "")
	cacheFileName := replacer.Replace(response)
	return cacheFileName
}

// GetDataFromCache retrieves data from a provided `cacheFileName`
// within a `subDir` directory. The returned slice is cache-owned and
// must be treated as read-only by callers.
func (c *Cache) GetDataFromCache(cacheFileName string, subDir string) ([]byte, error) {
	key := newCacheKey(c.Location, subDir, cacheFileName)
	if data, ok := memCache.get(key); ok {
		return data, nil
	}

	fullPath := cachePath(c.Location, subDir, cacheFileName)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		memCache.set(key, fullPath, data)
	}

	return data, nil
}

// GetItemFromCache retrieves a file from a `cacheFileName`
// within a `subDir` directory and returns an `io.ReadSeeker`
func (c *Cache) GetItemFromCache(cacheFileName string, subDir string) (io.ReadSeeker, error) {
	fullPath := cachePath(c.Location, subDir, cacheFileName)
	file, err := os.Open(fullPath)
	return file, err
}

// PutItemInCache places `data` into file denoted by `cacheFileName`
// within `subDir`
func (c *Cache) PutItemInCache(cacheFileName string, subDir string, data []byte) error {
	key := newCacheKey(c.Location, subDir, cacheFileName)
	fullPath := cachePath(c.Location, subDir, cacheFileName)
	fullPathDirectory := filepath.Dir(fullPath)
	if err := os.MkdirAll(fullPathDirectory, 0755); err != nil {
		return err
	}

	tempPath := tempCachePath(fullPath)
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	num, err := file.Write(data)
	if err == nil && num != len(data) {
		err = io.ErrShortWrite
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	cacheWriteMu.Lock()
	defer cacheWriteMu.Unlock()

	if err := os.Rename(tempPath, fullPath); err != nil {
		return err
	}
	removeTemp = false

	memCache.set(key, fullPath, data)
	return nil
}

func newCacheKey(location string, subDir string, cacheFileName string) cacheKey {
	return cacheKey{location: location, subDir: subDir, name: cacheFileName}
}

func cachePath(location string, subDir string, cacheFileName string) string {
	return filepath.Join(location, subDir, cacheFileName)
}

func tempCachePath(fullPath string) string {
	seq := atomic.AddUint64(&cacheTempSeq, 1)
	return fullPath + ".tmp." + strconv.Itoa(os.Getpid()) + "." + strconv.FormatUint(seq, 10)
}

// CheckCache runs a check every `checkInterval` seconds
// and purges if the current cache size exceeds `maxBytes`
func CheckCache(cachePath string, checkInterval int, maxBytes int64) {
	// duration expressed in nano seconds
	nextRun := time.Now()
	for {
		if nextRun.Before(time.Now()) {
			if err := checkCacheOnce(cachePath, maxBytes); err != nil {
				log.Println("CheckCache Error: ", err)
				time.Sleep(5 * time.Second)
				continue
			}

			nextRun = nextRun.Add(time.Second * time.Duration(checkInterval))
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}

type diskCacheFile struct {
	name    string
	path    string
	size    int64
	modTime time.Time
}

func checkCacheOnce(cachePath string, maxBytes int64) error {
	files, err := ioutil.ReadDir(cachePath)
	if err != nil {
		return err
	}

	var currentBytes int64
	cacheFiles := make([]diskCacheFile, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !file.Mode().IsRegular() || !isSDSCacheFile(file.Name()) {
			continue
		}
		currentBytes += file.Size()
		cacheFiles = append(cacheFiles, diskCacheFile{
			name:    file.Name(),
			path:    filepath.Join(cachePath, file.Name()),
			size:    file.Size(),
			modTime: file.ModTime(),
		})
	}

	sort.Slice(cacheFiles, func(i, j int) bool {
		return cacheFiles[i].modTime.Before(cacheFiles[j].modTime)
	})

	for currentBytes > maxBytes && len(cacheFiles) > 0 {
		oldestFile := cacheFiles[0]
		cacheFiles = cacheFiles[1:]

		log.Println("Cache over Maximum. Removing Old File", oldestFile.name)
		cacheWriteMu.Lock()
		if err := os.Remove(oldestFile.path); err != nil {
			cacheWriteMu.Unlock()
			return err
		}
		memCache.deletePath(oldestFile.path)
		cacheWriteMu.Unlock()
		currentBytes -= oldestFile.size
	}

	return nil
}

func isSDSCacheFile(name string) bool {
	return strings.HasPrefix(name, "sds") && !strings.Contains(name, ".tmp.")
}
