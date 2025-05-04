package Cache

import (
	"sync"
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
