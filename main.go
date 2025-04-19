package main

import (
	cli "DiskSizer/CLI"
	utils "DiskSizer/Utils"
	"fmt"
)
func main() {
	timer := utils.StartTimer()
	
	// Initialize the CLI application
	fmt.Println("Welcome to you storage monitoring service ");
	cli.Execute()
	fmt.Printf("Process time: %.3f s\n", timer.Elapsed())
	
}