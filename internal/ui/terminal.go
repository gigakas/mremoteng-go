package ui

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/mRemoteNG/mremoteng-go/internal/terminal"
)

// termWidget renders a terminal emulator inside a tab and forwards keystrokes
// to the session. Modifier keys (Ctrl/Alt) are not delivered by the Fyne key
// API, so the session panel exposes Ctrl-C and Ctrl-D as buttons.
type termWidget struct {
	widget.BaseWidget

	emulator *terminal.Emulator
	grid     *widget.TextGrid
	send     func([]byte)
	onResize func(cols, rows int)
	styles   map[uint32]widget.TextGridStyle
	cols     int
	rows     int

	// Mouse selection: dragging highlights cells, and "copy screen" copies the
	// selection when there is one.
	selecting bool
	selStart  cellPos
	selEnd    cellPos
	// dragX/dragY accumulate the pointer delta so the selection start can be
	// derived even when the driver sends Dragged without a preceding MouseDown.
	dragX float32
	dragY float32

	// scrollOffset is how many buffer lines the view is scrolled up.
	scrollOffset int
	// onCopy is called with the text to copy (right click, shortcut).
	onCopy func(string)
	// onInterrupt sends Ctrl-C to the session.
	onInterrupt func()
	// autoFocus asks the first layout pass to take the keyboard: focusing before
	// the widget is rendered is dropped by the driver.
	autoFocus bool
	// fontSize is the terminal-only text size (Ctrl+scroll zoom).
	fontSize float32
	// zoom is the grid-level theme override; the grid owns its text size, so a
	// refresh (typing, scroll, selection) cannot fall back to the app theme.
	zoom *zoomTheme
	// renderer is the termRenderer built in CreateRenderer, so the grid can be
	// swapped in place on a zoom step.
	renderer *termRenderer
}

const (
	minFontSize     = 8
	maxFontSize     = 28
	defaultFontSize = 12
)

// textSize returns the size used for cell measurements.
func (t *termWidget) textSize() float32 {
	if t.fontSize <= 0 {
		return theme.TextSize()
	}
	return t.fontSize
}

// adjustZoom changes the terminal text size, clamped to a readable range.
func (t *termWidget) adjustZoom(delta float32) {
	size := t.textSize() + delta
	if size < minFontSize {
		size = minFontSize
	}
	if size > maxFontSize {
		size = maxFontSize
	}
	if size == t.textSize() {
		return
	}
	t.fontSize = size
	if t.zoom != nil {
		t.zoom.size = size
	}
	t.rebuildGrid()
	t.fit(t.Size())
	t.render()
}

func newTermWidget(cols, rows int, send func([]byte), onResize func(int, int)) *termWidget {
	t := &termWidget{
		emulator: terminal.New(cols, rows),
		fontSize: theme.TextSize(),
		send:     send,
		onResize: onResize,
		styles:   map[uint32]widget.TextGridStyle{},
		cols:     cols,
		rows:     rows,
	}
	t.zoom = &zoomTheme{base: fyne.CurrentApp().Settings().Theme(), size: t.textSize()}
	t.ExtendBaseWidget(t)
	return t
}

func (t *termWidget) CreateRenderer() fyne.WidgetRenderer {
	t.renderer = &termRenderer{parent: t, font: t.textSize()}
	t.renderer.objects = []fyne.CanvasObject{t.installGrid(fyne.NewSize(1, 1))}
	t.render()
	return t.renderer
}

// installGrid builds a text grid wrapped in the terminal's own theme override,
// so the grid reads the terminal size on every refresh instead of the app theme.
func (t *termWidget) installGrid(size fyne.Size) fyne.CanvasObject {
	grid := widget.NewTextGrid()
	grid.Scroll = fyne.ScrollNone
	if t.zoom == nil {
		t.zoom = &zoomTheme{base: fyne.CurrentApp().Settings().Theme(), size: t.textSize()}
	}
	t.grid = grid
	wrap := container.NewThemeOverride(grid, t.zoom)
	wrap.Resize(size)
	return wrap
}

// rebuildGrid swaps the text grid for a fresh one carrying the current size.
func (t *termWidget) rebuildGrid() {
	if t.renderer != nil {
		t.renderer.font = t.textSize()
		t.renderer.objects[0] = t.installGrid(t.Size())
	}
	t.render()
}

type termRenderer struct {
	parent  *termWidget
	objects []fyne.CanvasObject
	// font is the text size the current grid was built with; TextGrid caches its
	// row height, so a zoom change needs a fresh grid to scale both axes.
	font float32
}

func (r *termRenderer) Layout(size fyne.Size) {
	if r.font != r.parent.textSize() || r.parent.grid == nil {
		r.font = r.parent.textSize()
		r.parent.rebuildGrid()
		r.parent.fit(size)
		r.parent.render()
	} else {
		r.objects[0].Resize(size)
	}
	if r.parent.autoFocus {
		r.parent.autoFocus = false
		r.parent.focus()
	}
}
func (r *termRenderer) MinSize() fyne.Size           { return fyne.NewSize(160, 80) }
func (r *termRenderer) Refresh()                     { r.objects[0].Refresh() }
func (r *termRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *termRenderer) Destroy()                     {}

// Resize keeps the emulator grid in step with the available pixels.
func (t *termWidget) Resize(size fyne.Size) {
	t.BaseWidget.Resize(size)
	t.fit(size)
}

// traceResize records geometry changes when MRE_TRACE_RESIZE is set, so a
// layout oscillation can be diagnosed without a debugger.
func traceResize(size fyne.Size, cols, rows int) {
	if os.Getenv("MRE_TRACE_RESIZE") == "" {
		return
	}
	file, err := os.OpenFile("/tmp/mremoteng-resize.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	fmt.Fprintf(file, "size=%.0fx%.0f cols=%d rows=%d\n", size.Width, size.Height, cols, rows)
}

func (t *termWidget) fit(size fyne.Size) {
	cell := fyne.MeasureText("M", t.textSize(), fyne.TextStyle{Monospace: true})
	if cell.Width <= 0 || cell.Height <= 0 {
		return
	}
	padding := theme.Padding()
	cols := int((size.Width - 2*padding) / cell.Width)
	rows := int((size.Height - 2*padding) / cell.Height)
	cols = clampInt(cols, 20, 240)
	rows = clampInt(rows, 5, 90)
	if cols == t.cols && rows == t.rows {
		return
	}
	traceResize(size, cols, rows)
	t.cols, t.rows = cols, rows
	t.emulator.Resize(cols, rows)
	if t.onResize != nil {
		t.onResize(cols, rows)
	}
	t.render()
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func (t *termWidget) Tapped(*fyne.PointEvent) {
	t.focus()
}

// focus gives the terminal the keyboard.
func (t *termWidget) focus() {
	if canvas := fyne.CurrentApp().Driver().CanvasForObject(t); canvas != nil {
		canvas.Focus(t)
	}
}

// cellPos is a selection endpoint in character cells.
type cellPos struct {
	col int
	row int
}

// cellAt maps a widget position to a buffer cell.
func (t *termWidget) cellAt(position fyne.Position) cellPos {
	cell := fyne.MeasureText("M", t.textSize(), fyne.TextStyle{Monospace: true})
	if cell.Width <= 0 || cell.Height <= 0 {
		return cellPos{}
	}
	col := int(position.X / cell.Width)
	row := int(position.Y / cell.Height)
	_, start := t.viewStart()
	return cellPos{col: clampInt(col, 0, t.cols-1), row: start + clampInt(row, 0, t.rows-1)}
}

// viewStart returns the buffer lines and the index of the first visible line.
func (t *termWidget) viewStart() ([]string, int) {
	lines := t.emulator.Lines()
	start := len(lines) - t.rows - t.scrollOffset
	if start < 0 {
		start = 0
	}
	return lines, start
}

// Scrolled moves the view through the scrollback.
func (t *termWidget) Scrolled(event *fyne.ScrollEvent) {
	if event.Scrolled.DY == 0 {
		return
	}
	maxOffset := t.emulator.HistoryLen()
	t.scrollOffset += int(event.Scrolled.DY)
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}
	if t.scrollOffset > maxOffset {
		t.scrollOffset = maxOffset
	}
	t.render()
}

// MouseDown starts a selection with the primary button.
func (t *termWidget) MouseDown(event *desktop.MouseEvent) {
	if event.Button != desktop.MouseButtonPrimary {
		return
	}
	t.focus()
	t.selStart = t.cellAt(event.Position)
	t.selEnd = t.selStart
	t.selecting = true
	t.dragX, t.dragY = 0, 0
	t.render()
}

// TappedSecondary copies the current selection, like a Linux terminal, and
// clears the highlight so the copied block is obvious.
func (t *termWidget) TappedSecondary(*fyne.PointEvent) {
	if _, _, ok := t.selectionRange(); !ok {
		return
	}
	text := t.selectedText()
	if t.onCopy != nil {
		t.onCopy(text)
	}
	t.clearSelection()
}

// clearSelection drops the highlight.
func (t *termWidget) clearSelection() {
	t.selStart = cellPos{}
	t.selEnd = cellPos{}
	t.selecting = false
	t.dragX, t.dragY = 0, 0
	t.render()
}

// TypedShortcut turns the control shortcuts Fyne recognises into terminal
// control bytes, because key events carry no modifier information.
func (t *termWidget) TypedShortcut(shortcut fyne.Shortcut) {
	switch typed := shortcut.(type) {
	case *fyne.ShortcutCopy:
		if t.onInterrupt != nil {
			t.onInterrupt()
		}
	case *fyne.ShortcutCut:
		t.sendBytes([]byte{controlByte(fyne.KeyX)})
	case *fyne.ShortcutSelectAll:
		t.sendBytes([]byte{controlByte(fyne.KeyA)})
	case *fyne.ShortcutUndo:
		t.sendBytes([]byte{controlByte(fyne.KeyZ)})
	case *fyne.ShortcutRedo:
		t.sendBytes([]byte{controlByte(fyne.KeyY)})
	case *fyne.ShortcutPaste:
		clipboard := fyne.CurrentApp().Clipboard()
		if clipboard == nil || t.send == nil {
			return
		}
		if content := clipboard.Content(); content != "" {
			t.sendBytes([]byte(content))
		}
	case *desktop.CustomShortcut:
		if typed.Modifier&fyne.KeyModifierControl == 0 {
			return
		}
		if code, ok := controlCodes[typed.KeyName]; ok {
			t.sendBytes([]byte{code})
		}
	}
}

// controlCodes maps a key to its Ctrl byte.
var controlCodes = func() map[fyne.KeyName]byte {
	codes := map[fyne.KeyName]byte{
		fyne.KeySpace: 0x00,
	}
	for letter := byte('a'); letter <= 'z'; letter++ {
		codes[fyne.KeyName(strings.ToUpper(string(letter)))] = letter - 'a' + 1
	}
	codes[fyne.KeyName("[")] = 0x1b
	codes[fyne.KeyName("\\")] = 0x1c
	codes[fyne.KeyName("]")] = 0x1d
	codes[fyne.KeyName("^")] = 0x1e
	codes[fyne.KeyName("_")] = 0x1f
	return codes
}()

// controlByte returns the Ctrl byte for a letter, or zero when unknown.
func controlByte(key fyne.KeyName) byte {
	return controlCodes[key]
}

// Dragged extends the selection. When the driver reports a drag without a
// preceding MouseDown, the start cell is reconstructed from the pointer delta.
func (t *termWidget) Dragged(event *fyne.DragEvent) {
	t.dragX += event.Dragged.DX
	t.dragY += event.Dragged.DY
	if !t.selecting {
		start := fyne.NewPos(event.Position.X-t.dragX, event.Position.Y-t.dragY)
		t.selStart = t.cellAt(start)
		t.selecting = true
	}
	t.extendSelection(event.Position)
	t.render()
}

// extendSelection updates the drag end, scrolling the view when the pointer goes
// past the top or bottom edge so long outputs can be selected in one drag.
func (t *termWidget) extendSelection(position fyne.Position) {
	cell := fyne.MeasureText("M", t.textSize(), fyne.TextStyle{Monospace: true})
	if cell.Width <= 0 || cell.Height <= 0 {
		return
	}
	col := clampInt(int(position.X/cell.Width), 0, t.cols-1)
	history := t.emulator.HistoryLen()
	switch {
	case position.Y < 0:
		steps := 1 + int(-position.Y/cell.Height)
		t.scrollOffset = clampInt(t.scrollOffset+steps, 0, history)
		_, start := t.viewStart()
		t.selEnd = cellPos{col: col, row: start}
	case position.Y > t.Size().Height:
		steps := 1 + int((position.Y-t.Size().Height)/cell.Height)
		t.scrollOffset = clampInt(t.scrollOffset-steps, 0, history)
		_, start := t.viewStart()
		t.selEnd = cellPos{col: col, row: start + t.rows - 1}
	default:
		t.selEnd = t.cellAt(position)
	}
}

// DragEnd finishes the selection.
func (t *termWidget) DragEnd() {
	t.selecting = false
	t.dragX, t.dragY = 0, 0
	t.render()
}

// DoubleTapped selects the whole line under the pointer.
func (t *termWidget) DoubleTapped(event *fyne.PointEvent) {
	cell := t.cellAt(event.Position)
	t.selStart = cellPos{col: 0, row: cell.row}
	t.selEnd = cellPos{col: t.cols - 1, row: cell.row}
	t.render()
}

// selectionRange returns the normalised selection bounds.
func (t *termWidget) selectionRange() (cellPos, cellPos, bool) {
	if t.selStart == t.selEnd {
		return cellPos{}, cellPos{}, false
	}
	start, end := t.selStart, t.selEnd
	if start.row > end.row || (start.row == end.row && start.col > end.col) {
		start, end = end, start
	}
	return start, end, true
}

// selected reports whether a cell is inside the selection.
func (t *termWidget) selected(col, row int) bool {
	start, end, ok := t.selectionRange()
	if !ok {
		return false
	}
	if row < start.row || row > end.row {
		return false
	}
	if start.row == end.row {
		return col >= start.col && col <= end.col
	}
	if row == start.row {
		return col >= start.col
	}
	if row == end.row {
		return col <= end.col
	}
	return true
}

func (t *termWidget) FocusGained() {}
func (t *termWidget) FocusLost()   {}

// AcceptsTab keeps Tab inside the terminal: Fyne otherwise moves focus to the
// next widget, which breaks shell completion. Shift-Tab also arrives as Tab
// because key events carry no modifier information.
func (t *termWidget) AcceptsTab() bool { return true }

func (t *termWidget) TypedRune(r rune) {
	t.sendBytes([]byte(string(r)))
}

func (t *termWidget) TypedKey(event *fyne.KeyEvent) {
	// While the view is scrolled up, arrows and paging move through the
	// scrollback instead of reaching the shell; at the bottom they behave
	// normally, so command history and cursor keys keep working.
	if t.scrollThroughHistory(event.Name) {
		return
	}
	if sequence, known := keySequences[event.Name]; known {
		t.sendBytes(sequence)
	}
}

// scrollThroughHistory handles navigation keys while browsing history. It
// reports whether the key was consumed.
func (t *termWidget) scrollThroughHistory(name fyne.KeyName) bool {
	history := t.emulator.HistoryLen()
	switch name {
	case fyne.KeyPageUp:
		// PageUp always browses the scrollback; the plain up arrow keeps its
		// shell meaning while the view is at the bottom.
		t.scrollOffset = clampInt(t.scrollOffset+t.rows, 0, history)
		t.render()
		return true
	case fyne.KeyUp:
		if t.scrollOffset == 0 {
			return false
		}
		t.scrollOffset = clampInt(t.scrollOffset+1, 0, history)
		t.render()
		return true
	case fyne.KeyDown, fyne.KeyPageDown:
		if t.scrollOffset == 0 {
			return false
		}
		step := 1
		if name == fyne.KeyPageDown {
			step = t.rows
		}
		t.scrollOffset = clampInt(t.scrollOffset-step, 0, history)
		t.render()
		return true
	case fyne.KeyHome:
		if t.scrollOffset == 0 {
			return false
		}
		t.scrollOffset = history
		t.render()
		return true
	case fyne.KeyEnd:
		if t.scrollOffset == 0 {
			return false
		}
		t.scrollOffset = 0
		t.render()
		return true
	default:
		return false
	}
}

func (t *termWidget) sendBytes(data []byte) {
	if t.send == nil || len(data) == 0 {
		return
	}
	t.send(data)
}

var keySequences = map[fyne.KeyName][]byte{
	fyne.KeyEnter:     []byte("\r"),
	fyne.KeyReturn:    []byte("\r"),
	fyne.KeyBackspace: []byte{0x7f},
	fyne.KeyDelete:    []byte("\x1b[3~"),
	fyne.KeyTab:       []byte("\t"),
	fyne.KeyEscape:    []byte("\x1b"),
	fyne.KeyUp:        []byte("\x1b[A"),
	fyne.KeyDown:      []byte("\x1b[B"),
	fyne.KeyRight:     []byte("\x1b[C"),
	fyne.KeyLeft:      []byte("\x1b[D"),
	fyne.KeyHome:      []byte("\x1b[H"),
	fyne.KeyEnd:       []byte("\x1b[F"),
	fyne.KeyPageUp:    []byte("\x1b[5~"),
	fyne.KeyPageDown:  []byte("\x1b[6~"),
}

// write feeds remote output into the emulator and repaints. It must run on the
// UI thread.
func (t *termWidget) write(data []byte) {
	if _, err := t.emulator.Write(data); err != nil {
		return
	}
	t.render()
}

// selectedText returns the highlighted text.
func (t *termWidget) selectedText() string {
	start, end, ok := t.selectionRange()
	if !ok {
		return ""
	}
	buffer := t.emulator.Lines()
	lines := make([]string, 0, end.row-start.row+1)
	for row := start.row; row <= end.row && row < len(buffer); row++ {
		line := []rune(buffer[row])
		first, last := 0, len(line)-1
		if row == start.row {
			first = start.col
		}
		if row == end.row {
			last = end.col
		}
		if first > last {
			lines = append(lines, "")
			continue
		}
		if last >= len(line) {
			last = len(line) - 1
		}
		if first <= last {
			lines = append(lines, string(line[first:last+1]))
		}
	}
	return strings.Join(lines, "\n")
}

// screenText returns the selection, or the whole buffer when nothing is
// selected.
func (t *termWidget) screenText() string {
	if text := t.selectedText(); text != "" {
		return text
	}
	return t.emulator.BufferText()
}

func (t *termWidget) render() {
	if t.grid == nil {
		return
	}
	snapshot := t.emulator.Snapshot()
	rows := make([]widget.TextGridRow, snapshot.Rows)
	lines, start := t.viewStart()
	historyLen := t.emulator.HistoryLen()
	for y := 0; y < snapshot.Rows; y++ {
		bufferRow := start + y
		cells := make([]widget.TextGridCell, snapshot.Cols)
		for x := 0; x < snapshot.Cols; x++ {
			var char rune = ' '
			foreground, background := terminal.DefaultFG, terminal.DefaultBG
			if bufferRow < len(lines) {
				line := []rune(lines[bufferRow])
				if x < len(line) {
					char = line[x]
				}
			}
			// Keep the emulator colours for rows that are on screen.
			screenRow := bufferRow - historyLen
			if screenRow >= 0 && screenRow < snapshot.Rows {
				cell := snapshot.Cells[screenRow][x]
				if cell.Rune != 0 {
					char = cell.Rune
				}
				foreground, background = cell.FG, cell.BG
				if snapshot.CursorOn && t.scrollOffset == 0 && x == snapshot.CursorX && screenRow == snapshot.CursorY {
					foreground, background = background, foreground
				}
			}
			if t.selected(x, bufferRow) {
				foreground, background = background, foreground
			}
			cells[x] = widget.TextGridCell{Rune: char, Style: t.style(foreground, background)}
		}
		rows[y] = widget.TextGridRow{Cells: cells}
	}
	t.grid.Rows = rows
	t.grid.Refresh()
}

func (t *termWidget) style(fg, bg int32) widget.TextGridStyle {
	key := uint32(fg+2)<<16 | uint32(bg+2)
	if style, known := t.styles[key]; known {
		return style
	}
	style := &widget.CustomTextGridStyle{
		TextStyle: fyne.TextStyle{Monospace: true},
		FGColor:   paletteColor(fg),
		BGColor:   paletteColor(bg),
	}
	t.styles[key] = style
	return style
}

// paletteColor resolves xterm palette indices and the two defaults.
func paletteColor(id int32) color.Color {
	switch {
	case id == terminal.DefaultFG:
		return theme.ForegroundColor()
	case id == terminal.DefaultBG:
		return theme.BackgroundColor()
	case id >= 0 && id < 16:
		base := xtermBasePalette[id]
		return color.NRGBA{R: base.R, G: base.G, B: base.B, A: 0xff}
	case id >= 16 && id < 232:
		index := id - 16
		levels := [6]uint8{0, 95, 135, 175, 215, 255}
		return color.NRGBA{R: levels[index/36], G: levels[(index/6)%6], B: levels[index%6], A: 0xff}
	case id >= 232 && id < 256:
		gray := uint8(8 + (id-232)*10)
		return color.NRGBA{R: gray, G: gray, B: gray, A: 0xff}
	default:
		return theme.ForegroundColor()
	}
}

var xtermBasePalette = [16]color.NRGBA{
	{R: 0x00, G: 0x00, B: 0x00}, {R: 0xcd, G: 0x00, B: 0x00}, {R: 0x00, G: 0xcd, B: 0x00}, {R: 0xcd, G: 0xcd, B: 0x00},
	{R: 0x00, G: 0x00, B: 0xee}, {R: 0xcd, G: 0x00, B: 0xcd}, {R: 0x00, G: 0xcd, B: 0xcd}, {R: 0xe5, G: 0xe5, B: 0xe5},
	{R: 0x7f, G: 0x7f, B: 0x7f}, {R: 0xff, G: 0x00, B: 0x00}, {R: 0x00, G: 0xff, B: 0x00}, {R: 0xff, G: 0xff, B: 0x00},
	{R: 0x5c, G: 0x5c, B: 0xff}, {R: 0xff, G: 0x00, B: 0xff}, {R: 0x00, G: 0xff, B: 0xff}, {R: 0xff, G: 0xff, B: 0xff},
}
