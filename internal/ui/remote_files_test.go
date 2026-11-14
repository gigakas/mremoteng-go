package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
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

func TestRemoteBrowser_UpdateRow_FillsEachKindWithoutPanic(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	browser := &remoteBrowser{
		screen: newTestScreen(t, &fakeStore{}),
		entries: []sshclient.RemoteEntry{
			{Name: "reports", IsDir: true, Modified: time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)},
			{Name: "notes.txt", Size: 2048, Modified: time.Date(2026, 11, 12, 9, 30, 0, 0, time.UTC)},
		},
	}
	if got := browser.listLength(); got != 2 {
		t.Fatalf("listLength = %d, want 2", got)
	}

	row := browser.createRow().(*remoteRow)
	browser.updateRow(0, row)
	if row.name.Text != "reports/" || row.size.Text != "" {
		t.Errorf("directory row = %q/%q, want name with a trailing slash and no size", row.name.Text, row.size.Text)
	}
	browser.updateRow(1, row)
	if row.name.Text != "notes.txt" || row.size.Text != "2.0 KiB" {
		t.Errorf("file row = %q/%q, want the name and a readable size", row.name.Text, row.size.Text)
	}
	if row.date.Text != "2026-11-12 09:30" {
		t.Errorf("date = %q, want the modification time", row.date.Text)
	}

	// An out-of-range or foreign object must not panic: the list recycles rows
	// while the entries slice shrinks.
	browser.updateRow(9, row)
	browser.updateRow(0, widget.NewLabel("not a row"))
}
