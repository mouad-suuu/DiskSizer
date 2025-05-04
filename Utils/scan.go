package Utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
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

var spinnerDone = make(chan bool)
var size int64

func StartSpinner(processedSize *int64) string {
	symbols := []string{"|", "/", "-", "\\"}
	i := 0

	go func() {
		for {
			select {
			case <-spinnerDone:
				return
			default:
				size = atomic.LoadInt64(processedSize)
				fmt.Printf("\r%s Scanned: %s", color.YellowString(symbols[i%len(symbols)]), formatSize(size))
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
	return fmt.Sprintf("\r%s Scanned: %s", color.YellowString(symbols[i%len(symbols)]), formatSize(size))
}

func formatSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
