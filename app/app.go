package app

import (
	Utils "DiskSizer/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
    app      *tview.Application
    flex     *tview.Flex
    treeView *tview.TreeView
)

func StartApp(startPath string) {
    app = tview.NewApplication()

    statsText := tview.NewTextView().
        SetText(Utils.GetDiskStats()).
        SetTextAlign(tview.AlignLeft).
        SetDynamicColors(true)

    treeView = tview.NewTreeView()

    footerText := tview.NewTextView().
        SetText("ENTER: Open | BACKSPACE: Back | Q: Quit").
        SetTextAlign(tview.AlignCenter).
        SetDynamicColors(true)

    flex = tview.NewFlex().
        SetDirection(tview.FlexRow).
        AddItem(statsText, 5, 1, false).
        AddItem(treeView, 0, 8, true).
        AddItem(footerText, 1, 1, false)

    // handle keys
    treeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Key() {
        case tcell.KeyEnter:
            openSelectedNode()
        case tcell.KeyBackspace, tcell.KeyBackspace2:
            navigateUp()
        case tcell.KeyRune:
            if event.Rune() == 'q' || event.Rune() == 'Q' {
                app.Stop()
            }
        }
        return event
    })

    // TODO: build tree view based on startPath

    if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
        panic(err)
    }
}

func openSelectedNode() {
    // TODO: Handle opening a folder
}

func navigateUp() {
    // TODO: Handle going up to parent
}
