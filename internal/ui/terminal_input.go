package ui

import (
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// This file holds the terminal widget's input side: mouse, selection,
// scrolling and keyboard handling.

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
	cell := t.cellSize()
	if cell.Width <= 0 || cell.Height <= 0 {
		return cellPos{}
	}
	col := int(position.X / cell.Width)
	row := int(position.Y / cell.Height)
	start := t.viewStart()
	return cellPos{col: clampInt(col, 0, t.cols-1), row: start + clampInt(row, 0, t.rows-1)}
}

// viewStart returns the index of the first visible buffer line. It is
// history-length minus the scroll offset: the history plus the current
// screen always totals historyLen+rows lines, so this needs no full Lines()/
// Snapshot() call just to find where the view starts.
func (t *termWidget) viewStart() int {
	start := t.emulator.HistoryLen() - t.scrollOffset
	if start < 0 {
		start = 0
	}
	return start
}

// Wheel codes and the conversion from Fyne scroll units to wheel notches. SGR
// reports the wheel as buttons 64 and 65; the legacy X10 encoding has no room
// for those and uses buttons 4 and 5 instead.
const (
	wheelUpButton   = 64
	wheelDownButton = 65
	legacyWheelBase = 60
	// scrollUnitsPerNotch is what the desktop driver sends for one wheel click
	// on Linux (a touchpad sends less, including fractions), so a program must
	// receive one event per notch rather than one per unit.
	scrollUnitsPerNotch = 25
	maxWheelNotches     = 10
)

// Scrolled hands the wheel to the program when it enabled mouse reporting,
// turns it into arrow keys while a full-screen program owns the alternate
// screen, and moves through the local scrollback otherwise.
//
// Without the first two cases a TUI client (a chat client, an editor, a
// multiplexer) never receives the wheel: the emulator consumed it for a
// scrollback that its alternate screen does not fill.
func (t *termWidget) Scrolled(event *fyne.ScrollEvent) {
	if event.Scrolled.DY == 0 {
		return
	}
	modes := t.emulator.Modes()
	switch {
	case modes.MouseReporting:
		t.sendWheel(t.screenCellAt(event.Position), wheelNotches(event.Scrolled.DY), modes.MouseSGR)
	case modes.AltScreen:
		t.sendArrowScroll(wheelNotches(event.Scrolled.DY), modes.AppCursor)
	default:
		// The local scrollback keeps using Fyne's units: a notch moves a
		// visible block, which is the behaviour its users already know.
		t.scrollBack(int(event.Scrolled.DY))
	}
}

// wheelNotches converts a Fyne scroll delta into whole wheel notches. A
// fractional delta (a touchpad) still counts as one notch, because otherwise a
// slow gesture would never reach the program.
func wheelNotches(delta float32) int {
	notches := int(delta / scrollUnitsPerNotch)
	if notches == 0 {
		if delta > 0 {
			notches = 1
		} else {
			notches = -1
		}
	}
	return clampInt(notches, -maxWheelNotches, maxWheelNotches)
}

// scrollBack moves the view through the scrollback and keeps it inside it.
func (t *termWidget) scrollBack(notches int) {
	t.scrollOffset = clampInt(t.scrollOffset+notches, 0, t.emulator.HistoryLen())
	t.render()
}

// screenCellAt maps a widget position to a cell of the visible screen. Unlike
// cellAt it ignores the scroll offset: a program that receives mouse or arrow
// events works with what is on screen, not with the buffer above it.
func (t *termWidget) screenCellAt(position fyne.Position) cellPos {
	cell := t.cellSize()
	if cell.Width <= 0 || cell.Height <= 0 {
		return cellPos{}
	}
	col := int(position.X / cell.Width)
	row := int(position.Y / cell.Height)
	return cellPos{col: clampInt(col, 0, t.cols-1), row: clampInt(row, 0, t.rows-1)}
}

// sendWheel reports one wheel notch per line at the cell under the pointer.
func (t *termWidget) sendWheel(at cellPos, notches int, sgr bool) {
	if t.send == nil {
		return
	}
	code := wheelUpButton
	if notches < 0 {
		code = wheelDownButton
	}
	for i := 0; i < wheelCount(notches); i++ {
		if sgr {
			t.send([]byte("\x1b[<" + strconv.Itoa(code) + ";" +
				strconv.Itoa(at.col+1) + ";" + strconv.Itoa(at.row+1) + "M"))
			continue
		}
		t.send([]byte{0x1b, '[', 'M',
			byte(32 + code - legacyWheelBase),
			byte(32 + at.col + 1),
			byte(32 + at.row + 1)})
	}
}

// sendArrowScroll turns a wheel notch into an arrow key, the alternate-scroll
// behaviour terminals have: a full-screen program without mouse reporting still
// receives the intent to move.
func (t *termWidget) sendArrowScroll(notches int, appCursor bool) {
	if t.send == nil {
		return
	}
	up, down := "\x1b[A", "\x1b[B"
	if appCursor {
		up, down = "\x1bOA", "\x1bOB"
	}
	key := up
	if notches < 0 {
		key = down
	}
	for i := 0; i < wheelCount(notches); i++ {
		t.send([]byte(key))
	}
}

// wheelCount is the number of notches, without its direction.
func wheelCount(notches int) int {
	if notches < 0 {
		return -notches
	}
	return notches
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

// MouseUp ends the press. Its real job is to make termWidget a desktop.Mouseable:
// Fyne only calls MouseDown on widgets that implement both halves, so without it
// the terminal took the keyboard solely from Tapped, which never fires when the
// pointer moves a pixel during the click (it becomes a drag) and which Fyne
// delays for a double-tap. A click like that left the terminal unfocused.
func (t *termWidget) MouseUp(*desktop.MouseEvent) {
	t.selecting = false
	t.dragX, t.dragY = 0, 0
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
		// The driver hands a focused Shortcutable widget every matching
		// CustomShortcut directly, bypassing the canvas-level registration in
		// shell.go, so the zoom keys must be handled here or they are silently
		// dropped while the terminal holds the keyboard.
		switch typed.KeyName {
		case fyne.KeyEqual:
			t.adjustZoom(2)
			return
		case fyne.KeyMinus:
			t.adjustZoom(-2)
			return
		case fyne.Key0:
			t.adjustZoom(theme.TextSize() - t.textSize())
			return
		}
		if code, ok := controlCodes[typed.KeyName]; ok {
			t.sendBytes([]byte{code})
		}
	}
}

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
	// Repainting the whole grid on every pointer move is what made a long
	// selection freeze; the selection itself is still updated per event.
	if now := time.Now(); now.Sub(t.lastDragPaint) >= dragPaintInterval {
		t.lastDragPaint = now
		t.render()
	}
}

// extendSelection updates the drag end, scrolling the view when the pointer goes
// past the top or bottom edge so long outputs can be selected in one drag.
func (t *termWidget) extendSelection(position fyne.Position) {
	cell := t.cellSize()
	if cell.Width <= 0 || cell.Height <= 0 {
		return
	}
	col := clampInt(int(position.X/cell.Width), 0, t.cols-1)
	history := t.emulator.HistoryLen()
	switch {
	case position.Y < 0:
		steps := 1 + int(-position.Y/cell.Height)
		t.scrollOffset = clampInt(t.scrollOffset+steps, 0, history)
		start := t.viewStart()
		t.selEnd = cellPos{col: col, row: start}
	case position.Y > t.Size().Height:
		steps := 1 + int((position.Y-t.Size().Height)/cell.Height)
		t.scrollOffset = clampInt(t.scrollOffset-steps, 0, history)
		start := t.viewStart()
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

func (t *termWidget) FocusLost() {}

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
	// Typing, Enter or a paste returns the view to the live line, like a
	// regular terminal.
	if t.scrollOffset != 0 {
		t.scrollOffset = 0
		t.render()
	}
	t.send(data)
}
