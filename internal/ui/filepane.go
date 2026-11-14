package ui

import (
	"fmt"
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
	"fyne.io/fyne/v2/driver/desktop"
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
	// Rename gives an entry a new name in the same directory.
	Rename(from, to string) error
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

// Rename keeps local behaviour in step with the remote pane: it refuses to take
// a name that is already in use, where os.Rename would replace the file.
func (localSource) Rename(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("%s already exists", filepath.Base(to))
	}
	return os.Rename(from, to)
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

func (r remoteSource) Join(dir, name string) string { return sshclient.JoinRemote(dir, name) }

func (r remoteSource) Mkdir(dir string) error { return r.files.MkdirAll(dir) }

func (r remoteSource) Remove(target string) error { return r.files.RemoveAll(target) }

func (r remoteSource) Rename(from, to string) error { return r.files.Rename(from, to) }

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
	source     fileSource
	dir        string
	entries    []fileEntry
	selected   map[int]bool
	anchor     int
	order      sortKey
	descending bool

	pathEntry     *widget.Entry
	headerButtons map[sortKey]*widget.Button
	headerBox     fyne.CanvasObject
	rows          *fyne.Container
	scroll        *container.Scroll
	status        *widget.Label
	transferLabel string
	onTransfer    func()
	onSelect      func([]fileEntry)
	onEnter       func(string)
	onNewFolder   func()
	onDelete      func([]fileEntry)
	onRename      func([]fileEntry)
}

func newFilePane(source fileSource, transferLabel string) *filePane {
	pane := &filePane{source: source, selected: map[int]bool{}, anchor: -1,
		transferLabel: transferLabel, headerButtons: map[sortKey]*widget.Button{}}
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
	p.scroll = container.NewVScroll(p.rows)
	return container.NewBorder(
		container.NewVBox(heading, container.NewBorder(nil, nil, container.NewHBox(up, open), nil, p.pathEntry), p.header(), widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), p.status, container.NewHBox(
			widget.NewButtonWithIcon("New folder", theme.ContentAddIcon(), func() {
				if p.onNewFolder != nil {
					p.onNewFolder()
				}
			}),
			widget.NewButtonWithIcon("Rename", theme.DocumentCreateIcon(), func() {
				if p.onRename != nil {
					p.onRename(p.selection())
				}
			}),
			widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
				if entries := p.selection(); len(entries) > 0 && p.onDelete != nil {
					p.onDelete(entries)
				}
			}),
			widget.NewButtonWithIcon(p.transferLabel, theme.NavigateNextIcon(), func() {
				if p.onTransfer != nil {
					p.onTransfer()
				}
			}),
		)),
		nil, nil,
		p.scroll,
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
	p.selected = map[int]bool{}
	p.anchor = -1
	p.pathEntry.SetText(dir)
	p.rebuildRows()
	p.status.SetText(countLabel(len(entries), dir))
	if p.onEnter != nil {
		p.onEnter(dir)
	}
}

// refresh re-lists the current directory, used after a transfer.
func (p *filePane) refresh() { p.navigate(p.dir) }

// selection returns every chosen entry in listing order. Directories are transfer
// items too, because both directions copy trees recursively.
func (p *filePane) selection() []fileEntry {
	chosen := make([]fileEntry, 0, len(p.selected))
	for index, entry := range p.entries {
		if p.selected[index] {
			chosen = append(chosen, entry)
		}
	}
	return chosen
}

// sortKey is what the pane orders by within each group.
type sortKey int

const (
	sortByName sortKey = iota
	sortBySize
	sortByDate
)

// headerLabels are the column titles, in the order they are drawn.
var headerLabels = map[sortKey]string{
	sortByName: "Name",
	sortBySize: "Size",
	sortByDate: "Date",
}

// defaultDescending is the direction a column takes when it gains the order:
// names read A-Z, sizes and dates start from the largest and the newest.
func defaultDescending(key sortKey) bool { return key != sortByName }

// sortFileEntries orders entries the way the pane asked, keeping directories
// before files in every order: navigation must not depend on the sort.
func sortFileEntries(entries []fileEntry, key sortKey, descending bool) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		comparison := 0
		switch key {
		case sortBySize:
			switch {
			case entries[i].Size > entries[j].Size:
				comparison = 1
			case entries[i].Size < entries[j].Size:
				comparison = -1
			}
		case sortByDate:
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

// sortColumn orders by one column, toggling the direction when the same column is
// asked again. It re-sorts what is on screen instead of listing the directory.
func (p *filePane) sortColumn(key sortKey) {
	if p.order == key {
		p.descending = !p.descending
	} else {
		p.order = key
		p.descending = defaultDescending(key)
	}
	sortFileEntries(p.entries, p.order, p.descending)
	p.selected = map[int]bool{}
	p.anchor = -1
	p.rebuildRows()
	p.refreshHeader()
	p.status.SetText(countLabel(len(p.entries), p.dir))
}

// refreshHeader marks the active column and its direction.
func (p *filePane) refreshHeader() {
	for key, button := range p.headerButtons {
		label := headerLabels[key]
		if key == p.order {
			if p.descending {
				label += " \u25bc"
			} else {
				label += " \u25b2"
			}
		}
		button.SetText(label)
	}
}

// header builds the clickable column titles, aligned like a row: a gap where the
// icon sits, the name in the middle and the two numeric columns on the right.
func (p *filePane) header() fyne.CanvasObject {
	button := func(key sortKey) *widget.Button {
		label := headerLabels[key]
		button := widget.NewButton(label, func() { p.sortColumn(key) })
		button.Importance = widget.LowImportance
		button.Alignment = widget.ButtonAlignLeading
		p.headerButtons[key] = button
		return button
	}
	sizeButton := button(sortBySize)
	sizeButton.Alignment = widget.ButtonAlignTrailing
	dateButton := button(sortByDate)
	dateButton.Alignment = widget.ButtonAlignTrailing
	p.refreshHeader()
	return columns(widget.NewLabel(""), button(sortByName), sizeButton, dateButton)
}

// rebuildRows replaces the visible rows. Rows are built instead of recycled: a
// widget.List was tried first and its item templates stayed empty in the real
// window, which showed icons without names.
func (p *filePane) rebuildRows() {
	p.rows.Objects = p.rows.Objects[:0]
	p.selected = map[int]bool{}
	p.anchor = -1
	for i, entry := range p.entries {
		p.rows.Add(newFileRow(entry, i, p))
	}
	p.rows.Refresh()
	// The scroll keeps the previous folder's offset, so a new folder with fewer
	// entries opened halfway down its own list and had to be scrolled back up.
	if p.scroll != nil {
		p.scroll.ScrollToTop()
	}
}

// selectIndex applies one click to the selection. A plain click selects only that
// row, Shift extends from the anchor and Control or Super toggles the row, which
// is the range behaviour every file manager has.
func (p *filePane) selectIndex(index int, modifier fyne.KeyModifier) {
	if index < 0 || index >= len(p.entries) {
		return
	}
	switch {
	case modifier&fyne.KeyModifierShift != 0 && p.anchor >= 0:
		p.selected = map[int]bool{}
		low, high := p.anchor, index
		if low > high {
			low, high = high, low
		}
		for i := low; i <= high; i++ {
			p.selected[i] = true
		}
	case modifier&(fyne.KeyModifierControl|fyne.KeyModifierSuper) != 0:
		if p.selected[index] {
			delete(p.selected, index)
		} else {
			p.selected[index] = true
		}
		p.anchor = index
	default:
		p.selected = map[int]bool{index: true}
		p.anchor = index
	}
	p.paintSelection()
	if p.onSelect != nil {
		p.onSelect(p.selection())
	}
}

// paintSelection pushes the set into the rows so the highlight follows the click.
func (p *filePane) paintSelection() {
	for index, object := range p.rows.Objects {
		row, ok := object.(*fileRow)
		if !ok {
			continue
		}
		row.setSelected(p.selected[index])
	}
}

// rowOpened enters a directory on double tap; a file has nothing to open.
func (p *filePane) rowOpened(row *fileRow) {
	if row.entry.IsDir {
		p.navigate(row.entry.Path)
	}
}

// Column widths in pixels. The header and every row are laid out through
// columns(), so a title sits exactly over its data whatever the content is.
const (
	iconColumnWidth = 26
	sizeColumnWidth = 92
	dateColumnWidth = 130
	fileRowHeight   = 26
)

// columns lays out one line of a pane: a fixed icon slot, the name filling the
// rest, then the two fixed numeric columns. Both a data row and the header call
// it, which is what keeps them aligned.
func columns(icon, name, size, date fyne.CanvasObject) fyne.CanvasObject {
	iconSlot := container.NewGridWrap(fyne.NewSize(iconColumnWidth, fileRowHeight), icon)
	sizeSlot := container.NewGridWrap(fyne.NewSize(sizeColumnWidth, fileRowHeight), size)
	dateSlot := container.NewGridWrap(fyne.NewSize(dateColumnWidth, fileRowHeight), date)
	return container.NewBorder(nil, nil, container.NewCenter(iconSlot),
		container.NewHBox(sizeSlot, dateSlot), container.NewPadded(name))
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
	row.icon = widget.NewIcon(fileIcon())
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

// folderIcon is the folder sample tinted with the accent colour, so a directory
// is told apart from a file at a glance and not by reading the trailing slash.
// Resolved per row, not as a package variable: the theme accessors need a running
// Fyne application, and a package-level call logs an error before one exists.
func folderIcon() fyne.Resource { return theme.NewPrimaryThemedResource(theme.FolderIcon()) }

// fileIcon is the document sample in the normal foreground colour.
func fileIcon() fyne.Resource { return theme.NewThemedResource(theme.FileIcon()) }

// apply fills the row for entry.
func (r *fileRow) apply(entry fileEntry) {
	if entry.IsDir {
		r.icon.SetResource(folderIcon())
		r.name.SetText(entry.Name + "/")
		r.size.SetText("")
	} else {
		r.icon.SetResource(fileIcon())
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
	body := columns(r.icon, r.name, r.size, r.date)
	return widget.NewSimpleRenderer(container.NewStack(r.back, body))
}

// MouseDown selects the row, carrying the modifier so Shift and Control work.
// A plain click arrives here too, which is why the row no longer implements
// Tapped: that would fire again without modifiers and undo a Ctrl click.
func (r *fileRow) MouseDown(event *desktop.MouseEvent) {
	if r.pane == nil || event.Button != desktop.MouseButtonPrimary {
		return
	}
	r.pane.selectIndex(r.index, event.Modifier)
}

// MouseUp completes the desktop.Mouseable contract. Fyne requires both methods
// before it dispatches MouseDown at all, so without this the row received no
// click whatsoever.
func (r *fileRow) MouseUp(*desktop.MouseEvent) {}

// The rows must satisfy these interfaces or an entire class of clicks silently
// disappears; asserting them at compile time is the cheapest guard there is.
var (
	_ desktop.Mouseable   = (*fileRow)(nil)
	_ fyne.DoubleTappable = (*fileRow)(nil)
)

// DoubleTapped enters a directory.
func (r *fileRow) DoubleTapped(*fyne.PointEvent) {
	if r.pane != nil {
		r.pane.rowOpened(r)
	}
}

// MinSize keeps every row the same height regardless of its labels.
func (r *fileRow) MinSize() fyne.Size {
	return fyne.NewSize(120, fileRowHeight+2)
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
