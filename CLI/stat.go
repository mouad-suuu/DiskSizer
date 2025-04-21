package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/cobra"
)

var statCmd = &cobra.Command{
	Use:   "stat",
	Short: "Displays storage status",
	Run: func(cmd *cobra.Command, args []string) {
		stat()
	},
}

func stat() {
	header := color.New(color.FgCyan, color.Bold).SprintFunc()
	section := color.New(color.FgHiYellow, color.Bold).SprintFunc()
	label := color.New(color.FgGreen).SprintFunc()
	value := color.New(color.FgWhite).SprintFunc()
	UsedPercent := color.New(color.FgRed, color.Bold).SprintFunc()
	FreePercent := color.New(color.FgGreen, color.Bold).SprintFunc()

	fmt.Println(header("\n📊 Storage Status"))
	fmt.Println(section("────────────────────────────"))

	partitions, err := disk.Partitions(true)
	if err != nil {
		color.Red("Error fetching partitions: %v\n", err)
		return
	}

	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			color.Yellow("  Skipping %s due to error: %v", p.Mountpoint, err)
			continue
		}

		fmt.Printf("\n%s %s\n", label("Mountpoint:"), value(p.Mountpoint))
		fmt.Printf("  %-10s %-3s %s\n", label("Total:"), fmtSize(usage.Total), "GB")
		fmt.Printf("  %s %-3s %s\n", label("Free:"), fmtSize(usage.Free), "GB")
		fmt.Printf("  %s %-3s %s\n", label("Used:"), fmtSize(usage.Used), "GB")
		fmt.Printf("  %s %s\n", label("Used space Percent:"), UsedPercent(fmt.Sprintf("%.2f%%", usage.UsedPercent)))
		fmt.Printf("  %s %s\n", label("Free space Percent:"), FreePercent(fmt.Sprintf("%.2f%%", ( 100 - usage.UsedPercent))))
	}
}

func fmtSize(size uint64) string {
	return fmt.Sprintf("%.2f", float64(size)/1e9)
}

func init() {
	rootCmd.AddCommand(statCmd)
}
