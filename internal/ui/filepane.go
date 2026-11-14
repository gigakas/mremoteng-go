package ui

import (
	"image/color"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	sshclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/ssh"
)

// fileEntry is one row in either pane: a local or a remote item, normalised so
// the two sides share the list, the sorting and the formatting.
type fileEntry struct {
	Name     string
	Path     string
	Size     int64
	Modified time.Time
	IsDir    bool
}

// fileSource is a navigable tree. The local pane and the remote pane differ only
// in how they list a directory and how they spell paths, so the widget below is
// written once.
type fileSource interface {
	// Title is the pane heading, for example the host name.
	Title() string
	// Initial returns the directory to open first.
	Initial() (string, error)
	// List returns one directory, directories first then files, each by name.
	List(dir string) ([]fileEntry, error)
	// Parent returns the directory above dir, stopping at the root.
	Parent(dir string) string
	// Join builds a child path the way this source spells paths.
	Join(dir, name string) string
	// Mkdir creates a directory, including missing parents.
	Mkdir(dir string) error
	// Remove deletes a file, or a directory with everything inside it.
	Remove(target string) error
}

// localSource reads the local filesystem.
type localSource struct{}

func (localSource) Title() string { return "Local" }

func (localSource) Initial() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func (localSource) List(dir string) ([]fileEntry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]fileEntry, 0, len(items))
	for _, item := range items {
		info, infoErr := item.Info()
		if infoErr != nil {
			continue
		}
		entries = append(entries, fileEntry{
			Name:     item.Name(),
			Path:     filepath.Join(dir, item.Name()),
			Size:     info.Size(),
			Modified: info.ModTime(),
			IsDir:    item.IsDir(),
		})
	}
	sortEntries(entries)
	return entries, nil
}

func (localSource) Join(dir, name string) string { return filepath.Join(dir, name) }

func (localSource) Mkdir(dir string) error { return os.MkdirAll(dir, 0o755) }

func (localSource) Remove(target string) error { return os.RemoveAll(target) }

func (localSource) Parent(dir string) string {
	cleaned := filepath.Clean(dir)
	parent := filepath.Dir(cleaned)
	if parent == cleaned {
		return cleaned
	}
	return parent
}

// remoteSource reads a directory over an open SFTP session.
type remoteSource struct {
	files *sshclient.RemoteFiles
	host  string
}

func (r remoteSource) Title() string { return r.host }

func (r remoteSource) Initial() (string, error) { return r.files.WorkingDir() }

func (r remoteSource) List(dir string) ([]fileEntry, error) {
	items, err := r.files.List(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]fileEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, fileEntry{
			Name:     item.Name,
			Path:     sshclient.JoinRemote(dir, item.Name),
			Size:     item.Size,
			Modified: item.Modified,
			IsDir:    item.IsDir,
		})
	}
	sortEntries(entries)
	return entries, nil
}

func (r remoteSource) Parent(dir string) string { return sshclient.ParentDir(dir) }

func (r remoteSource) Join(dir, name string) string { return sshclient.JoinRemote(dir, name) }

func (r remoteSource) Mkdir(dir string) error { return r.files.MkdirAll(dir) }

func (r remoteSource) Remove(target string) error { return r.files.RemoveAll(target) }

// sortEntries puts directories first, then files, each group by name.
func sortEntries(entries []fileEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
}

// filePane is one side of the browser: a path bar, a list and the current
// selection. It knows nothing about transfers, which the dialog orchestrates.
type filePane struct {
	source      fileSource
	dir         string
	entries     []fileEntry
	selected    int
	selectedRow *fileRow
	order       sortKey
	descending  bool

	pathEntry   *widget.Entry
	orderSelect *widget.Select
	rows        *fyne.Container
	status      *widget.Label
	onSelect    func(fileEntry)
	onEnter     func(string)
	onNewFolder func()
	onDelete    func(fileEntry)
}

func newFilePane(source fileSource) *filePane {
	pane := &filePane{source: source, selected: -1}
	pane.pathEntry = widget.NewEntry()
	pane.pathEntry.OnSubmitted = func(text string) { pane.navigate(text) }
	pane.status = widget.NewLabel("")
	pane.status.Importance = widget.LowImportance
	pane.rows = container.NewVBox()
	return pane
}

// content lays out the pane: heading, path bar and list above a status line.
func (p *filePane) content() fyne.CanvasObject {
	heading := widget.NewLabelWithStyle(p.source.Title(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	up := widget.NewButtonWithIcon("Up", theme.MoveUpIcon(), func() { p.navigate(p.source.Parent(p.dir)) })
	open := widget.NewButton("Open", func() { p.navigate(p.pathEntry.Text) })
	p.orderSelect = widget.NewSelect(orderChoices, func(choice string) { p.applyOrder(choice) })
	p.orderSelect.SetSelected(orderChoices[0])
	return container.NewBorder(
		container.NewVBox(heading, container.NewBorder(nil, nil, container.NewHBox(up, open), p.orderSelect, p.pathEntry), widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), p.status, container.NewHBox(
			widget.NewButtonWithIcon("New folder", theme.ContentAddIcon(), func() {
				if p.onNewFolder != nil {
					p.onNewFolder()
				}
			}),
			widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
				if entry, ok := p.selection(); ok && p.onDelete != nil {
					p.onDelete(entry)
				}
			}),
		)),
		nil, nil,
		container.NewVScroll(p.rows),
	)
}

// start opens the source's initial directory.
func (p *filePane) start() {
	dir, err := p.source.Initial()
	if err != nil {
		p.status.SetText("Could not open the starting directory: " + err.Error())
		return
	}
	p.navigate(dir)
}

// navigate lists dir and keeps the path bar in step.
func (p *filePane) navigate(dir string) {
	dir = trimDir(dir)
	if dir == "" {
		dir = p.dir
	}
	entries, err := p.source.List(dir)
	if err != nil {
		p.status.SetText("Could not list " + dir + ": " + err.Error())
		return
	}
	sortFileEntries(entries, p.order, p.descending)
	p.dir = dir
	p.entries = entries
	p.selected = -1
	p.pathEntry.SetText(dir)
	p.rebuildRows()
	p.status.SetText(countLabel(len(entries), dir))
	if p.onEnter != nil {
		p.onEnter(dir)
	}
}

// refresh re-lists the current directory, used after a transfer.
func (p *filePane) refresh() { p.navigate(p.dir) }

// selection returns the chosen entry, file or directory: a directory is a
// transfer item too, because both directions copy trees recursively.
func (p *filePane) selection() (fileEntry, bool) {
	if p.selected < 0 || p.selected >= len(p.entries) {
		return fileEntry{}, false
	}
	return p.entries[p.selected], true
}

// sortKey is what the pane orders by within each group.
type sortKey int

const (
	sortByName sortKey = iota
	sortByDate
)

// orderChoices are the labels of the pane order control, one per key and
// direction so a single control says everything.
var orderChoices = []string{
	"Name A-Z",
	"Name Z-A",
	"Date newest",
	"Date oldest",
}

// parseOrder maps a control label back to a key and a direction.
func parseOrder(choice string) (sortKey, bool) {
	switch choice {
	case "Name Z-A":
		return sortByName, true
	case "Date newest":
		return sortByDate, true
	case "Date oldest":
		return sortByDate, false
	default:
		return sortByName, false
	}
}

// sortFileEntries orders entries the way the pane asked, keeping directories
// before files in every order: navigation must not depend on the sort.
func sortFileEntries(entries []fileEntry, key sortKey, descending bool) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		comparison := 0
		if key == sortByDate {
			switch {
			case entries[i].Modified.After(entries[j].Modified):
				comparison = 1
			case entries[i].Modified.Before(entries[j].Modified):
				comparison = -1
			}
		}
		if comparison == 0 {
			left, right := strings.ToLower(entries[i].Name), strings.ToLower(entries[j].Name)
			switch {
			case left < right:
				comparison = -1
			case left > right:
				comparison = 1
			}
		}
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
}

// applyOrder re-sorts what is on screen without listing the directory again.
func (p *filePane) applyOrder(choice string) {
	p.order, p.descending = parseOrder(choice)
	sortFileEntries(p.entries, p.order, p.descending)
	p.selected = -1
	p.selectedRow = nil
	p.rebuildRows()
	p.status.SetText(countLabel(len(p.entries), p.dir))
}

// rebuildRows replaces the visible rows. Rows are built instead of recycled: a
// widget.List was tried first and its item templates stayed empty in the real
// window, which showed icons without names.
func (p *filePane) rebuildRows() {
	p.rows.Objects = p.rows.Objects[:0]
	p.selected = -1
	p.selectedRow = nil
	for i, entry := range p.entries {
		p.rows.Add(newFileRow(entry, i, p))
	}
	p.rows.Refresh()
}

// rowTapped selects one row and tells the dialog, which is what the transfer
// buttons act on. Entering a directory is a double tap, so a folder can be
// selected and seen as selected.
func (p *filePane) rowTapped(row *fileRow) {
	if p.selectedRow != nil && p.selectedRow != row {
		p.selectedRow.setSelected(false)
	}
	p.selected = row.index
	p.selectedRow = row
	row.setSelected(true)
	if p.onSelect != nil {
		p.onSelect(row.entry)
	}
}

// rowOpened enters a directory on double tap; a file has nothing to open.
func (p *filePane) rowOpened(row *fileRow) {
	if row.entry.IsDir {
		p.navigate(row.entry.Path)
	}
}

// fileRow is one clickable line: icon, name, size and date. It is a widget so it
// can receive taps, it holds its own labels so nothing is looked up by index, and
// it draws a selection background so the user can see what a transfer will move.
type fileRow struct {
	widget.BaseWidget

	entry  fileEntry
	index  int
	pane   *filePane
	icon   *widget.Icon
	name   *widget.Label
	size   *widget.Label
	date   *widget.Label
	back   *canvas.Rectangle
	picked bool
}

func newFileRow(entry fileEntry, index int, pane *filePane) *fileRow {
	row := &fileRow{entry: entry, index: index, pane: pane}
	row.icon = widget.NewIcon(theme.FileIcon())
	row.name = widget.NewLabel("")
	// A long name truncates instead of running under the size and the date.
	row.name.Truncation = fyne.TextTruncateEllipsis
	row.size = widget.NewLabel("")
	row.size.Alignment = fyne.TextAlignTrailing
	row.date = widget.NewLabel("")
	row.date.Importance = widget.LowImportance
	row.back = canvas.NewRectangle(color.Transparent)
	row.apply(entry)
	row.ExtendBaseWidget(row)
	return row
}

// apply fills the row for entry.
func (r *fileRow) apply(entry fileEntry) {
	if entry.IsDir {
		r.icon.SetResource(theme.FolderIcon())
		r.name.SetText(entry.Name + "/")
		r.size.SetText("")
	} else {
		r.icon.SetResource(theme.FileIcon())
		r.name.SetText(entry.Name)
		r.size.SetText(formatSize(entry.Size))
	}
	r.date.SetText(formatModified(entry.Modified))
}

// setSelected paints or clears the selection background.
func (r *fileRow) setSelected(on bool) {
	if r.picked == on {
		return
	}
	r.picked = on
	if on {
		r.back.FillColor = theme.Color(theme.ColorNameSelection)
	} else {
		r.back.FillColor = color.Transparent
	}
	r.back.Refresh()
}

// CreateRenderer lays the row out: icon, the name in the middle so it can
// truncate, then size and date on the right.
func (r *fileRow) CreateRenderer() fyne.WidgetRenderer {
	iconBox := container.NewGridWrap(fyne.NewSize(18, 18), r.icon)
	tail := container.NewHBox(r.size, r.date)
	body := container.NewBorder(nil, nil, container.NewCenter(iconBox),
		container.NewPadded(tail), container.NewPadded(r.name))
	return widget.NewSimpleRenderer(container.NewStack(r.back, body))
}

// Tapped selects the row.
func (r *fileRow) Tapped(*fyne.PointEvent) {
	if r.pane != nil {
		r.pane.rowTapped(r)
	}
}

// DoubleTapped enters a directory.
func (r *fileRow) DoubleTapped(*fyne.PointEvent) {
	if r.pane != nil {
		r.pane.rowOpened(r)
	}
}

// MinSize keeps every row the same height regardless of its labels.
func (r *fileRow) MinSize() fyne.Size {
	return fyne.NewSize(120, 26)
}

// trimDir normalises a typed path without inventing one.
func trimDir(dir string) string {
	if dir == "" {
		return ""
	}
	// path.Clean handles both spellings; the local pane re-cleans with filepath.
	return path.Clean(dir)
}

// countLabel is the pane status line.
func countLabel(entries int, dir string) string {
	if entries == 1 {
		return "1 entry in " + dir
	}
	return itoa(entries) + " entries in " + dir
}
