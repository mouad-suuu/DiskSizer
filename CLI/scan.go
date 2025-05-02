package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fatih/color"

	"strconv"

	"github.com/spf13/cobra"
)

type Entry struct {
	Name     string
	Size     int64
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
	label := color.New(color.FgGreen).SprintFunc()
	value := color.New(color.FgHiWhite).SprintFunc()

	fmt.Printf("\n%s %s\n", label("📂 Scanning path:"), value(filepath.Clean(path)))
	fmt.Printf("%s %d\n\n", label("🔎 Scan depth:"), depth)

	var processedSize int64
	StartSpinner("Scanning...", &processedSize)

	start := time.Now()
	root, skippedSize, err := scanDir(path, depth, 0, &processedSize)

	spinnerDone <- true
	fmt.Printf("\r") // Clear spinner line
	if err != nil {
		color.Red("❌ Error scanning path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Scan complete in %s\n", time.Since(start).Truncate(time.Millisecond))
	fmt.Printf("%s %s\n\n", label("📦 Total accessible size:"), value(formatSize(root.Size)))
	printEntry(root, root.Size, 0, depth)

	if skippedSize > 0 {
		total := root.Size + skippedSize
		percent := float64(skippedSize) / float64(total) * 100
		fmt.Printf("\n%s %s (%.2f%%)\n",
			color.New(color.FgHiRed, color.Bold).Sprint("⚠️  Skipped due to errors/permissions:"),
			value(formatSize(skippedSize)),
			percent,
		)
	}
}

type DirEntry struct {
	Path     string
	Name     string
	Size     int64
	Children []DirEntry
}

func scanDir(path string, maxDepth, currentDepth int, processedSize *int64) (DirEntry, int64, error) {
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

		childEntry, skippedChild, err := scanDir(fullPath, maxDepth, currentDepth+1, processedSize)
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

func printEntry(e DirEntry, total int64, level int, maxDepth int) {
	indent := strings.Repeat("  ", level)
	percent := float64(e.Size) / float64(total) * 100

	// Color setup
	nameColor := color.New(color.FgHiCyan).SprintFunc()
	sizeColor := color.New(color.FgHiWhite).SprintFunc()
	icon := "📁"
	if len(e.Children) == 0 {
		icon = "📄"
	}

	// Dynamic percentage color
	var percentColor *color.Color
	switch {
	case percent <= 7:
		percentColor = color.New(color.FgGreen)
	case percent <= 30:
		percentColor = color.New(color.FgBlue)
	case percent <= 50:
		percentColor = color.New(color.FgMagenta)
	default:
		percentColor = color.New(color.FgRed)
	}

	name := fmt.Sprintf("%s %s", icon, nameColor(e.Name))
	size := fmt.Sprintf("%9s", formatSize(e.Size))
	percentStr := fmt.Sprintf("(%6.2f%%)", percent)

	fmt.Printf("%s%-30s %s %s\n", indent, name, sizeColor(size), percentColor.Sprint(percentStr))

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

var spinnerDone = make(chan bool)

func StartSpinner(label string, processedSize *int64) {
	symbols := []string{"|", "/", "-", "\\"}
	i := 0

	go func() {
		for {
			select {
			case <-spinnerDone:
				return
			default:
				size := atomic.LoadInt64(processedSize)
				fmt.Printf("\r%s %s Scanned: %s", color.YellowString(symbols[i%len(symbols)]), label, formatSize(size))
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
