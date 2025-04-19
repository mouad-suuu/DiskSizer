package cli

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/cobra"
)

// statCmd represents the stat command
var statCmd = &cobra.Command{
	Use:   "stat",
	Short: "Displays storage status",
	Run: func(cmd *cobra.Command, args []string) {
		stat() // call the stat function
	},
}

// stat is the function to gather and display the storage stats
func stat() {
	fmt.Println("The Status of the Storage in your device")
	partitions, err := disk.Partitions(true)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, p := range partitions {
		fmt.Println("Mountpoint:", p.Mountpoint)
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			fmt.Println("  Error:", err)
			continue
		}
		fmt.Printf("  Total: %.2f GB\n", float64(usage.Total)/1e9)
		fmt.Printf("  Free:  %.2f GB\n", float64(usage.Free)/1e9)
		fmt.Printf("  Used:  %.2f GB\n", float64(usage.Used)/1e9)
		fmt.Printf("  Used Percent: %.2f%%\n", usage.UsedPercent)
	}
}

func init() {
	// Add the stat command to the root command
	rootCmd.AddCommand(statCmd)
}
