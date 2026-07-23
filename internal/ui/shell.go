// Package ui implements the Fyne-based desktop application: the main
// window shell (this file), the connection tree (stage 3.2), session tabs
// (stage 3.3), and the remaining stage 3.x panels as they land.
//
// v1 uses a fixed layout (tree + tabs), not the original WinForms app's
// dockable-panel layout (WeifenLuo docking) — a known, deliberate UX
// regression recorded in the blueprint, revisited only if demanded.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// AppID is the Fyne application identifier, required for Preferences
// (stage 3.5's settings persistence) to work at all — omitting it makes
// Fyne log an error and silently disable the preferences API.
const AppID = "go.mremoteng.mremoteng"

// Shell is the main application window: a menu bar and a fixed two-pane
// layout (connection tree on the left, session tabs filling the rest).
// Stage 3.1 only builds the frame — Tree and Tabs are placeholders here,
// replaced by stage 3.2 and 3.3 respectively without changing Shell's
// structure.
type Shell struct {
	App    fyne.App
	Window fyne.Window

	// Tree and Tabs hold whatever stage 3.2/3.3 currently render there.
	// Exported so those stages can call SetTree/SetTabs without Shell
	// needing to know about the connection model or protocol backends —
	// keeping internal/ui's stage 3.1 code free of a dependency on
	// internal/connection or internal/protocol.
	tree fyne.CanvasObject
	tabs fyne.CanvasObject

	content *fyne.Container
}

// NewShell builds the main window: menu, layout, and default (placeholder)
// panes. Call Window.ShowAndRun to start the application — that call
// blocks until the window closes, so it belongs at the end of main(), not
// here (keeping NewShell itself synchronous and testable).
func NewShell(a fyne.App) *Shell {
	s := &Shell{App: a}
	s.Window = a.NewWindow("mRemoteNG")

	s.tree = widget.NewLabel("Connections")
	s.tabs = widget.NewLabel("No sessions open")
	s.content = container.NewBorder(nil, nil, s.tree, nil, s.tabs)

	s.Window.SetMainMenu(s.buildMenu())
	s.Window.SetContent(s.content)
	s.Window.Resize(fyne.NewSize(1024, 768))

	return s
}

// SetTree replaces the left-hand pane (the placeholder label from
// NewShell, or whatever was set before). Stage 3.2 calls this with the
// real connection tree widget.
func (s *Shell) SetTree(o fyne.CanvasObject) {
	s.tree = o
	s.rebuildContent()
}

// SetTabs replaces the main pane. Stage 3.3 calls this with the real
// session-tabs container.
func (s *Shell) SetTabs(o fyne.CanvasObject) {
	s.tabs = o
	s.rebuildContent()
}

func (s *Shell) rebuildContent() {
	s.content.Objects = []fyne.CanvasObject{container.NewBorder(nil, nil, s.tree, nil, s.tabs)}
	s.content.Refresh()
}

func (s *Shell) buildMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("New Connection", func() {}),       // wired in stage 3.2/3.4
		fyne.NewMenuItem("New Connections File", func() {}), // wired once internal/serialize save paths are reachable from the UI
		fyne.NewMenuItemSeparator(),
		&fyne.MenuItem{Label: "Quit", IsQuit: true, Action: s.App.Quit},
	)

	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Connections", func() {}), // toggles the tree pane once 3.2 lands
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			dialog.ShowInformation("About mRemoteNG",
				"mRemoteNG — Go/Fyne migration\nhttps://github.com/mRemoteNG/mRemoteNG",
				s.Window)
		}),
	)

	return fyne.NewMainMenu(fileMenu, viewMenu, helpMenu)
}
