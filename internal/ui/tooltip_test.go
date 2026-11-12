package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// shortHintDelay makes the hover delay test-sized and restores it afterwards.
func shortHintDelay(t *testing.T, delay time.Duration) {
	t.Helper()
	old := hintDelay
	hintDelay = delay
	t.Cleanup(func() { hintDelay = old })
}

func TestHintButton_Hover_ShowsTheHintAndRestoresTheStatus(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	shortHintDelay(t, 5*time.Millisecond)
	before := s.status.Text

	button := newHintButton(theme.InfoIcon(), "Show or hide the activity log", func() {}, s.hint)
	button.MouseIn(&desktop.MouseEvent{})

	deadline := time.Now().Add(2 * time.Second)
	for s.status.Text != "Show or hide the activity log" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := s.status.Text; got != "Show or hide the activity log" {
		t.Fatalf("status = %q, want the button hint", got)
	}
	if s.status.Importance != widget.LowImportance {
		t.Error("hint is not shown as secondary information")
	}

	button.MouseOut()
	if got := s.status.Text; got != before {
		t.Errorf("status = %q after leaving, want %q back", got, before)
	}
	if s.hintSaved != nil {
		t.Error("hint state was not cleared on leave")
	}
}

func TestHintButton_LeaveBeforeTheDelay_ShowsNothing(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	shortHintDelay(t, 80*time.Millisecond)
	before := s.status.Text

	button := newHintButton(theme.InfoIcon(), "Show or hide the activity log", func() {}, s.hint)
	button.MouseIn(&desktop.MouseEvent{})
	button.MouseOut()
	time.Sleep(150 * time.Millisecond)

	if got := s.status.Text; got != before {
		t.Errorf("status = %q, want the untuched %q", got, before)
	}
}

func TestHintButton_HoverWhileProblemIsShown_RestoresTheProblem(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	shortHintDelay(t, 5*time.Millisecond)
	s.setProblem("Could not save")

	button := newHintButton(theme.MenuIcon(), "Show or hide the connection tree", func() {}, s.hint)
	button.MouseIn(&desktop.MouseEvent{})
	deadline := time.Now().Add(2 * time.Second)
	for s.hintSaved == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	button.MouseOut()

	if got := s.status.Text; got != "Could not save" {
		t.Errorf("status = %q, want the problem restored", got)
	}
	if s.status.Importance != widget.DangerImportance {
		t.Error("restoring the problem lost its danger styling")
	}
}

func TestHintButton_Tap_StillRunsTheAction(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	taps := 0
	button := newHintButton(theme.InfoIcon(), "Toggle the log", func() { taps++ }, s.hint)

	button.Tapped(&fyne.PointEvent{})

	if taps != 1 {
		t.Errorf("taps = %d, want the action to run once", taps)
	}
}

func TestHint_EmptyTextWithoutPriorHint_DoesNothing(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	before := s.status.Text

	s.hint("")

	if got := s.status.Text; got != before {
		t.Errorf("status = %q, want %q untouched", got, before)
	}
}
