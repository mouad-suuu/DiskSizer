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
func getUsageColor(percent float64) tcell.Color {
	switch {
	case percent <= 50:
		// Green (0,255,0) to Yellow (255,255,0)
		// Interpolate red from 0 to 255
		r := int(255 * (percent / 50)) // 0 → 255
		g := 255                       // constant
		return tcell.NewRGBColor(int32(r), int32(g), 0)

	case percent <= 80:
		// Yellow (255,255,0) to Orange (255,165,0)
		// Interpolate green from 255 → 165
		g := int(255 - ((percent - 50) / 30 * 90)) // 255 → 165
		return tcell.NewRGBColor(255, int32(g), 0)

	default:
		// Orange (255,165,0) to Red (255,0,0)
		// Interpolate green from 165 → 0
		g := int(165 * ((100 - percent) / 20)) // 165 → 0
		return tcell.NewRGBColor(255, int32(g), 0)
	}
}

// GetDiskStatsInteractive returns disk stats with clickable elements
func GetDiskStatsInteractive(textView *tview.TextView, app *tview.Application) string {
	// Set up clickable handler
	styling.InstallClickHandler(textView, app)

	var result strings.Builder

	result.WriteString(styling.CreateHeader("📊 System Storage Status"))
	result.WriteString("")

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
		}) + "   ")

		// Add usage details
		totalGB := fmtSize(usage.Total)
		usedGB := fmtSize(usage.Used)
		freeGB := fmtSize(usage.Free)

		progressBar := styling.CreateProgressBar(usedGB, totalGB, 40)
		result.WriteString(progressBar + "\n")
		usedColor := getUsageColor(usage.UsedPercent)
		freeColor := getUsageColor(100 - usage.UsedPercent)

		totalInfo := styling.CreateInfoText("Total", fmt.Sprintf("%.2f GB", totalGB), tcell.ColorWhite)
		usedInfo := styling.CreateInfoText("Used", fmt.Sprintf("%.2f GB (%.1f%%)", usedGB, usage.UsedPercent), usedColor)
		freeInfo := styling.CreateInfoText("Free", fmt.Sprintf("%.2f GB (%.1f%%)", freeGB, 100-usage.UsedPercent), freeColor)

		result.WriteString(totalInfo + "   |   ")
		result.WriteString(usedInfo + "   |   ")
		result.WriteString(freeInfo + "")
	}

	return result.String()
}

// Implement these functions to handle the actions
func exploreDirectory(path string, app *tview.Application) {
	// This function is called when a disk partition is clicked
	// It should update the app state to navigate to the selected directory
	// You can implement the actual navigation logic here
	// For example, you might want to call a function in the app package
	// that updates the tree view to show the selected directory
}
