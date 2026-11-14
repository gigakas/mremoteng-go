package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
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

	pane := newFilePane(localSource{}, "Transfer")
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

func TestFilePane_SelectIndex_ShiftExtendsAndControlToggles(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	pane := newFilePane(localSource{}, "Transfer")
	pane.entries = []fileEntry{
		{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"},
	}

	// A plain click selects one row and sets the anchor.
	pane.selectIndex(1, 0)
	if got := len(pane.selection()); got != 1 {
		t.Fatalf("plain click selected %d entries, want 1", got)
	}

	// Shift extends from the anchor, inclusive, both directions.
	pane.selectIndex(3, fyne.KeyModifierShift)
	if got := len(pane.selection()); got != 3 {
		t.Errorf("shift click selected %d entries, want 3", got)
	}
	pane.selectIndex(0, fyne.KeyModifierShift)
	if got := len(pane.selection()); got != 2 {
		t.Errorf("shift back selected %d entries, want 2", got)
	}

	// Control toggles without disturbing the rest.
	pane.selectIndex(2, 0) // a plain click first, so the set is {c}
	pane.selectIndex(3, fyne.KeyModifierControl)
	if got := len(pane.selection()); got != 2 {
		t.Errorf("control click on an unselected row left %d entries, want 2", got)
	}
	pane.selectIndex(2, fyne.KeyModifierControl)
	chosen := pane.selection()
	if len(chosen) != 1 || chosen[0].Name != "d" {
		t.Errorf("control click on a selected row = %+v, want only d", chosen)
	}
}

func TestFilePane_Selection_ReturnsEveryChosenEntryInOrder(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	pane := newFilePane(localSource{}, "Transfer")
	pane.entries = []fileEntry{{Name: "reports", IsDir: true}, {Name: "notes.txt"}}
	pane.selectIndex(0, 0)
	pane.selectIndex(1, fyne.KeyModifierControl)

	chosen := pane.selection()
	if len(chosen) != 2 {
		t.Fatalf("selection = %d entries, want both", len(chosen))
	}
	if !chosen[0].IsDir || chosen[0].Name != "reports" {
		t.Errorf("first = %+v, want the directory first in listing order", chosen[0])
	}
	if chosen[1].Name != "notes.txt" {
		t.Errorf("second = %+v, want notes.txt", chosen[1])
	}
}

func TestSortFileEntries_KeepsDirectoriesFirstInEveryOrder(t *testing.T) {
	entries := []fileEntry{
		{Name: "b.txt", Size: 10, Modified: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{Name: "alpha", IsDir: true, Modified: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)},
		{Name: "a.txt", Size: 900, Modified: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
		{Name: "zeta", IsDir: true, Modified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	cases := []struct {
		key        sortKey
		descending bool
		want       []string
	}{
		{sortByName, false, []string{"alpha", "zeta", "a.txt", "b.txt"}},
		{sortByName, true, []string{"zeta", "alpha", "b.txt", "a.txt"}},
		{sortBySize, false, []string{"alpha", "zeta", "b.txt", "a.txt"}},
		{sortBySize, true, []string{"zeta", "alpha", "a.txt", "b.txt"}},
		{sortByDate, true, []string{"alpha", "zeta", "a.txt", "b.txt"}},
		{sortByDate, false, []string{"zeta", "alpha", "b.txt", "a.txt"}},
	}
	for _, tc := range cases {
		copied := append([]fileEntry(nil), entries...)
		sortFileEntries(copied, tc.key, tc.descending)
		got := make([]string, 0, len(copied))
		for _, entry := range copied {
			got = append(got, entry.Name)
		}
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("%v desc=%v = %v, want %v", tc.key, tc.descending, got, tc.want)
		}
		if !copied[0].IsDir || !copied[1].IsDir {
			t.Errorf("%v desc=%v put a file before the directories: %v", tc.key, tc.descending, got)
		}
	}
}

func TestFilePane_SortColumn_TogglesDirectionAndRefreshesHeader(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	pane := newFilePane(localSource{}, "Transfer")
	pane.entries = []fileEntry{
		{Name: "b.txt", Size: 10, Modified: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{Name: "a.txt", Size: 900, Modified: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
	}
	pane.header()

	pane.sortColumn(sortByDate)
	if pane.entries[0].Name != "a.txt" {
		t.Errorf("date order = %q first, want the newest", pane.entries[0].Name)
	}
	pane.sortColumn(sortByDate)
	if pane.entries[0].Name != "b.txt" {
		t.Errorf("second date click = %q first, want the oldest", pane.entries[0].Name)
	}
	pane.sortColumn(sortBySize)
	if pane.entries[0].Name != "a.txt" {
		t.Errorf("size order = %q first, want the largest", pane.entries[0].Name)
	}
	if got := pane.headerButtons[sortBySize].Text; !strings.Contains(got, "Size") || !strings.Contains(got, "\u25bc") {
		t.Errorf("size header = %q, want it marked as the active descending column", got)
	}
	if got := pane.headerButtons[sortByName].Text; got != "Name" {
		t.Errorf("inactive header = %q, want a plain label", got)
	}
	if len(pane.rows.Objects) != 2 {
		t.Errorf("rows = %d, want one per entry", len(pane.rows.Objects))
	}
}

func TestFileRow_Icons_DistinguishFoldersFromFiles(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	dirRow := newFileRow(fileEntry{Name: "reports", IsDir: true}, 0, nil)
	fileRow := newFileRow(fileEntry{Name: "notes.txt"}, 1, nil)

	if dirRow.icon.Resource == nil || fileRow.icon.Resource == nil {
		t.Fatal("a row has no icon")
	}
	if dirRow.icon.Resource.Name() == fileRow.icon.Resource.Name() {
		t.Error("folders and files show the same icon, so they cannot be told apart")
	}
	if !strings.Contains(dirRow.icon.Resource.Name(), "folder") {
		t.Errorf("folder row icon = %q, want a folder sample", dirRow.icon.Resource.Name())
	}
	if !strings.Contains(fileRow.icon.Resource.Name(), "file") {
		t.Errorf("file row icon = %q, want a document sample", fileRow.icon.Resource.Name())
	}
}

// TestColumns_HeaderAndRow_ShareTheSameOffsets is the regression guard for the
// reported misalignment: the header and a data row must place the numeric
// columns at identical x positions, otherwise a title does not sit over its data.
func TestColumns_HeaderAndRow_ShareTheSameOffsets(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	offsets := func(line fyne.CanvasObject) []float32 {
		line.Resize(fyne.NewSize(700, fileRowHeight))
		body, ok := line.(*fyne.Container)
		if !ok {
			t.Fatalf("line type = %T, want *fyne.Container", line)
		}
		// Border objects: the right-hand group holds the size and date slots.
		var group *fyne.Container
		for _, child := range body.Objects {
			if candidate, isHBox := child.(*fyne.Container); isHBox && len(candidate.Objects) == 2 {
				group = candidate
			}
		}
		if group == nil {
			t.Fatal("numeric column group not found")
		}
		group.Resize(group.MinSize())
		return []float32{group.Objects[0].Position().X, group.Objects[1].Position().X}
	}

	row := columns(widget.NewLabel(""), widget.NewLabel("name"), widget.NewLabel("size"), widget.NewLabel("date"))
	header := columns(widget.NewLabel(""), widget.NewButton("Name", nil), widget.NewButton("Size", nil), widget.NewButton("Date", nil))

	rowOffsets := offsets(row)
	headerOffsets := offsets(header)
	if rowOffsets[0] != headerOffsets[0] || rowOffsets[1] != headerOffsets[1] {
		t.Errorf("column offsets: row %v, header %v — want identical", rowOffsets, headerOffsets)
	}
}

func TestFileRow_MouseDown_SelectsTheRowWithItsModifiers(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)

	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	pane := newFilePane(localSource{}, "Transfer")
	window := test.NewWindow(pane.content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(600, 300))
	pane.navigate(root)
	if len(pane.rows.Objects) != 2 {
		t.Fatalf("rows = %d, want 2", len(pane.rows.Objects))
	}

	first := pane.rows.Objects[0].(*fileRow)
	first.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	if chosen := pane.selection(); len(chosen) != 1 {
		t.Fatalf("after a plain click the selection has %d entries, want 1", len(chosen))
	}

	// Control through the row, not through the pane, so the modifier path from
	// the mouse event to the selection is covered end to end.
	second := pane.rows.Objects[1].(*fileRow)
	second.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary, Modifier: fyne.KeyModifierControl})
	if chosen := pane.selection(); len(chosen) != 2 {
		t.Errorf("after Control+click the selection has %d entries, want 2", len(chosen))
	}

	// A click elsewhere during a drag, or the secondary button, must not select.
	third := &fileRow{index: 0, pane: pane}
	third.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonSecondary})
	if chosen := pane.selection(); len(chosen) != 2 {
		t.Errorf("secondary click changed the selection to %d entries", len(chosen))
	}
}
