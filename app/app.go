package app

import (
	cache "DiskSizer/Cache"
	"DiskSizer/styling"
	"os"
	"path/filepath"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Application global variables
var (
	// UI components
	app        *tview.Application
	flex       *tview.Flex
	treeView   *tview.TreeView
	statsView  *tview.TextView
	headerView *tview.TextView
	footerView *tview.TextView

	// State tracking - exported for use in other files
	CurrentPath   string // Exported for use in navigation.go
	processedSize int64
	ProcessedTime float64

	// Cache and scanning management
	dirCache   *cache.DirSizeCache
	scanCancel chan bool
	scanMutex  sync.Mutex
	isScanning bool
)

func StartApp(startPath string) {
	app = tview.NewApplication()
	dirCache = cache.NewDirSizeCache()
	scanCancel = make(chan bool, 1)

	// If no start path provided, use current directory
	if startPath == "" {
		var err error
		startPath, err = os.Getwd()
		if err != nil {
			startPath = "/"
		}
	}
	CurrentPath = startPath

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

	// Initialize processed size
	processedSize = 0

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
			CurrentPath = path
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
