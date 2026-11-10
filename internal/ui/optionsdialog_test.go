package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/mRemoteNG/mremoteng-go/internal/settings"
	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

// TestShowOptionsDialog_DoesNotPanic is deliberately a smoke test, not a
// full interaction test: simulating a real form confirm/cancel click
// through Fyne's dialog package from a headless test would mean reaching
// into Fyne's own dialog internals rather than internal/ui's, for
// uncertain benefit — ShowOptionsDialog itself is a thin wrapper around
// dialog.ShowForm, which is Fyne's own tested code. What's worth
// confirming here is that building and showing the form (theme Select
// pre-populated, entry pre-filled) doesn't panic against a real target
// settings.Settings.
func TestShowOptionsDialog_DoesNotPanic(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	s := settings.Default()
	s.LastConnectionsFile = "C:/connections.xml"

	ui.ShowOptionsDialog(win, s, func(*settings.Settings) {})
}
