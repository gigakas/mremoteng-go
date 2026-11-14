package ui

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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
	source   fileSource
	dir      string
	entries  []fileEntry
	selected int

	pathEntry *widget.Entry
	rows      *fyne.Container
	status    *widget.Label
	onSelect  func(fileEntry)
	onEnter   func(string)
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
	return container.NewBorder(
		container.NewVBox(heading, container.NewBorder(nil, nil, container.NewHBox(up, open), nil, p.pathEntry), widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), p.status),
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

// selection returns the chosen file, or false when a directory or nothing is
// selected. Directories are navigation, not transfer items.
func (p *filePane) selection() (fileEntry, bool) {
	if p.selected < 0 || p.selected >= len(p.entries) {
		return fileEntry{}, false
	}
	entry := p.entries[p.selected]
	if entry.IsDir {
		return fileEntry{}, false
	}
	return entry, true
}

// rebuildRows replaces the visible rows. Rows are built instead of recycled: a
// widget.List was tried first and its item templates stayed empty in the real
// window, which showed icons without names.
func (p *filePane) rebuildRows() {
	p.rows.Objects = p.rows.Objects[:0]
	for i, entry := range p.entries {
		row := newFileRow(entry, p.rowTapped)
		row.index = i
		p.rows.Add(row)
	}
	p.rows.Refresh()
}

// rowTapped enters a directory or selects a file.
func (p *filePane) rowTapped(entry fileEntry, id int) {
	if entry.IsDir {
		p.navigate(entry.Path)
		return
	}
	p.selected = id
	if p.onSelect != nil {
		p.onSelect(entry)
	}
}

// fileRow is one tappable line: icon, name, size and date. It is a widget so it
// can receive taps, and it holds its own labels so nothing is looked up by index.
type fileRow struct {
	widget.BaseWidget

	entry fileEntry
	index int
	icon  *widget.Icon
	name  *widget.Label
	size  *widget.Label
	date  *widget.Label
	onTap func(fileEntry, int)
}

func newFileRow(entry fileEntry, onTap func(fileEntry, int)) *fileRow {
	row := &fileRow{entry: entry, onTap: onTap}
	row.icon = widget.NewIcon(theme.FileIcon())
	row.name = widget.NewLabel("")
	row.size = widget.NewLabel("")
	row.size.Alignment = fyne.TextAlignTrailing
	row.date = widget.NewLabel("")
	row.date.Importance = widget.LowImportance
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

// CreateRenderer lays the row out: icon, then the name, then size and date.
func (r *fileRow) CreateRenderer() fyne.WidgetRenderer {
	iconBox := container.NewGridWrap(fyne.NewSize(18, 18), r.icon)
	tail := container.NewHBox(layout.NewSpacer(), r.size, r.date)
	body := container.NewBorder(nil, nil, r.name, tail)
	return widget.NewSimpleRenderer(
		container.NewBorder(nil, nil, container.NewCenter(iconBox), nil, body),
	)
}

// Tapped enters the directory or selects the file.
func (r *fileRow) Tapped(*fyne.PointEvent) {
	if r.onTap != nil {
		r.onTap(r.entry, r.index)
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
