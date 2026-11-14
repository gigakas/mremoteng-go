package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestLocalSource_List_DirectoriesFirstThenFilesByName(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"zeta", "alpha"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, file := range []string{"beta.txt", "alpha.txt"} {
		if err := os.WriteFile(filepath.Join(root, file), []byte("content"), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	entries, err := localSource{}.List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"alpha", "zeta", "alpha.txt", "beta.txt"}
	if len(entries) != len(want) {
		t.Fatalf("entries = %d, want %d", len(entries), len(want))
	}
	for i, name := range want {
		if entries[i].Name != name {
			t.Errorf("entry %d = %q, want %q", i, entries[i].Name, name)
		}
	}
	if !entries[0].IsDir || entries[2].IsDir {
		t.Errorf("kinds not reported: %+v", entries)
	}
	if entries[2].Size != int64(len("content")) {
		t.Errorf("file size = %d, want %d", entries[2].Size, len("content"))
	}
	if entries[2].Path != filepath.Join(root, "alpha.txt") {
		t.Errorf("path = %q, want it joined with the directory", entries[2].Path)
	}
}

func TestLocalSource_Parent_StopsAtTheRoot(t *testing.T) {
	if got := (localSource{}).Parent("/"); got != "/" {
		t.Errorf("Parent(/) = %q, want /", got)
	}
	if got := (localSource{}).Parent("/home/tester"); got != "/home" {
		t.Errorf("Parent(/home/tester) = %q, want /home", got)
	}
}

func TestFilePane_UpdateRow_FillsEachKindWithoutPanic(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	pane := newFilePane(localSource{})
	pane.entries = []fileEntry{
		{Name: "reports", IsDir: true, Modified: time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)},
		{Name: "notes.txt", Size: 2048, Modified: time.Date(2026, 11, 12, 9, 30, 0, 0, time.UTC)},
	}
	if got := pane.listLength(); got != 2 {
		t.Fatalf("listLength = %d, want 2", got)
	}

	row := pane.createRow().(*fileRow)
	pane.updateRow(0, row)
	if row.name.Text != "reports/" || row.size.Text != "" {
		t.Errorf("directory row = %q/%q, want a trailing slash and no size", row.name.Text, row.size.Text)
	}
	pane.updateRow(1, row)
	if row.name.Text != "notes.txt" || row.size.Text != "2.0 KiB" {
		t.Errorf("file row = %q/%q, want the name and a readable size", row.name.Text, row.size.Text)
	}
	if row.date.Text != "2026-11-12 09:30" {
		t.Errorf("date = %q, want the modification time", row.date.Text)
	}

	// The list recycles rows while the entry slice shrinks, so these must not panic.
	pane.updateRow(9, row)
	pane.updateRow(0, widget.NewLabel("not a row"))
}

func TestFilePane_Selection_DirectoryIsNavigationNotATransfer(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	pane := newFilePane(localSource{})
	pane.entries = []fileEntry{{Name: "reports", IsDir: true}, {Name: "notes.txt"}}
	pane.selected = -1
	if _, ok := pane.selection(); ok {
		t.Error("an empty selection reported a file")
	}
	pane.selected = 0
	if _, ok := pane.selection(); ok {
		t.Error("a directory reported itself as a transfer item")
	}
	pane.selected = 1
	entry, ok := pane.selection()
	if !ok || entry.Name != "notes.txt" {
		t.Errorf("selection = %+v/%v, want notes.txt", entry, ok)
	}
}
