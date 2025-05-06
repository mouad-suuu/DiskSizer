package Utils

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"sync"
	"sync/atomic"
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

// ScanDir scans a directory tree with adaptive parallel processing
func ScanDir(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
	// // Check if this is a known problematic directory that should be skipped or processed differently
	baseName := filepath.Base(path)
	// if skipDirs[baseName] {
	// 	// For problematic directories, do a fast estimate instead of deep scanning
	// 	entry := DirEntry{
	// 		Path: path,
	// 		Name: baseName,
	// 	}

	// 	// Quick size estimation for problematic directories
	// 	size, err := EstimateDirectorySize(path, 10)
	// 	if err != nil {
	// 		return entry, 0, err
	// 	}

	// 	entry.Size = size
	// 	// atomic.AddInt64(processedSize, entry.Size)
	// 	return entry, 0, nil
	// }

	// Determine whether to use parallel or sequential scanning
	// Use parallel scanning for:
	// 1. Top-level directories (currentDepth < 2)
	// 2. Directories with many entries regardless of depth
	info, err := os.Lstat(path)
	if err != nil {
		return DirEntry{Path: path, Name: baseName}, 0, err
	}

	if !info.IsDir() {
		entry := DirEntry{
			Path: path,
			Name: baseName,
			Size: info.Size(),
		}
		atomic.AddInt64(processedSize, entry.Size)
		return entry, 0, nil
	}

	// Read directory entries to determine count
	entries, err := os.ReadDir(path)
	if err != nil {
		return DirEntry{Path: path, Name: baseName, Size: info.Size()}, info.Size(), nil
	}

	// Use parallel scanning for directories with many entries or at shallow depths
	if maxDepth == 0 || currentDepth < 2 || len(entries) > 20 {
		return scanDirParallel(path, maxDepth, currentDepth, processedSize, entries)
	}

	return scanDirSequential(path, maxDepth, currentDepth, processedSize, entries)
}

// scanDirSequential performs a sequential directory scan (for deeper levels)
func scanDirSequential(path string, maxDepth, currentDepth int, processedSize *int64, entries []os.DirEntry) (DirEntry, int64, error) {
	entry := DirEntry{
		Path: path,
		Name: filepath.Base(path),
	}

	var totalSize, skipped int64
	for _, e := range entries {
		fullPath := filepath.Join(path, e.Name())

		childInfo, err := os.Lstat(fullPath)
		if err != nil || childInfo.Mode()&os.ModeSymlink != 0 {
			if err != nil {
				skipped += 1024 // assume 1KB for error cases
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
	return entry, skipped, nil
}

// scanDirParallel performs an optimized parallel directory scan
func scanDirParallel(path string, maxDepth, currentDepth int, processedSize *int64, entries []os.DirEntry) (DirEntry, int64, error) {
	entry := DirEntry{
		Path: path,
		Name: filepath.Base(path),
	}

	// For very small directories, just process sequentially
	if len(entries) < 5 {
		return scanDirSequential(path, maxDepth, currentDepth, processedSize, entries)
	}

	var wg sync.WaitGroup
	resultChan := make(chan ScanResult, len(entries))

	// Increase worker count for IO-bound operations
	// Use 2-4x CPU count for better IO throughput
	workerCount := runtime.NumCPU() * 3
	if workerCount > 24 {
		workerCount = 24 // Cap at reasonable maximum
	}

	// Create work queue with appropriate buffer size
	workQueue := make(chan WorkItem, len(entries))

	// Start worker goroutines
	for i := 0; i < workerCount; i++ {
		go func() {
			for work := range workQueue {
				// Check if this is a problematic directory before scanning

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

	// Process entries to create work items
	// Group work items by priority (process regular files first, then directories)
	var regularFiles []os.DirEntry
	var directories []os.DirEntry

	for _, e := range entries {
		if e.IsDir() {
			directories = append(directories, e)
		} else {
			regularFiles = append(regularFiles, e)
		}
	}

	// Process regular files first (these are quick)
	for _, e := range regularFiles {
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

	// Then process directories (potentially more time-consuming)
	for _, e := range directories {
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
	return entry, skipped, nil
}
