package main

import (
	"DiskSizer/app"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
)

func main() {
	// Define command line flags

	var startPath string
	if len(os.Args) > 1 {
		startPath = os.Args[1]
	}

	enableProfiling := flag.Bool("profile", false, "Enable CPU profiling")
	// maxWorkers := flag.Int("workers", runtime.NumCPU(), "Maximum number of worker goroutines")
	// cacheSize := flag.Int("cache", 1000, "Maximum number of directories to cache")
	// enableFastScan := flag.Bool("fast", true, "Enable fast scan mode (uses sampling for large directories)")

	flag.Parse()

	// Enable CPU profiling if requested
	if *enableProfiling {
		f, err := os.Create("disksizer_cpu.prof")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create CPU profile: %v\n", err)
			return
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Could not start CPU profile: %v\n", err)
			return
		}
		defer pprof.StopCPUProfile()

		// Also capture memory profile
		memFile, err := os.Create("disksizer_mem.prof")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create memory profile: %v\n", err)
		} else {
			defer func() {
				runtime.GC() // Get up-to-date statistics
				if err := pprof.WriteHeapProfile(memFile); err != nil {
					fmt.Fprintf(os.Stderr, "Could not write memory profile: %v\n", err)
				}
				memFile.Close()
			}()
		}
	}

	// // Configure scanner based on command line arguments
	// Utils.SetGlobalScanConfig(Utils.ScanConfig{
	// 	MaxDepth:          0, // No depth limit
	// 	MaxWorkers:        *maxWorkers,
	// 	IgnoreHidden:      true,
	// 	IncludeSymlinks:   false,
	// 	SamplingThreshold: 1000,
	// })

	// // Initialize cache
	// Utils.InitializeCache(*cacheSize)

	// // Enable fast scan mode if requested
	// if *enableFastScan {
	// 	Utils.EnableFastScan()
	// }

	// Start the application
	app.StartApp(startPath)
}
