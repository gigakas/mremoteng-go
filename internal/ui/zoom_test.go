package ui

import (
	"io"
	"testing"
	"time"

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

// TestScrolled_CtrlHeld_ZoomsInsteadOfScrolling covers wheel zoom, which Fyne
// cannot detect on its own: ScrollEvent carries no modifiers, so the terminal
// asks the windowing system through the injectable ctrlHeld seam.
func TestScrolled_CtrlHeld_ZoomsInsteadOfScrolling(t *testing.T) {
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
	// The emulator records scrollback by aligning consecutive screens, so it needs
	// incremental output rather than one big chunk.
	fillHistory := func() {
		for i := 0; i < 120; i++ {
			panel.applyOutput([]byte("scrolled history\r\n"))
		}
	}
	fillHistory()
	before := rowHeight(term)

	term.ctrlHeld = func() bool { return true }
	term.lastZoom = time.Time{}
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 1)})
	zoomed := rowHeight(term)
	if zoomed <= before {
		t.Errorf("Ctrl+scroll did not zoom: before %.0f, after %.0f", before, zoomed)
	}
	if term.scrollOffset != 0 {
		t.Errorf("Ctrl+scroll moved the view: offset %d", term.scrollOffset)
	}

	// A second notch inside the throttle window is ignored, so a spinning wheel
	// does not rebuild the grid on every event.
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 1)})
	if got := rowHeight(term); got != zoomed {
		t.Errorf("notches inside the throttle window stacked: %.0f -> %.0f", zoomed, got)
	}

	// Releasing Ctrl returns the wheel to the scrollback. The grid refit during the
	// zoom drops the history, so fill it again first.
	fillHistory()
	term.ctrlHeld = func() bool { return false }
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 1)})
	if term.scrollOffset == 0 {
		t.Error("plain wheel did not move the view")
	}
	if got := rowHeight(term); got != zoomed {
		t.Errorf("plain wheel changed the text size: %.0f -> %.0f", zoomed, got)
	}
}
