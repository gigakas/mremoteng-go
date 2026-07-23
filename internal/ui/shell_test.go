package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// This file uses fyne.io/fyne/v2/test — Fyne's own headless test driver,
// which renders to an in-memory software canvas rather than a real
// window. No screenshot or visual check was possible while writing this
// package: launching a real Fyne window in this development environment
// produces a window with a valid, visible handle (confirmed via Win32
// APIs — IsWindowVisible=true, on-screen coordinates) that nonetheless
// never appears in a screenshot taken from the same session, for reasons
// not fully diagnosed (ruled out: different Windows session — same
// SessionId throughout; GL-specific compositing — same result with
// FYNE_RENDERER=software). So: everything below is verified through
// Fyne's headless driver and manual runs that check the process starts
// and doesn't crash (see scripts/smoke.sh) — never by looking at it. See
// the stage 3.1 audit for the full account.
//
// Tests live in package ui (not ui_test) because Shell's content tree
// (tree/tabs fields) has no public introspection API beyond Content(),
// which would need brittle type-assertions into fyne.Container.Objects to
// verify from outside — checking the fields directly is simpler and less
// coupled to Border's internal layout.

func TestNewShell_BuildsWindowWithMenuAndPlaceholders(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	s := NewShell(a)

	if s.Window == nil {
		t.Fatal("NewShell did not set Window")
	}
	if s.Window.Content() == nil {
		t.Fatal("NewShell did not set window content")
	}

	menu := s.Window.MainMenu()
	if menu == nil {
		t.Fatal("NewShell did not set a main menu")
	}
	wantMenus := []string{"File", "View", "Help"}
	if len(menu.Items) != len(wantMenus) {
		t.Fatalf("menu count = %d, want %d", len(menu.Items), len(wantMenus))
	}
	for i, want := range wantMenus {
		if got := menu.Items[i].Label; got != want {
			t.Errorf("menu[%d].Label = %q, want %q", i, got, want)
		}
	}

	if _, ok := s.tree.(*widget.Label); !ok {
		t.Errorf("tree placeholder = %T, want *widget.Label (stage 3.2 replaces this)", s.tree)
	}
	if _, ok := s.tabs.(*widget.Label); !ok {
		t.Errorf("tabs placeholder = %T, want *widget.Label (stage 3.3 replaces this)", s.tabs)
	}
	if _, ok := s.properties.(*widget.Label); !ok {
		t.Errorf("properties placeholder = %T, want *widget.Label (stage 3.4 replaces this)", s.properties)
	}
}

func TestNewShell_FileMenuHasQuitItem(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	s := NewShell(a)
	fileMenu := s.Window.MainMenu().Items[0]

	var quitItem *fyne.MenuItem
	for _, item := range fileMenu.Items {
		if item.IsQuit {
			quitItem = item
			break
		}
	}
	if quitItem == nil {
		t.Fatal("File menu has no Quit item (IsQuit=true)")
	}
	if quitItem.Action == nil {
		t.Error("Quit item has no Action")
	}
}

func TestShell_SetTree_ReplacesPlaceholder(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	s := NewShell(a)
	marker := widget.NewLabel("real tree (stage 3.2)")
	s.SetTree(marker)

	if s.tree != fyne.CanvasObject(marker) {
		t.Error("SetTree did not update the tree pane")
	}
}

func TestShell_SetTabs_ReplacesPlaceholder(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	s := NewShell(a)
	marker := widget.NewLabel("real tabs (stage 3.3)")
	s.SetTabs(marker)

	if s.tabs != fyne.CanvasObject(marker) {
		t.Error("SetTabs did not update the tabs pane")
	}
}

func TestShell_SetProperties_ReplacesPlaceholder(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	s := NewShell(a)
	marker := widget.NewLabel("real properties (stage 3.4)")
	s.SetProperties(marker)

	if s.properties != fyne.CanvasObject(marker) {
		t.Error("SetProperties did not update the properties pane")
	}
}
