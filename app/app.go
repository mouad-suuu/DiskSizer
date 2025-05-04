package app

import (
	"DiskSizer/Utils"
	"DiskSizer/styling"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	app         *tview.Application
	flex        *tview.Flex
	treeView    *tview.TreeView
	statsView   *tview.TextView
	headerView  *tview.TextView
	footerView  *tview.TextView
	currentPath string
)

func StartApp(startPath string) {
	app = tview.NewApplication()

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

	// Create styled footer
	footerStyle := styling.NewStyleBuilder().
		WithTextColor(tcell.ColorGray).
		Build()
	footerText := styling.ApplyStyle("ENTER: Open/Collapse | BACKSPACE: Back | Q: Quit | SPACE: Refresh", footerStyle)

	footerView = tview.NewTextView().
		SetText(footerText).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Create layout with progress view at the top
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

func addChildren(node *tview.TreeNode) {
	path := node.GetReference().(string)
	var processedSize int64

	go func() {
		app.QueueUpdateDraw(func() {
			// Start the spinner and show it on the progressView at the top
			// scanner := Utils.StartSpinner(&processedSize)
			// progressView.SetText(scanner)
		})

		dirEntry, _, err := Utils.ScanDir(path, 1, 0, &processedSize)
		if err != nil {
			app.QueueUpdateDraw(func() {
				statsView.SetText("[red]Error scanning path")
				// progressView.SetText("")
			})
			return
		}

		app.QueueUpdateDraw(func() {
			node.ClearChildren()

			// Sort children by size (larger files first)
			sort.Slice(dirEntry.Children, func(i, j int) bool {
				return dirEntry.Children[i].Size > dirEntry.Children[j].Size
			})

			for _, child := range dirEntry.Children {
				icon := getFileIcon(child.Name, len(child.Children) > 0)
				display := fmt.Sprintf("%s %s  [gray](%s)", icon, child.Name, Utils.FormatSize(child.Size))
				childNode := tview.NewTreeNode(display).
					SetReference(filepath.Join(path, child.Name)).
					SetSelectable(true)

				if len(child.Children) > 0 {
					childNode.SetColor(tcell.ColorGreen)
				} else {
					childNode.SetColor(tcell.ColorWhite)
				}

				node.AddChild(childNode)
			}
			// progressView.SetText("Scanned: " + Utils.FormatSize(processedSize))
		})
	}()
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
	// Get current node
	node := treeView.GetCurrentNode()
	if node == nil {
		return
	}

	// Refresh the children
	addChildren(node)
	app.Draw()
}
