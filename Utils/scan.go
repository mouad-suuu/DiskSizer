package Utils

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
)

type DirEntry struct {
	Path     string
	Name     string
	Size     int64
	Children []DirEntry
}

func ScanDir(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
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

// GetSpinnerChars returns a slice of spinner animation characters
func GetSpinnerChars() []string {
	return []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
}
