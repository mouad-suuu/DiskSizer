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

// BatchWorkItem groups several paths for batch processing
type BatchWorkItem struct {
	Paths        []string
	MaxDepth     int
	CurrentDepth int
}

// BatchSize determines how many files to batch together
const BatchSize = 200

// IsSmallFileThreshold is the threshold for considering a file "small"
const IsSmallFileThreshold = 32 * 1024 // 32KB

// ScanDir scans a directory tree with adaptive parallel processing
func ScanDir(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
	baseName := filepath.Base(path)

	// Check file info
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

	// Read directory entries
	entries, err := os.ReadDir(path)
	if err != nil {
		return DirEntry{Path: path, Name: baseName, Size: info.Size()}, info.Size(), nil
	}

	// Quick check for small file directory
	if len(entries) > 100 {
		// Do a sampling to check if this is a small file directory
		isSmallFileDir := checkSmallFileDirectory(path, entries)
		if isSmallFileDir {
			return scanSmallFileDir(path, maxDepth, currentDepth, processedSize, entries)
		}
	}

	// Use parallel scanning for directories with many entries or at shallow depths
	if maxDepth == 0 || currentDepth < 2 || len(entries) > 20 {
		return scanDirParallel(path, maxDepth, currentDepth, processedSize, entries)
	}

	return scanDirSequential(path, maxDepth, currentDepth, processedSize, entries)
}

// checkSmallFileDirectory samples some files to determine if this is a small file directory
func checkSmallFileDirectory(dirPath string, entries []os.DirEntry) bool {
	// Sample at most 20 files to check if most are small
	sampleSize := 20
	if len(entries) < sampleSize {
		sampleSize = len(entries)
	}

	smallFileCount := 0
	for i := 0; i < sampleSize; i++ {
		// Sample evenly across the directory
		index := (i * len(entries)) / sampleSize
		if !entries[index].IsDir() {
			fullPath := filepath.Join(dirPath, entries[index].Name())
			info, err := os.Lstat(fullPath)
			if err == nil && info.Size() < IsSmallFileThreshold {
				smallFileCount++
			}
		}
	}

	// If more than 75% of sampled files are small, consider it a small file directory
	return smallFileCount > (sampleSize * 3 / 4)
}

// scanSmallFileDir optimizes scanning for directories with many small files
func scanSmallFileDir(path string, maxDepth, currentDepth int, processedSize *int64, entries []os.DirEntry) (DirEntry, int64, error) {
	entry := DirEntry{
		Path: path,
		Name: filepath.Base(path),
	}

	// OPTIMIZATION: Use specialized batch processing for small files
	var regularFiles []os.DirEntry
	var directories []os.DirEntry

	// Separate files and directories
	for _, e := range entries {
		if e.IsDir() {
			directories = append(directories, e)
		} else {
			regularFiles = append(regularFiles, e)
		}
	}

	// Process small files in larger batches for efficiency
	var wg sync.WaitGroup
	var totalSize, skipped int64
	var mu sync.Mutex
	var children []DirEntry

	// Process regular files in batches
	if len(regularFiles) > 0 {
		// Create batches of files
		fileBatches := [][]os.DirEntry{}
		currentBatch := []os.DirEntry{}

		for _, file := range regularFiles {
			currentBatch = append(currentBatch, file)
			if len(currentBatch) >= BatchSize {
				fileBatches = append(fileBatches, currentBatch)
				currentBatch = []os.DirEntry{}
			}
		}
		if len(currentBatch) > 0 {
			fileBatches = append(fileBatches, currentBatch)
		}

		// Process batches in parallel with limited concurrency
		batchSemaphore := make(chan struct{}, runtime.NumCPU()*2)
		for _, batch := range fileBatches {
			batch := batch // Create local copy for goroutine
			wg.Add(1)
			batchSemaphore <- struct{}{}

			go func() {
				defer wg.Done()
				defer func() { <-batchSemaphore }()

				// Process this batch of files
				var batchSize int64
				var batchEntries []DirEntry

				for _, file := range batch {
					fullPath := filepath.Join(path, file.Name())
					info, err := os.Lstat(fullPath)
					if err != nil || info.Mode()&os.ModeSymlink != 0 {
						continue
					}

					fileSize := info.Size()
					batchSize += fileSize
					atomic.AddInt64(processedSize, fileSize)

					batchEntries = append(batchEntries, DirEntry{
						Path: fullPath,
						Name: file.Name(),
						Size: fileSize,
					})
				}

				// Add batch results to the global results
				mu.Lock()
				totalSize += batchSize
				children = append(children, batchEntries...)
				mu.Unlock()
			}()
		}
	}

	// Process directories normally - we don't batch these
	dirResultChan := make(chan ScanResult, len(directories))
	if len(directories) > 0 {
		// Create a semaphore to limit directory concurrency
		dirSemaphore := make(chan struct{}, runtime.NumCPU())

		for _, dir := range directories {
			dir := dir // Create local copy for goroutine
			wg.Add(1)

			go func() {
				defer wg.Done()

				fullPath := filepath.Join(path, dir.Name())
				info, err := os.Lstat(fullPath)
				if err != nil || info.Mode()&os.ModeSymlink != 0 {
					return
				}

				// Acquire semaphore before scanning subdirectory
				dirSemaphore <- struct{}{}
				childEntry, childSkipped, childErr := ScanDir(fullPath, maxDepth, currentDepth+1, processedSize)
				<-dirSemaphore // Release semaphore

				dirResultChan <- ScanResult{
					Entry:   childEntry,
					Skipped: childSkipped,
					Error:   childErr,
				}
			}()
		}
	}

	// Close directory result channel when all directories are processed
	go func() {
		wg.Wait()
		close(dirResultChan)
	}()

	// Collect directory results
	for result := range dirResultChan {
		if result.Error != nil {
			skipped += 0 // Could add size estimation here
			continue
		}
		mu.Lock()
		children = append(children, result.Entry)
		totalSize += result.Entry.Size
		skipped += result.Skipped
		mu.Unlock()
	}

	// Sort children by size (larger files first)
	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	entry.Children = children
	entry.Size = totalSize
	return entry, skipped, nil
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

	// Adaptive worker count based on directory size and system
	workerCount := runtime.NumCPU() * 3
	if len(entries) > 1000 {
		// For very large directories, reduce worker count to prevent thrashing
		workerCount = runtime.NumCPU() * 2
	}
	if workerCount > 24 {
		workerCount = 24 // Cap at reasonable maximum
	}

	// Create work queue with appropriate buffer size
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

// FlatScanDirSmallFiles is a specialized flat scanner for small files
// It avoids building the full tree for better performance
func FlatScanDirSmallFiles(path string) (int64, int64, error) {
	// Stats to track
	var totalSize int64
	var fileCount int64

	// Use filepath.Walk for a flat traversal
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		// Skip errors
		if err != nil {
			return nil
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Count and add size for files
		if !info.IsDir() {
			atomic.AddInt64(&fileCount, 1)
			atomic.AddInt64(&totalSize, info.Size())
		}

		return nil
	})

	return totalSize, fileCount, err
}
