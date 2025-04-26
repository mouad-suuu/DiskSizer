package Utils

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"
)

func GetDiskStats() string {
    var result string

    partitions, err := disk.Partitions(true)
    if err != nil {
        return fmt.Sprintf("Error fetching partitions: %v", err)
    }

    for _, p := range partitions {
        usage, err := disk.Usage(p.Mountpoint)
        if err != nil {
            continue
        }

        result += fmt.Sprintf("\nMountpoint: %s\n", p.Mountpoint)
        result += fmt.Sprintf("  Total: %.2f GB\n", float64(usage.Total)/1e9)
        result += fmt.Sprintf("  Free: %.2f GB\n", float64(usage.Free)/1e9)
        result += fmt.Sprintf("  Used: %.2f GB\n", float64(usage.Used)/1e9)
        result += fmt.Sprintf("  Used space Percent: %.2f%%\n", usage.UsedPercent)
        result += fmt.Sprintf("  Free space Percent: %.2f%%\n", 100-usage.UsedPercent)
    }
    return result
}
