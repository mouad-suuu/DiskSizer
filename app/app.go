package app

import (
	"DiskSizer/Cache"
	"DiskSizer/Utils"
	"DiskSizer/styling"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	app           *tview.Application
	flex          *tview.Flex
	treeView      *tview.TreeView
	statsView     *tview.TextView
	headerView    *tview.TextView
	footerView    *tview.TextView
	currentPath   string
	spinnerText   string
	processedSize int64
	ProcessedTime float64
	// New variables for enhanced performance
	dirCache    *Cache.DirSizeCache
	scanCancel  chan bool
	scanMutex   sync.Mutex
	isScanning  bool
	lastRefresh time.Time
)

func StartApp(startPath string) {
	app = tview.NewApplication()
	dirCache = Cache.NewDirSizeCache()
	scanCancel = make(chan bool, 1)

	// If no start path provided, use current directory
	if startPath == "" {
		var err error
		startPath, err = os.Getwd()
		if err != nil {
			startPath = "/"
		}
	}
	currentPath = startPath

	// Create styled header
	headerStyle := styling.NewStyleBuilder().
		WithBold().
		WithTextColor(tcell.ColorBlue).
		Build()
	headerText := styling.ApplyStyle("Welcome to DiskSizer, your tool to manage and organize your storage", headerStyle)

	headerView = tview.NewTextView().
		SetText(headerText).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Initialize spinner text
	spinnerText = ""

	// Create stats view with interactive elements
	statsView = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	// Install handler for clickable elements
	styling.InstallClickHandler(statsView, app)

	// Update stats with interactive elements
	updateStats()

	// Create tree view
	rootDir := filepath.Base(startPath)
	root := tview.NewTreeNode(rootDir).
		SetReference(startPath).
		SetSelectable(true).
		SetColor(tcell.ColorGreen)
	treeView = tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	// Add children for the root node
	addChildren(root)

	// Set up the tree handler
	treeView.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return
		}

		path := reference.(string)
		info, err := os.Stat(path)
		if err != nil {
			return
		}

		if info.IsDir() {
			// If it's collapsed, expand it
			if node.IsExpanded() {
				node.Collapse()
			} else {
				node.SetExpanded(true)
				if len(node.GetChildren()) == 0 {
					addChildren(node)
				}
			}
			currentPath = path
			updateStats()
		} else {
			// Handle file selection
			// You could show file details or other actions
		}
	})

	// Create styled footer with additional key mappings
	footerStyle := styling.NewStyleBuilder().
		WithTextColor(tcell.ColorGray).
		Build()
	footerText := styling.ApplyStyle("ENTER: Open/Collapse | BACKSPACE: Back | Q: Quit | SPACE: Refresh | C: Clear Cache", footerStyle)

	footerView = tview.NewTextView().
		SetText(footerText).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Create layout without separate progress view
	flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(headerView, 1, 0, false).
		AddItem(statsView, 7, 0, false).
		AddItem(treeView, 0, 1, true).
		AddItem(footerView, 1, 0, false)

	// Handle key events
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			navigateUp()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q', 'Q':
				app.Stop()
				return nil
			case ' ':
				updateStats()
				refreshCurrentDir()
				return nil
			case 'c', 'C':
				clearCache()
				return nil
			case 'e', 'E':
				// Quick estimate mode
				estimateCurrentDir()
				return nil
			case 's', 'S':
				// Stop current scan if running
				cancelScan()
				return nil
			}
		}
		return event
	})

	// Start the application
	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func updateStats() {
	// Use the new interactive stats
	statsText := Utils.GetDiskStatsInteractive(statsView, app)
	statsView.SetText(statsText)
}

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
			symbols := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
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
							spinnerText = fmt.Sprintf("%s[yellow] Scanning directory... [blue]Processed: %s [gray](%.1fs)",
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
					Utils.FormatSize(cachedEntry.Size), len(cachedEntry.Children))).
					SetSelectable(false).
					SetColor(tcell.ColorGreen)
				node.AddChild(statusNode)

				// Convert the Cache.DirEntry to Utils.DirEntry
				utilsEntry := Utils.ConvertFromCacheEntry(cachedEntry)
				addDirEntryToNode(node, utilsEntry, path)
			})
			return
		}

		// Perform the actual directory scan with new cached method
		dirEntry, skipped, err := Utils.CachedScanDir(path, 1, 0, &processedSize, dirCache)
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
				Utils.FormatSize(dirEntry.Size), len(dirEntry.Children), Utils.FormatSize(skipped), ProcessedTime)).
				SetSelectable(false).
				SetColor(tcell.ColorBlue)
			node.AddChild(statusNode)

			// Add directory entries to the node
			addDirEntryToNode(node, dirEntry, path)
		})
	}()
}

// Helper function to add directory entries to a node
func addDirEntryToNode(node *tview.TreeNode, dirEntry Utils.DirEntry, path string) {
	// Sort children by size (larger files first)
	sort.Slice(dirEntry.Children, func(i, j int) bool {
		return dirEntry.Children[i].Size > dirEntry.Children[j].Size
	})

	// Add all the directory entries
	for _, child := range dirEntry.Children {
		isDir := len(child.Children) > 0
		icon := getFileIcon(child.Name, isDir)

		// Format size with different colors based on relative size
		sizeColor := getSizeColor(child.Size, dirEntry.Size)
		sizeText := fmt.Sprintf("[%s](%s)[white]", sizeColor, Utils.FormatSize(child.Size))

		display := fmt.Sprintf("%s %s  %s", icon, child.Name, sizeText)
		childNode := tview.NewTreeNode(display).
			SetReference(filepath.Join(path, child.Name)).
			SetSelectable(true)

		if isDir {
			childNode.SetColor(tcell.ColorGreen)
		} else {
			childNode.SetColor(tcell.ColorWhite)
		}

		node.AddChild(childNode)
	}
}

// Helper function to get color based on relative size
func getSizeColor(size, totalSize int64) string {
	if totalSize == 0 {
		return "white"
	}

	ratio := float64(size) / float64(totalSize)

	switch {
	case ratio > 0.5:
		return "red"
	case ratio > 0.25:
		return "orange"
	case ratio > 0.1:
		return "yellow"
	default:
		return "green"
	}
}

func getFileIcon(filename string, isDir bool) string {
	if isDir {
		return "📁" // Folder icon for directories
	}

	ext := filepath.Ext(filename)
	switch ext {
	case ".go":
		return "🔷"
	case ".txt", ".md":
		return "📝"
	case ".jpg", ".png", ".gif":
		return "🖼️"
	case ".mp3", ".wav":
		return "🎵"
	case ".mp4", ".avi", ".mov":
		return "🎞️"
	case ".pdf":
		return "📕"
	case ".zip", ".tar", ".gz":
		return "📦"
	case ".exe", ".app":
		return "⚙️"
	default:
		return "📄"
	}
}

func navigateUp() {
	// Get current node
	node := treeView.GetCurrentNode()
	if node == nil {
		return
	}

	// Get the reference path
	ref := node.GetReference()
	if ref == nil {
		return
	}

	path := ref.(string)
	parentPath := filepath.Dir(path)

	// Don't go above the initial directory
	if parentPath == path {
		return
	}

	// Find the parent node
	root := treeView.GetRoot()
	parentNode := findNodeByPath(root, parentPath)

	if parentNode != nil {
		treeView.SetCurrentNode(parentNode)
		currentPath = parentPath
		updateStats()
	}
}

func findNodeByPath(node *tview.TreeNode, targetPath string) *tview.TreeNode {
	// Check if this is the node we're looking for
	ref := node.GetReference()
	if ref != nil && ref.(string) == targetPath {
		return node
	}

	// Check children
	for _, child := range node.GetChildren() {
		if found := findNodeByPath(child, targetPath); found != nil {
			return found
		}
	}

	return nil
}

func refreshCurrentDir() {
	// Cancel any ongoing scan first
	cancelScan()

	// Get current node
	node := treeView.GetCurrentNode()
	if node == nil {
		return
	}

	// Clear existing children
	node.ClearChildren()

	// Refresh the children
	addChildren(node)
	app.Draw()
}
