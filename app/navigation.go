package app

import (
	"path/filepath"
)

// navigateUp moves up one directory in the tree view
func navigateUp() {
	currentNode := treeView.GetCurrentNode()
	if currentNode == nil {
		return
	}

	currentRef := currentNode.GetReference()
	if currentRef == nil {
		return
	}

	path := currentRef.(string)
	parentPath := filepath.Dir(path)

	// If already at root, do nothing
	if parentPath == path {
		return
	}

	// Find parent node
	rootNode := treeView.GetRoot()
	if rootNode == nil {
		return
	}

	parentNode := findNodeByPath(rootNode, parentPath)
	if parentNode != nil {
		treeView.SetCurrentNode(parentNode)
		CurrentPath = parentPath
		app.QueueUpdateDraw(func() {
			updateStats()
		})
	}
}

// refreshCurrentDir refreshes the current directory view
func refreshCurrentDir() {
	currentNode := treeView.GetCurrentNode()
	if currentNode == nil {
		return
	}

	// Remove all children of the current node
	currentNode.ClearChildren()

	// Add new children with fresh scan
	addChildren(currentNode)
}
