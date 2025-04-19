package cli

import (
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"strconv"

	"github.com/spf13/cobra"
)


type Entry struct {
	Name string
	Size int64
	Children []Entry
}

var scanCmd = &cobra.Command{
	Use:   "scan [path] [depth]",
	Short: "Scans the specified directory and displays the size of top-level files and folders.",
	Run: func(cmd *cobra.Command, args []string) {
		// Set default values
		path := "."
		depth := 2

		// Override path if provided
		if len(args) >= 1 {
			path = args[0]
			if path == "*" {
				path = "/" // or "C:\\" on Windows if you want to scan the whole disk
			}
		}

		// Override depth if provided
		if len(args) >= 2 {
			d, err := strconv.Atoi(args[1])
			if err != nil || d < 1 || d > 5 {
				fmt.Println("Invalid depth. Must be a number between 1 and 5.")
				os.Exit(1)
			}
			depth = d
		}

		// Call the scan function with the path and depth
		scan(path, depth)
	},
}
func scan(path string, depth int) {
	fmt.Printf("Scanning path: %s\n", filepath.Clean(path))
	fmt.Printf("Scan depth: %d\n\n", depth)

	root, skippedSize, err := scanDir(path, depth, 0)
	if err != nil {
		fmt.Printf("Error scanning path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total accessible size: %s\n\n", formatSize(root.Size))
	printEntry(root, root.Size, 0, depth)

	if skippedSize > 0 {
		total := root.Size + skippedSize
		percent := float64(skippedSize) / float64(total) * 100
		fmt.Printf("\n⚠️  Skipped files/folders due to permission or errors: %s (%.2f%%)\n", formatSize(skippedSize), percent)
	}
}


type DirEntry struct {
	Path     string
	Name     string
	Size     int64
	Children []DirEntry
}

func scanDir(path string, maxDepth int, currentDepth int) (DirEntry, int64, error) {
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
		return entry, 0, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return entry, info.Size(), nil // Count whole folder as skipped
	}

	var totalSize int64
	var skipped int64
	for _, e := range entries {
		fullPath := filepath.Join(path, e.Name())

		// Avoid symlink loops
		childInfo, err := os.Lstat(fullPath)
		if err != nil || childInfo.Mode()&os.ModeSymlink != 0 {
			if err != nil {
				skipped += info.Size()
			}
			continue
		}

		childEntry, skippedChild, err := scanDir(fullPath, maxDepth, currentDepth+1)
		if err != nil {
			skipped += childInfo.Size()
			continue
		}
		entry.Children = append(entry.Children, childEntry)
		totalSize += childEntry.Size
		skipped += skippedChild
	}

	entry.Size = totalSize
	return entry, skipped, nil
}


func printEntry(e DirEntry, total int64, level int, maxDepth int) {
	indent := strings.Repeat("  ", level)
	percent := float64(e.Size) / float64(total) * 100
	fmt.Printf("%s%s: %s (%.2f%%)\n", indent, e.Name, formatSize(e.Size), percent)

	if level+1 < maxDepth {
		for _, child := range e.Children {
			printEntry(child, total, level+1, maxDepth)
		}
	}
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


func init() {
	rootCmd.AddCommand(scanCmd)
}