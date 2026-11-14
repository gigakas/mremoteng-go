package ui

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"fyne.io/fyne/v2"
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
	list      *widget.List
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
	pane.list = widget.NewList(pane.listLength, pane.createRow, pane.updateRow)
	pane.list.OnSelected = pane.rowSelected
	pane.list.OnUnselected = func(widget.ListItemID) {
		pane.selected = -1
		if pane.onSelect != nil {
			pane.onSelect(fileEntry{})
		}
	}
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
		p.list,
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
	p.list.UnselectAll()
	p.list.Refresh()
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

func (p *filePane) rowSelected(id widget.ListItemID) {
	if id < 0 || id >= len(p.entries) {
		return
	}
	entry := p.entries[id]
	if entry.IsDir {
		p.navigate(entry.Path)
		return
	}
	p.selected = int(id)
	if p.onSelect != nil {
		p.onSelect(entry)
	}
}

// listLength satisfies widget.List.
func (p *filePane) listLength() int { return len(p.entries) }

// createRow builds one reusable row. It is a typed struct rather than a bare
// container: container.NewBorder keeps its nil slots, so recovering widgets by
// index panics on the first empty slot.
func (p *filePane) createRow() fyne.CanvasObject {
	icon := widget.NewIcon(theme.FileIcon())
	name := widget.NewLabel("")
	size := widget.NewLabel("")
	size.Alignment = fyne.TextAlignTrailing
	date := widget.NewLabel("")
	date.Importance = widget.LowImportance
	box := container.NewBorder(nil, nil, container.NewCenter(icon), nil,
		container.NewBorder(nil, nil, name, container.NewHBox(size, date)))
	return &fileRow{Container: box, icon: icon, name: name, size: size, date: date}
}

// updateRow fills one row for the entry at id.
func (p *filePane) updateRow(id widget.ListItemID, object fyne.CanvasObject) {
	row, ok := object.(*fileRow)
	if !ok || id < 0 || id >= len(p.entries) {
		return
	}
	entry := p.entries[id]
	if entry.IsDir {
		row.icon.SetResource(theme.FolderIcon())
		row.name.SetText(entry.Name + "/")
		row.size.SetText("")
	} else {
		row.icon.SetResource(theme.FileIcon())
		row.name.SetText(entry.Name)
		row.size.SetText(formatSize(entry.Size))
	}
	row.date.SetText(formatModified(entry.Modified))
}

// fileRow is one listing row: a container plus the widgets the update step needs.
type fileRow struct {
	*fyne.Container
	icon *widget.Icon
	name *widget.Label
	size *widget.Label
	date *widget.Label
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
