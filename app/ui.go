package app

import (
	"DiskSizer/Utils"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// updateStats updates the disk stats view with interactive elements
func updateStats() {
	// Use the interactive stats
	statsText := Utils.GetDiskStatsInteractive(statsView, app)
	statsView.SetText(statsText)
}

// addDirEntryToNode adds a directory entry to a tree node
func addDirEntryToNode(node *tview.TreeNode, dirEntry Utils.DirEntry, path string) {
	// Sort children by size (larger files first)
	sort.Slice(dirEntry.Children, func(i, j int) bool {
		return dirEntry.Children[i].Size > dirEntry.Children[j].Size
	})

	// Add all the directory entries
	for _, child := range dirEntry.Children {
		isDir := len(child.Children) > 0
		childPath := filepath.Join(path, child.Name)

		var childNode *tview.TreeNode
		if isDir {
			childNode = tview.NewTreeNode(fmt.Sprintf("%s %s (%s)",
				Utils.GetFileIcon(child.Name, true),
				child.Name,
				Utils.FormatSize(child.Size))).SetReference(childPath).SetSelectable(true).SetColor(tcell.ColorGreen)
		} else {
			childNode = tview.NewTreeNode(fmt.Sprintf("%s %s (%s)",
				Utils.GetFileIcon(child.Name, false),
				child.Name,
				Utils.FormatSize(child.Size))).SetReference(childPath).SetSelectable(true).SetColor(tcell.ColorWhite)
		}

		node.AddChild(childNode)
	}
}

// findNodeByPath finds a tree node by path
func findNodeByPath(node *tview.TreeNode, targetPath string) *tview.TreeNode {
	ref := node.GetReference()
	if ref != nil && ref.(string) == targetPath {
		return node
	}

	for _, child := range node.GetChildren() {
		if result := findNodeByPath(child, targetPath); result != nil {
			return result
		}
	}

	return nil
}
