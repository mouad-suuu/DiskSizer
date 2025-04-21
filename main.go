package main

import (
	cli "DiskSizer/CLI"
	utils "DiskSizer/Utils"
	"fmt"

	"github.com/fatih/color"
)

func main() {
	timer := utils.StartTimer()

	// Fancy welcome message
	welcome := color.New(color.FgHiBlue, color.Bold).SprintFunc()
	fmt.Println(welcome("\n🚀 Welcome to DiskSizer, your storage monitoring service"))

	cli.Execute()

	elapsed := timer.Elapsed()
	color.New(color.FgHiMagenta).Printf("⏱️  Process time: %.3f s\n", elapsed)
}
