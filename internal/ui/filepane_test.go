package ui

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
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

func TestFilePane_Navigate_BuildsRowsWithNames(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "reports"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	pane := newFilePane(localSource{})
	window := test.NewWindow(pane.content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(600, 300))

	pane.navigate(root)

	if got := len(pane.rows.Objects); got != 2 {
		t.Fatalf("rows = %d, want one per entry", got)
	}
	first, ok := pane.rows.Objects[0].(*fileRow)
	if !ok {
		t.Fatalf("row type = %T, want *fileRow", pane.rows.Objects[0])
	}
	second := pane.rows.Objects[1].(*fileRow)

	// The reported bug: rows were drawn with an icon and no text.
	if first.name.Text != "reports/" || first.size.Text != "" {
		t.Errorf("directory row = %q/%q, want a trailing slash and no size", first.name.Text, first.size.Text)
	}
	if second.name.Text != "notes.txt" || second.size.Text != "5 B" {
		t.Errorf("file row = %q/%q, want the name and the readable size", second.name.Text, second.size.Text)
	}
	if second.date.Text == "" {
		t.Error("file row has no modification date")
	}
	if first.index != 0 || second.index != 1 {
		t.Errorf("row indexes = %d/%d, want 0/1", first.index, second.index)
	}
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
