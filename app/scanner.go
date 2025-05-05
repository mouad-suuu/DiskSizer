package app

import (
	"fmt"
	"time"

	cache "DiskSizer/Cache"
	"DiskSizer/Utils"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// addChildren adds children to a tree node
func addChildren(node *tview.TreeNode) {
	scanMutex.Lock()
	// If another scan is in progress, cancel it
	if isScanning {
		select {
		case scanCancel <- true:
			// Signal sent
		default:
			// Channel already has a signal
		}
	}

	// Reset the cancel channel
	select {
	case <-scanCancel:
		// Clear any existing signal
	default:
		// Channel already empty
	}

	isScanning = true
	scanMutex.Unlock()

	path := node.GetReference().(string)

	// Create a spinner node that will appear at the top of the tree
	spinnerNode := tview.NewTreeNode("").SetSelectable(false)
	node.AddChild(spinnerNode)

	go func() {
		defer func() {
			scanMutex.Lock()
			isScanning = false
			scanMutex.Unlock()
		}()

		// Start the spinner
		stopSpinner := make(chan bool)
		spinnerActive := true

		// Run the spinner in a separate goroutine
		go func() {
			symbols := Utils.GetSpinnerChars()
			i := 0
			startTime := time.Now()
			ProcessedTime = 0
			for {
				select {
				case <-stopSpinner:
					return
				case <-scanCancel:
					app.QueueUpdateDraw(func() {
						spinnerNode.SetText("[yellow]Scan cancelled")
					})
					return
				default:
					if spinnerActive {
						elapsed := time.Since(startTime).Seconds()
						ProcessedTime = elapsed
						app.QueueUpdateDraw(func() {
							spinnerText := fmt.Sprintf("%s[yellow] Scanning directory... [blue]Processed: %s [gray](%.1fs)",
								symbols[i%len(symbols)], Utils.FormatSize(processedSize), elapsed)
							spinnerNode.SetText(spinnerText)
						})
					}
					time.Sleep(100 * time.Millisecond)
					i++
				}
			}
		}()

		// First check if we have this in the cache
		if cachedEntry, found := dirCache.Get(path); found {
			// Use the cached data instead of rescanning
			spinnerActive = false
			close(stopSpinner)

			app.QueueUpdateDraw(func() {
				// Remove the spinner node
				node.RemoveChild(spinnerNode)

				// Add a status node showing this is from cache
				statusNode := tview.NewTreeNode(fmt.Sprintf("[green]From Cache: %s - %d items[/green]",
					Utils.FormatSize(cachedEntry.Size), len(cachedEntry.Children))).SetSelectable(false).SetColor(tcell.ColorGreen)
				node.AddChild(statusNode)

				// Convert the Cache.DirEntry to Utils.DirEntry
				utilsEntry := cache.ToUtilsDirEntry(cachedEntry)
				addDirEntryToNode(node, utilsEntry, path)
			})
			return
		}

		// Perform the actual directory scan with cached method
		dirEntry, skipped, err := cache.CachedScanDir(path, 1, 0, &processedSize, dirCache)
		spinnerActive = false
		close(stopSpinner)

		select {
		case <-scanCancel:
			app.QueueUpdateDraw(func() {
				// Remove the spinner node
				node.RemoveChild(spinnerNode)
				// Add cancelled node
				cancelledNode := tview.NewTreeNode("[yellow]Scan cancelled").SetSelectable(false)
				node.AddChild(cancelledNode)
			})
			return
		default:
			// Continue processing
		}

		if err != nil {
			app.QueueUpdateDraw(func() {
				statsView.SetText("[red]Error scanning path")
				// Remove spinner node
				node.RemoveChild(spinnerNode)
				// Add error node
				errorNode := tview.NewTreeNode("[red]Error scanning directory").SetSelectable(false)
				node.AddChild(errorNode)
			})
			return
		}

		app.QueueUpdateDraw(func() {
			// Remove the spinner node
			node.RemoveChild(spinnerNode)

			// Add a status node at the top showing scan results
			statusNode := tview.NewTreeNode(fmt.Sprintf("[blue]Scanned: %s - %d items (Skipped: %s) [gray](%.1fs)",
				Utils.FormatSize(dirEntry.Size), len(dirEntry.Children), Utils.FormatSize(skipped), ProcessedTime)).SetSelectable(false).SetColor(tcell.ColorBlue)
			node.AddChild(statusNode)

			// Add directory entries to the node
			addDirEntryToNode(node, dirEntry, path)
		})
	}()
}

// clearCache clears the directory cache
func clearCache() {
	// Cancel any running scan
	cancelScan()

	// Clear the directory cache
	dirCache.Clear()

	// Update UI
	app.QueueUpdateDraw(func() {
		statsView.SetText("[yellow]Cache cleared. Press SPACE to refresh your view.")
	})
}

// cancelScan cancels a running directory scan
func cancelScan() {
	scanMutex.Lock()
	defer scanMutex.Unlock()

	if isScanning {
		select {
		case scanCancel <- true:
			// Signal sent
		default:
			// Channel already has a signal
		}

		isScanning = false
		app.QueueUpdateDraw(func() {
			statsView.SetText("[yellow]Scan cancelled. Press SPACE to restart scan.")
		})
	}
}

// estimateCurrentDir performs a quick size estimation of the current directory
func estimateCurrentDir() {
	node := treeView.GetCurrentNode()
	if node == nil {
		return
	}

	path := node.GetReference().(string)

	// Start the estimation in the background
	go func() {
		app.QueueUpdateDraw(func() {
			statsView.SetText("[yellow]Estimating directory size...")
		})

		// Get a quick estimate using sampling
		size, err := Utils.EstimateDirectorySize(path, 20)

		app.QueueUpdateDraw(func() {
			if err != nil {
				statsView.SetText("[red]Error estimating directory size")
			} else {
				statsView.SetText(fmt.Sprintf("[green]Estimated Size: %s[white]\n(This is an approximate value based on sampling)",
					Utils.FormatSize(size)))
			}
		})
	}()
}
