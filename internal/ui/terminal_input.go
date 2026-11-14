package ui

import (
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
