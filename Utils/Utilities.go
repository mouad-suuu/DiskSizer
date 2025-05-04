package Utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/windows"
)

func FormatSize(size int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
		TB = 1 << 40
	)

	switch {
	case size >= TB:
		return FormatFloat(float64(size)/TB) + " TB"
	case size >= GB:
		return FormatFloat(float64(size)/GB) + " GB"
	case size >= MB:
		return FormatFloat(float64(size)/MB) + " MB"
	case size >= KB:
		return FormatFloat(float64(size)/KB) + " KB"
	default:
		return FormatFloat(float64(size)) + " B"
	}
}

func FormatFloat(f float64) string {
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

func GetSizeColor(size, totalSize int64) string {
	if totalSize == 0 {
		return "white"
	}

	ratio := float64(size) / float64(totalSize)

	switch {
	case ratio > 0.5:
		return "red"
	case ratio > 0.25:
		return "orange"
	case ratio > 0.1:
		return "yellow"
	default:
		return "green"
	}
}

func GetFileIcon(filename string, isDir bool) string {
	if isDir {
		return "📁" // Folder icon for directories
	}

	ext := filepath.Ext(filename)
	switch ext {
	case ".go":
		return "🔷"
	case ".txt", ".md":
		return "📝"
	case ".jpg", ".png", ".gif":
		return "🖼️"
	case ".mp3", ".wav":
		return "🎵"
	case ".mp4", ".avi", ".mov":
		return "🎞️"
	case ".pdf":
		return "📕"
	case ".zip", ".tar", ".gz":
		return "📦"
	case ".exe", ".app":
		return "⚙️"
	default:
		return "📄"
	}
}
