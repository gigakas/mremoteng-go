// Package terminal wraps a VT100/xterm emulator so the SSH session can render
// a console inside the application instead of requiring an external terminal.
//
// The emulator interprets escape sequences; this package exposes a snapshot of
// the visible screen that the UI paints and a bounded scrollback history.
package terminal

import (
	"bytes"
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
	// historyCells and lastCells mirror history and last with colours, so
	// scrolled-back lines keep the styling they had on screen.
	historyCells [][]Cell
	lastCells    [][]Cell
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

// writeChunkBytes caps how much of a run without newlines is fed per step, so a
// single enormous line cannot make one step unbounded.
const writeChunkBytes = 8 * 1024

// Write feeds terminal output into the emulator and records scrolled lines.
//
// The scrollback is rebuilt by comparing consecutive screens, and that
// comparison can only recognise a shift smaller than the screen is tall. A block
// larger than the screen - a build log, a cat of a big file - would therefore
// scroll past unnoticed, so the input is fed in steps that end at a newline and
// carry at most rows-1 of them. Splitting there never cuts an escape sequence: a
// newline is not part of one, and a CRLF stays together.
func (e *Emulator) Write(p []byte) (int, error) {
	dropsHistory := bytes.Contains(p, eraseScrollback)
	written := 0
	for len(p) > 0 {
		chunk := e.nextChunk(p)
		n, err := e.term.Write(chunk)
		written += n
		if n > 0 {
			p = p[n:]
		}
		e.trackScroll()
		if err != nil {
			return written, err
		}
		if n == 0 {
			// No progress: stop instead of spinning on the same bytes.
			return written, nil
		}
	}
	if dropsHistory {
		e.history, e.historyCells = nil, nil
	}
	return written, nil
}

// nextChunk returns the next slice to feed: it ends at a newline and carries at
// most rows-1 newlines, which keeps every step alignable by trackScroll.
func (e *Emulator) nextChunk(p []byte) []byte {
	_, rows := e.term.Size()
	// Half a screen per step: the alignment needs real rows on both sides, and a
	// step of rows-1 leaves only one row to compare, which is usually blank right
	// after a scroll.
	perStep := rows / 2
	if perStep < 1 {
		perStep = 1
	}
	limit := len(p)
	if limit > writeChunkBytes {
		limit = writeChunkBytes
	}
	newlines := 0
	for i := 0; i < limit; i++ {
		if p[i] != '\n' {
			continue
		}
		newlines++
		if newlines >= perStep {
			return p[:i+1]
		}
	}
	return p[:limit]
}

// eraseScrollback is ESC[3J, which `clear` sends to drop the scrollback too.
var eraseScrollback = []byte("\x1b[3J")

// trackScroll appends lines that left the screen. Shell output scrolls one or a
// few rows at a time; the detection compares the previous screen with the new
// one and is deliberately conservative.
func (e *Emulator) trackScroll() {
	snapshot := e.Snapshot()
	current := linesOf(snapshot)
	if e.last == nil {
		e.last, e.lastCells = current, snapshot.Cells
		return
	}
	rows := len(current)
	if len(e.last) != rows {
		e.last, e.lastCells = current, snapshot.Cells
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
		e.historyCells = append(e.historyCells, e.lastCells[:shift]...)
		if len(e.history) > historyLimit {
			e.history = append([]string(nil), e.history[len(e.history)-historyLimit:]...)
			e.historyCells = append([][]Cell(nil), e.historyCells[len(e.historyCells)-historyLimit:]...)
		}
		break
	}
	e.last, e.lastCells = current, snapshot.Cells
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
	return linesOf(e.Snapshot())
}

// linesOf converts a snapshot into plain text rows with trailing blanks trimmed.
func linesOf(snapshot Snapshot) []string {
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

// History returns the scrollback lines, without the visible screen. Unlike
// Lines, this needs no Snapshot: the UI calls it on every scroll and render,
// and a second full screen copy there was expensive enough to visibly stall
// the terminal during a fast scroll burst. Callers must not mutate the
// result.
func (e *Emulator) History() []string {
	return e.history
}

// HistoryCells returns the scrollback with colours, parallel to History.
// Callers must not mutate the result.
func (e *Emulator) HistoryCells() [][]Cell {
	return e.historyCells
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
