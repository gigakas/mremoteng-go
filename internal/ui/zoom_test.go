package ui

import (
	"io"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

// rowHeight reports the height the grid renders one text row at, which is what
// the theme's text size ultimately controls.
func rowHeight(term *termWidget) float32 {
	rows := len(term.grid.Rows)
	if rows == 0 {
		return 0
	}
	return term.grid.MinSize().Height / float32(rows)
}

// TestInstallTerminal_ZoomThenActivity_KeepsTheTextSize guards the reported bug:
// after a zoom the terminal fell back to its previous size as soon as remote
// output arrived or the view scrolled. The cause was a second ThemeOverride per
// zoom: Fyne caches overrides per object, and a refresh of the orphaned first
// wrapper re-registered the old text size.
func TestInstallTerminal_ZoomThenActivity_KeepsTheTextSize(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	shell := &fakeShell{output: reader}
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		return shell, sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.applyConnect(panel.target(), sshclient.Options{}, shell,
		sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil)

	window := test.NewWindow(panel.content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(1000, 700))

	term := panel.term
	if term.gridWrap == nil {
		t.Fatal("connect did not install the terminal theme override")
	}
	firstGrid := term.grid
	before := rowHeight(term)

	term.adjustZoom(4)
	window.Resize(fyne.NewSize(1000, 700))
	zoomed := rowHeight(term)
	if zoomed <= before {
		t.Fatalf("zoom did not enlarge the text: before %.0f, after %.0f", before, zoomed)
	}
	if term.grid == firstGrid {
		t.Error("zoom kept the old text grid; TextGrid caches the theme size per row")
	}

	// Everything the user does after zooming must preserve the size.
	term.Refresh()
	panel.applyOutput([]byte("hello world\r\nsecond line\r\n"))
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 3)})
	if got := rowHeight(term); got != zoomed {
		t.Errorf("text size reverted after activity and scroll: want %.0f, got %.0f", zoomed, got)
	}

	// A second zoom step keeps working through the same wrapper.
	term.adjustZoom(2)
	window.Resize(fyne.NewSize(1000, 700))
	if got := rowHeight(term); got <= zoomed {
		t.Errorf("second zoom step did not enlarge the text: was %.0f, got %.0f", zoomed, got)
	}
}

// TestZoomFocusedTerminal_ShortcutPath_ChangesTextSize covers the path the
// shortcuts and the View menu take: the panel is found through the canvas
// focus, not passed directly.
func TestZoomFocusedTerminal_ShortcutPath_ChangesTextSize(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	shell := &fakeShell{output: reader}
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		return shell, sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.applyConnect(panel.target(), sshclient.Options{}, shell,
		sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil)

	s.win.SetContent(panel.content())
	s.win.Resize(fyne.NewSize(1000, 700))
	term := panel.term
	term.focus()
	if _, ok := s.focusedTerminal(); !ok {
		t.Fatal("the terminal does not own the canvas focus, so the shortcuts cannot reach it")
	}

	before := rowHeight(term)
	s.zoomFocusedTerminal(2)
	s.win.Resize(fyne.NewSize(1000, 700))
	if got := rowHeight(term); got <= before {
		t.Errorf("zoom in did nothing: before %.0f, after %.0f", before, got)
	}
	zoomed := rowHeight(term)

	s.zoomFocusedTerminal(-2)
	s.win.Resize(fyne.NewSize(1000, 700))
	if got := rowHeight(term); got >= zoomed {
		t.Errorf("zoom out did nothing: zoomed %.0f, after %.0f", zoomed, got)
	}

	s.resetFocusedTerminalZoom()
	s.win.Resize(fyne.NewSize(1000, 700))
	if got := rowHeight(term); got != before {
		t.Errorf("reset did not restore the base size: want %.0f, got %.0f", before, got)
	}
}

// TestScrolled_MovesThroughScrollbackWithoutChangingZoom guards the plain wheel:
// scrolling must browse the history, never touch the text size.
func TestScrolled_MovesThroughScrollbackWithoutChangingZoom(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	shell := &fakeShell{output: reader}
	s.openShell = func(sshclient.Target, sshclient.Options, sshclient.TermSize) (shellSession, sshclient.Result, error) {
		return shell, sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil
	}
	panel := s.newSessionPanel(sshPanelProfile(t, "tester"))
	panel.applyConnect(panel.target(), sshclient.Options{}, shell,
		sshclient.Result{ServerVersion: "SSH-2.0-test", HostKeyVerified: true}, nil)
	window := test.NewWindow(panel.content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(1000, 700))

	term := panel.term
	// Incremental output, because the emulator records scrollback by aligning
	// consecutive screens rather than scanning one large chunk.
	for i := 0; i < 120; i++ {
		panel.applyOutput([]byte("scrolled history\r\n"))
	}
	before := rowHeight(term)
	size := term.textSize()

	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 1)})
	if term.scrollOffset == 0 {
		t.Error("plain wheel did not move the view")
	}
	if term.textSize() != size || rowHeight(term) != before {
		t.Errorf("plain wheel changed the zoom: text %.0f row %.0f", term.textSize(), rowHeight(term))
	}
}
