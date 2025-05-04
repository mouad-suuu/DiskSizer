package Utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

type DirEntry struct {
	Path     string
	Name     string
	Size     int64
	Children []DirEntry
}

// WorkItem represents a directory scan work item for the worker pool
type WorkItem struct {
	Path         string
	MaxDepth     int
	CurrentDepth int
}

// ScanResult represents the result of a directory scan
type ScanResult struct {
	Entry   DirEntry
	Skipped int64
	Error   error
}

// ScanDir scans a directory tree with parallel processing for better performance
func ScanDir(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
	// For small depths, use concurrent scanning for better performance
	if maxDepth == 0 || currentDepth < 2 {
		return scanDirParallel(path, maxDepth, currentDepth, processedSize)
	}
	return scanDirSequential(path, maxDepth, currentDepth, processedSize)
}

// scanDirSequential performs a sequential directory scan (for deeper levels)
func scanDirSequential(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
	entry := DirEntry{
		Path: path,
		Name: filepath.Base(path),
	}

	info, err := os.Stat(path)
	if err != nil {
		return entry, 0, err
	}
	if !info.IsDir() {
		entry.Size = info.Size()
		atomic.AddInt64(processedSize, entry.Size)
		return entry, 0, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return entry, info.Size(), nil
	}

	var totalSize, skipped int64
	for _, e := range entries {
		fullPath := filepath.Join(path, e.Name())
		childInfo, err := os.Lstat(fullPath)
		if err != nil || childInfo.Mode()&os.ModeSymlink != 0 {
			if err != nil {
				skipped += info.Size()
			}
			continue
		}

		childEntry, skippedChild, err := ScanDir(fullPath, maxDepth, currentDepth+1, processedSize)
		if err != nil {
			skipped += childInfo.Size()
			continue
		}
		entry.Children = append(entry.Children, childEntry)
		totalSize += childEntry.Size
		skipped += skippedChild
	}

	// Sort children by size (larger files first)
	sort.Slice(entry.Children, func(i, j int) bool {
		return entry.Children[i].Size > entry.Children[j].Size
	})

	entry.Size = totalSize
	atomic.AddInt64(processedSize, entry.Size)
	return entry, skipped, nil
}

// scanDirParallel performs a parallel directory scan using worker pools
func scanDirParallel(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
	entry := DirEntry{
		Path: path,
		Name: filepath.Base(path),
	}

	info, err := os.Stat(path)
	if err != nil {
		return entry, 0, err
	}
	if !info.IsDir() {
		entry.Size = info.Size()
		atomic.AddInt64(processedSize, entry.Size)
		return entry, 0, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return entry, info.Size(), nil
	}

	// For small directories, just process sequentially
	if len(entries) < 5 {
		return scanDirSequential(path, maxDepth, currentDepth, processedSize)
	}

	var wg sync.WaitGroup
	resultChan := make(chan ScanResult, len(entries))
	workerCount := runtime.NumCPU()

	// Create work queue
	workQueue := make(chan WorkItem, len(entries))

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		go func() {
			for work := range workQueue {
				childEntry, childSkipped, childErr := ScanDir(work.Path, work.MaxDepth, work.CurrentDepth, processedSize)
				resultChan <- ScanResult{
					Entry:   childEntry,
					Skipped: childSkipped,
					Error:   childErr,
				}
				wg.Done()
			}
		}()
	}

	// Add work to the queue
	for _, e := range entries {
		fullPath := filepath.Join(path, e.Name())
		childInfo, err := os.Lstat(fullPath)
		if err != nil || childInfo.Mode()&os.ModeSymlink != 0 {
			continue
		}

		wg.Add(1)
		workQueue <- WorkItem{
			Path:         fullPath,
			MaxDepth:     maxDepth,
			CurrentDepth: currentDepth + 1,
		}
	}

	// Close work queue when all tasks are added
	close(workQueue)

	// Wait for completion in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var totalSize, skipped int64
	var children []DirEntry

	for result := range resultChan {
		if result.Error != nil {
			skipped += 0 // Could add file size estimation here
			continue
		}
		children = append(children, result.Entry)
		totalSize += result.Entry.Size
		skipped += result.Skipped
	}

	// Sort children by size (larger files first)
	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	entry.Children = children
	entry.Size = totalSize
	atomic.AddInt64(processedSize, entry.Size)
	return entry, skipped, nil
}

// CachedScanDir implements a caching layer on top of ScanDir
func CachedScanDir(path string, maxDepth, currentDepth int, processedSize *int64, cache *DirSizeCache) (DirEntry, int64, error) {
	// Check cache first
	if entry, found := cache.Get(path); found {
		atomic.AddInt64(processedSize, entry.Size)
		return entry, 0, nil
	}

	// Not in cache, scan normally
	entry, skipped, err := ScanDir(path, maxDepth, currentDepth, processedSize)
	if err == nil {
		// Add to cache
		cache.Set(path, entry)
	}

	return entry, skipped, err
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

func FormatSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)

	switch {
	case size >= TB:
		return formatFloat(float64(size)/TB) + " TB"
	case size >= GB:
		return formatFloat(float64(size)/GB) + " GB"
	case size >= MB:
		return formatFloat(float64(size)/MB) + " MB"
	case size >= KB:
		return formatFloat(float64(size)/KB) + " KB"
	default:
		return formatFloat(float64(size)) + " B"
	}
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func GetSpinnerChars() []string {
	return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}

// FastFileInfo provides quick file size estimation for large directories
func FastFileInfo(path string) (int64, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, false, err
	}

	isDir := info.IsDir()
	size := info.Size()

	return size, isDir, nil
}

// GetUsableSpace returns the available disk space
func GetUsableSpace(path string) (uint64, error) {
	// Windows-specific implementation
	if runtime.GOOS == "windows" {
		// Get the volume path (e.g., C:\)
		volumePath := filepath.VolumeName(path)
		if volumePath == "" {
			// If path doesn't have a volume name, use the current directory
			cwd, err := os.Getwd()
			if err != nil {
				return 0, err
			}
			volumePath = filepath.VolumeName(cwd)
		}

		// Ensure volume path ends with separator
		if !strings.HasSuffix(volumePath, "\\") {
			volumePath += "\\"
		}

		// Use Windows API via golang.org/x/sys/windows
		var free, total, totalFree uint64
		windows.GetDiskFreeSpaceEx(
			windows.StringToUTF16Ptr(volumePath),
			&free,
			&total,
			&totalFree)

		return free, nil
	}

	// For non-Windows platforms, return an error
	return 0, fmt.Errorf("GetUsableSpace not implemented for %s", runtime.GOOS)
}

// EstimateDirectorySize provides a fast size estimate by sampling
func EstimateDirectorySize(path string, sampleSize int) (int64, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	// For small directories, just get the exact size
	if len(entries) <= sampleSize {
		var size int64
		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			entrySize, _, err := FastFileInfo(entryPath)
			if err == nil {
				size += entrySize
			}
		}
		return size, nil
	}

	// For large directories, sample a subset
	var totalSize int64
	sampleCount := 0

	// Pick evenly distributed samples
	step := len(entries) / sampleSize
	for i := 0; i < len(entries) && sampleCount < sampleSize; i += step {
		if i >= len(entries) {
			break
		}

		entryPath := filepath.Join(path, entries[i].Name())
		entrySize, _, err := FastFileInfo(entryPath)
		if err == nil {
			totalSize += entrySize
			sampleCount++
		}
	}

	if sampleCount == 0 {
		return 0, nil
	}

	// Extrapolate from samples
	avgSize := totalSize / int64(sampleCount)
	estimatedSize := avgSize * int64(len(entries))

	return estimatedSize, nil
}
