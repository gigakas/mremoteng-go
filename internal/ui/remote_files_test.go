package ui

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestFormatSize_RendersReadableUnits(t *testing.T) {
	cases := map[int64]string{
		0:              "0 B",
		512:            "512 B",
		1024:           "1.0 KiB",
		1536:           "1.5 KiB",
		1048576:        "1.0 MiB",
		5 * 1073741824: "5.0 GiB",
		-1:             "?",
	}
	for input, want := range cases {
		if got := formatSize(input); got != want {
			t.Errorf("formatSize(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatModified_EmptyWhenUnknown(t *testing.T) {
	if got := formatModified(time.Time{}); got != "" {
		t.Errorf("formatModified(zero) = %q, want an empty string", got)
	}
	stamp := time.Date(2026, 11, 12, 9, 30, 0, 0, time.UTC)
	if got := formatModified(stamp); got != "2026-11-12 09:30" {
		t.Errorf("formatModified = %q, want the short date form", got)
	}
}

func TestProgressText_ReportsBytesAndPercentage(t *testing.T) {
	got := progressText("Uploading big.iso", 512, 1024)
	if !strings.Contains(got, "512 B of 1.0 KiB") || !strings.Contains(got, "(50%)") {
		t.Errorf("progressText = %q, want bytes and a percentage", got)
	}
	unknown := progressText("Downloading tree", 4096, 0)
	if !strings.Contains(unknown, "4.0 KiB") || strings.Contains(unknown, "%") {
		t.Errorf("progressText without a total = %q, want bytes and no percentage", unknown)
	}
}

func TestRemoteBrowser_Logf_CapsTheWindowLogAndMirrorsIt(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	screen := newTestScreen(t, &fakeStore{})
	browser := &remoteBrowser{screen: screen, logBox: container.NewVBox()}

	for i := 0; i < transferLogLimit+20; i++ {
		browser.logf("info", "transfer line")
	}

	if len(browser.log) != transferLogLimit {
		t.Errorf("window log = %d lines, want it capped at %d", len(browser.log), transferLogLimit)
	}
	if len(browser.logBox.Objects) != transferLogLimit {
		t.Errorf("log view = %d rows, want one per kept line", len(browser.logBox.Objects))
	}
	if len(screen.log.entries) == 0 {
		t.Error("the activity log received nothing from the transfer window")
	}
}

func TestRemoteBrowser_ProgressWithoutWidgets_DoesNotPanic(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	// A transfer can only start from the shown window, but the helpers must not
	// depend on that: a nil widget is a no-op, not a crash.
	browser := &remoteBrowser{screen: newTestScreen(t, &fakeStore{})}
	browser.startProgress("Uploading")
	browser.setProgress(1, 2)
	browser.finishProgress()
	browser.logf("info", "still alive")
}

func TestStopTransfer_WithoutRunningTransfer_DoesNothing(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	browser := &remoteBrowser{screen: newTestScreen(t, &fakeStore{}), status: widget.NewLabel("before")}
	browser.stopTransfer()

	if browser.status.Text != "before" {
		t.Errorf("status = %q, want it untouched when nothing is running", browser.status.Text)
	}
}

func TestStopTransfer_CancelsTheRunningCopy(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	browser := &remoteBrowser{screen: newTestScreen(t, &fakeStore{}), status: widget.NewLabel("")}
	cancelled := false
	browser.cancel = func() { cancelled = true }

	browser.stopTransfer()

	if !cancelled {
		t.Fatal("stopTransfer did not call the cancel function")
	}
	if browser.cancel != nil {
		t.Error("the cancel function was not cleared after stopping")
	}
	if !strings.Contains(browser.status.Text, "Stopping") {
		t.Errorf("status = %q, want a stopping notice", browser.status.Text)
	}
}

func TestRemoteBrowser_Close_IsIdempotent(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	browser := &remoteBrowser{screen: newTestScreen(t, &fakeStore{}), status: widget.NewLabel("")}
	// No session was ever opened here, and closing twice must not panic: a
	// browser is dismissed from more than one place.
	browser.close()
	browser.close()
	if browser.files != nil {
		t.Error("close left a session reference behind")
	}
}

func TestModal_Hide_ReturnsTheKeyboardToThePreviousObject(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	screen := newTestScreen(t, &fakeStore{})
	term := newTermWidget(80, 24, func([]byte) {}, nil)
	screen.win.SetContent(term)
	screen.win.Resize(fyne.NewSize(400, 300))
	term.focus()
	if _, ok := screen.win.Canvas().Focused().(*termWidget); !ok {
		t.Fatal("the terminal did not take the focus in the first place")
	}

	popup := screen.showModal(widget.NewLabel("dialog"), fyne.NewSize(300, 120))
	// Opening a modal popup takes the canvas focus; simulate that, because the
	// test driver does not move it by itself.
	screen.win.Canvas().Focus(widget.NewEntry())
	popup.Hide()

	if _, ok := screen.win.Canvas().Focused().(*termWidget); !ok {
		t.Error("hiding the dialog left the keyboard away from the terminal")
	}
	if popup.previous != nil {
		t.Error("the remembered focus was not cleared")
	}
}
