package cache

import (
	"DiskSizer/Utils"
	"sync"
	"sync/atomic"
)

type DirEntry struct {
	Path     string
	Name     string
	Size     int64
	Children []DirEntry
}

// DirSizeCache provides caching for directory sizes
type DirSizeCache struct {
	cache map[string]DirEntry
	mutex sync.RWMutex
}

// NewDirSizeCache creates a new directory size cache
func NewDirSizeCache() *DirSizeCache {
	return &DirSizeCache{
		cache: make(map[string]DirEntry),
	}
}

// Get retrieves a directory entry from the cache
func (c *DirSizeCache) Get(path string) (DirEntry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	entry, found := c.cache[path]
	return entry, found
}

// Set adds a directory entry to the cache
func (c *DirSizeCache) Set(path string, entry DirEntry) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache[path] = entry
}

// Clear empties the cache
func (c *DirSizeCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = make(map[string]DirEntry)
}

// FromUtilsDirEntry converts a Utils.DirEntry to a Cache.DirEntry
func FromUtilsDirEntry(entry Utils.DirEntry) DirEntry {
	cacheChildren := make([]DirEntry, len(entry.Children))
	for i, child := range entry.Children {
		cacheChildren[i] = FromUtilsDirEntry(child)
	}

	return DirEntry{
		Path:     entry.Path,
		Name:     entry.Name,
		Size:     entry.Size,
		Children: cacheChildren,
	}
}

// ToUtilsDirEntry converts a Cache.DirEntry to a Utils.DirEntry
func ToUtilsDirEntry(cacheEntry DirEntry) Utils.DirEntry {
	children := make([]Utils.DirEntry, len(cacheEntry.Children))
	for i, child := range cacheEntry.Children {
		children[i] = ToUtilsDirEntry(child)
	}

	return Utils.DirEntry{
		Path:     cacheEntry.Path,
		Name:     cacheEntry.Name,
		Size:     cacheEntry.Size,
		Children: children,
	}
}

// CachedScanDir implements a caching layer on top of ScanDir
func CachedScanDir(path string, maxDepth, currentDepth int, processedSize *int64, cache *DirSizeCache) (Utils.DirEntry, int64, error) {
	// Check cache first
	if cacheEntry, found := cache.Get(path); found {
		// Convert from Cache.DirEntry to Utils.DirEntry
		utilsEntry := ToUtilsDirEntry(cacheEntry)
		atomic.AddInt64(processedSize, utilsEntry.Size)
		return utilsEntry, 0, nil
	}

	// Not in cache, scan normally
	utilsEntry, skipped, err := Utils.ScanDir(path, maxDepth, currentDepth, processedSize)
	if err == nil {
		// Convert to Cache.DirEntry before adding to cache
		cacheEntry := FromUtilsDirEntry(utilsEntry)
		cache.Set(path, cacheEntry)
	}

	return utilsEntry, skipped, err
}
