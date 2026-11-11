// Package terminal wraps a VT100/xterm emulator so the SSH session can render
// a console inside the application instead of requiring an external terminal.
//
// The emulator interprets escape sequences; this package exposes a snapshot of
// the visible screen that the UI paints and a bounded scrollback history.
package terminal

import (
	"strings"

	"github.com/hinshun/vt10x"
)

// Color identifiers used by Snapshot cells. Palette entries 0-255 follow the
// xterm palette; the two defaults resolve to the UI's foreground/background.
const (
	DefaultFG int32 = -1
	DefaultBG int32 = -2
)

// Cell is one screen position.
type Cell struct {
	Rune rune
	FG   int32
	BG   int32
}

// Snapshot is the visible screen plus cursor state.
type Snapshot struct {
	Cells    [][]Cell
	Cols     int
	Rows     int
	CursorX  int
	CursorY  int
	CursorOn bool
	Title    string
}

// historyLimit bounds the scrollback so a long session cannot grow without
// limit.
const historyLimit = 2000

// Emulator interprets terminal output and exposes the resulting screen plus a
// scrollback of the lines that scrolled off the top.
type Emulator struct {
	term    vt10x.Terminal
	history []string
	last    []string
}

// New creates an emulator sized in character cells.
func New(cols, rows int) *Emulator {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return &Emulator{term: vt10x.New(vt10x.WithSize(cols, rows))}
}

// Write feeds terminal output into the emulator and records scrolled lines.
func (e *Emulator) Write(p []byte) (int, error) {
	n, err := e.term.Write(p)
	e.trackScroll()
	return n, err
}

// trackScroll appends lines that left the screen. Shell output scrolls one or a
// few rows at a time; the detection compares the previous screen with the new
// one and is deliberately conservative.
func (e *Emulator) trackScroll() {
	current := e.screenLines()
	if e.last == nil {
		e.last = current
		return
	}
	rows := len(current)
	if len(e.last) != rows {
		e.last = current
		return
	}
	// An unchanged screen is a redraw (prompt, cursor move), not a scroll:
	// recording it would duplicate history.
	if equalLines(current, e.last) {
		return
	}
	for shift := 1; shift < rows; shift++ {
		matches := true
		compared := 0
		for row := 0; row < rows-shift; row++ {
			previous := e.last[row+shift]
			// A blank row at the bottom of the previous screen is the cursor
			// line before the newline: treat it as a wildcard.
			if previous == "" {
				continue
			}
			if previous != current[row] {
				matches = false
				break
			}
			compared++
		}
		if !matches || compared == 0 {
			continue
		}
		e.history = append(e.history, e.last[:shift]...)
		if len(e.history) > historyLimit {
			e.history = append([]string(nil), e.history[len(e.history)-historyLimit:]...)
		}
		break
	}
	e.last = current
}

// equalLines compares two screens row by row.
func equalLines(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// screenLines returns the visible screen as plain text rows.
func (e *Emulator) screenLines() []string {
	snapshot := e.Snapshot()
	lines := make([]string, 0, snapshot.Rows)
	for _, row := range snapshot.Cells {
		var builder []rune
		for _, cell := range row {
			if cell.Rune == 0 {
				builder = append(builder, ' ')
				continue
			}
			builder = append(builder, cell.Rune)
		}
		lines = append(lines, strings.TrimRight(string(builder), " "))
	}
	return lines
}

// Lines returns the scrollback followed by the visible screen.
func (e *Emulator) Lines() []string {
	lines := append([]string(nil), e.history...)
	return append(lines, e.screenLines()...)
}

// HistoryLen reports how many scrollback lines are stored.
func (e *Emulator) HistoryLen() int {
	if e == nil {
		return 0
	}
	return len(e.history)
}

// BufferText is the whole buffer: scrollback plus screen, trailing blanks
// trimmed.
func (e *Emulator) BufferText() string {
	lines := e.Lines()
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// Resize changes the screen size in character cells.
func (e *Emulator) Resize(cols, rows int) {
	if cols < 1 || rows < 1 {
		return
	}
	e.term.Resize(cols, rows)
}

// Size reports the current size in character cells.
func (e *Emulator) Size() (cols, rows int) {
	return e.term.Size()
}

// Snapshot copies the visible screen. The emulator lock is held while copying
// so a concurrent Write cannot produce a torn frame.
func (e *Emulator) Snapshot() Snapshot {
	e.term.Lock()
	defer e.term.Unlock()

	cols, rows := e.term.Size()
	cells := make([][]Cell, rows)
	for y := 0; y < rows; y++ {
		row := make([]Cell, cols)
		for x := 0; x < cols; x++ {
			glyph := e.term.Cell(x, y)
			row[x] = Cell{Rune: glyph.Char, FG: colorID(glyph.FG, DefaultFG), BG: colorID(glyph.BG, DefaultBG)}
		}
		cells[y] = row
	}
	cursor := e.term.Cursor()
	return Snapshot{
		Cells:    cells,
		Cols:     cols,
		Rows:     rows,
		CursorX:  cursor.X,
		CursorY:  cursor.Y,
		CursorOn: e.term.CursorVisible(),
		Title:    e.term.Title(),
	}
}

func colorID(color vt10x.Color, fallback int32) int32 {
	switch {
	case color == vt10x.DefaultFG:
		return DefaultFG
	case color == vt10x.DefaultBG:
		return DefaultBG
	default:
		return int32(color)
	}
}
