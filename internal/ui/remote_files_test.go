package ui

import (
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
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
