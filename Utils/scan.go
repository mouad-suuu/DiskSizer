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
