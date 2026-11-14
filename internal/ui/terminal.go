package ui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	// cellWidth, cellHeight and cellFont cache one character cell: measuring the
	// font on every pointer move is what made a long selection feel frozen.
	cellWidth  float32
	cellHeight float32
	cellFont   float32
	// lastDragPaint throttles the drag repaint for the same reason.
	lastDragPaint time.Time

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
	// renderer is the termRenderer built in CreateRenderer, kept so a zoom step
	// can update its font bookkeeping.
	renderer *termRenderer
}

// dragPaintInterval is how often a drag repaints the grid.
const dragPaintInterval = 40 * time.Millisecond

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

// cellSize returns the size of one character cell, measured once per text size.
func (t *termWidget) cellSize() fyne.Size {
	if t.cellFont == t.textSize() && t.cellWidth > 0 && t.cellHeight > 0 {
		return fyne.NewSize(t.cellWidth, t.cellHeight)
	}
	measured := fyne.MeasureText("M", t.textSize(), fyne.TextStyle{Monospace: true})
	t.cellFont, t.cellWidth, t.cellHeight = t.textSize(), measured.Width, measured.Height
	return measured
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
	if t.renderer != nil {
		// Keep the renderer's own bookkeeping in step, or the next unrelated
		// Layout pass (e.g. a window resize) sees a stale font and redoes this
		// work again.
		t.renderer.font = size
	}
	// rebuildGrid already renders once against the pre-zoom cols/rows, and fit
	// renders again if the new font size changed them (almost always, since
	// cell measurements changed); a third unconditional render here only
	// repeated that same work and made every zoom step noticeably sluggish.
	t.rebuildGrid()
	t.fit(t.Size())
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

// rebuildGrid applies the current zoom size to the text grid. Fyne's
// TextGrid already re-measures its cell size from the theme on every
// Refresh (textGridContentRenderer.Refresh, fyne.io/fyne/v2@v2.8.2), so a
// zoom step only needs to refresh the existing grid in place. The earlier
// version tore down and recreated the whole widget on every step believing
// that was the only way to make Fyne pick up the new size; that tear-down
// (and the texture/geometry work it forced on the GPU painter) is what made
// each zoom step visibly stall. installGrid still runs once, from
// CreateRenderer, to build the grid the first time.
func (t *termWidget) rebuildGrid() {
	if t.grid != nil {
		t.grid.Refresh()
	} else if t.renderer != nil {
		t.renderer.objects[0] = t.installGrid(t.Size())
	}
	t.render()
	// A Refresh further up the widget (not just the inner grid) is still
	// needed: without it, a zoom step from the keyboard shortcut sat
	// invisible until some unrelated event (losing then regaining focus)
	// forced a full repaint.
	t.Refresh()
}

type termRenderer struct {
	parent  *termWidget
	objects []fyne.CanvasObject
	// font is the text size the grid was last refreshed at, so Layout can tell
	// when the size changed and the grid needs refreshing and refitting.
	font float32
}

func (r *termRenderer) Layout(size fyne.Size) {
	if r.font != r.parent.textSize() || r.parent.grid == nil {
		r.font = r.parent.textSize()
		r.parent.rebuildGrid()
		r.parent.fit(size)
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
	cell := t.cellSize()
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
	if history := t.emulator.HistoryLen(); t.scrollOffset > history {
		t.scrollOffset = history
		t.clearSelection()
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
	start := t.viewStart()
	historyLen := t.emulator.HistoryLen()
	// History only, never the current screen: on-screen rows get their rune
	// from snapshot.Cells below, and fetching that text again here would mean
	// a second full Snapshot() per render (Lines() takes one), which is
	// expensive enough on a fast scroll burst to visibly stall the terminal.
	historyCells := t.emulator.HistoryCells()
	for y := 0; y < snapshot.Rows; y++ {
		bufferRow := start + y
		cells := make([]widget.TextGridCell, snapshot.Cols)
		for x := 0; x < snapshot.Cols; x++ {
			var char rune = ' '
			foreground, background := terminal.DefaultFG, terminal.DefaultBG
			if bufferRow < historyLen && x < len(historyCells[bufferRow]) {
				cell := historyCells[bufferRow][x]
				if cell.Rune != 0 {
					char = cell.Rune
				}
				foreground, background = cell.FG, cell.BG
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
	}
	// A cell on the default background gets no background at all: the window
	// already paints it, and Fyne draws one GL rectangle per styled background,
	// so a full 80x24 screen cost 1920 rectangle draws per repaint (measured at
	// about 8 ms of CPU for each full-screen update).
	if bg != terminal.DefaultBG {
		style.BGColor = paletteColor(bg)
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
