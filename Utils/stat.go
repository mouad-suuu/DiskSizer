package Utils

import (
	"DiskSizer/styling"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shirou/gopsutil/v3/disk"
)

func fmtSize(size uint64) float64 {
	return float64(size) / 1e9 // Convert to GB
}

// GetDiskStats returns the disk statistics as a string
func GetDiskStats() string {
	var result strings.Builder

	result.WriteString(styling.CreateHeader("📊 Storage Status"))
	result.WriteString("\n")

	partitions, err := disk.Partitions(true)
	if err != nil {
		errorStyle := styling.NewStyleBuilder().
			WithTextColor(tcell.ColorRed).
			Build()
		result.WriteString(styling.ApplyStyle(fmt.Sprintf("Error fetching partitions: %v\n", err), errorStyle))
		return result.String()
	}

	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		// Add mountpoint info
		mountStyle := styling.NewStyleBuilder().
			WithBold().
			WithTextColor(tcell.ColorYellow).
			Build()
		result.WriteString("\n" + styling.ApplyStyle(p.Mountpoint, mountStyle) + "\n")

		// Add usage details
		totalGB := fmtSize(usage.Total)
		usedGB := fmtSize(usage.Used)
		freeGB := fmtSize(usage.Free)

		// Create info texts
		totalInfo := styling.CreateInfoText("Total", fmt.Sprintf("%.2f GB", totalGB))
		usedInfo := styling.CreateInfoText("Used", fmt.Sprintf("%.2f GB (%.1f%%)", usedGB, usage.UsedPercent))
		freeInfo := styling.CreateInfoText("Free", fmt.Sprintf("%.2f GB (%.1f%%)", freeGB, 100-usage.UsedPercent))

		result.WriteString(totalInfo + "\n")
		result.WriteString(usedInfo + "\n")
		result.WriteString(freeInfo + "\n")

		// Add progress bar
		result.WriteString("\n")
		progressBar := styling.CreateProgressBar(usedGB, totalGB, 40)
		result.WriteString(progressBar + "\n")
	}

	return result.String()
}

// GetDiskStatsInteractive returns disk stats with clickable elements
func GetDiskStatsInteractive(textView *tview.TextView, app *tview.Application) string {
	// Set up clickable handler
	styling.InstallClickHandler(textView, app)

	var result strings.Builder

	result.WriteString(styling.CreateHeader("📊 Storage Status"))
	result.WriteString("\n")

	partitions, err := disk.Partitions(true)
	if err != nil {
		errorStyle := styling.NewStyleBuilder().
			WithTextColor(tcell.ColorRed).
			Build()
		result.WriteString(styling.ApplyStyle(fmt.Sprintf("Error fetching partitions: %v\n", err), errorStyle))
		return result.String()
	}

	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		// Make mountpoint clickable to explore that location
		mountpoint := p.Mountpoint
		result.WriteString("\n" + styling.WrapWithAction(textView, mountpoint, func() {
			// This will be called when the mountpoint is clicked
			// You'll need to implement the actual directory navigation logic
			exploreDirectory(mountpoint, app)
		}) + "\n")

		// Add usage details
		totalGB := fmtSize(usage.Total)
		usedGB := fmtSize(usage.Used)
		freeGB := fmtSize(usage.Free)

		// Create info texts
		totalInfo := styling.CreateInfoText("Total", fmt.Sprintf("%.2f GB", totalGB))
		usedInfo := styling.CreateInfoText("Used", fmt.Sprintf("%.2f GB (%.1f%%)", usedGB, usage.UsedPercent))
		freeInfo := styling.CreateInfoText("Free", fmt.Sprintf("%.2f GB (%.1f%%)", freeGB, 100-usage.UsedPercent))

		result.WriteString(totalInfo + "\n")
		result.WriteString(usedInfo + "\n")
		result.WriteString(freeInfo + "\n")

		// Add progress bar
		result.WriteString("\n")
		progressBar := styling.CreateProgressBar(usedGB, totalGB, 40)
		result.WriteString(progressBar + "\n")

		// Add an analyze button
		result.WriteString(styling.WrapWithAction(textView, "📊 Analyze Space Usage", func() {
			analyzeSpaceUsage(mountpoint, app)
		}) + "\n")
	}

	return result.String()
}

// Implement these functions to handle the actions
func exploreDirectory(path string, app *tview.Application) {
	// This would be implemented to navigate to the selected directory
	// You'll want to update your app state to show the selected directory
}

func analyzeSpaceUsage(path string, app *tview.Application) {
	// This would be implemented to show detailed space analysis for the path
	// Maybe display a modal or change to an analysis view
}
