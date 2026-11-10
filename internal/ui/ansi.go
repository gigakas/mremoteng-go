package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ansiState is a minimal ANSI/VT100 byte-stream interpreter driving a
// widget.TextGrid. It is deliberately a small subset, not an xterm
// reimplementation — the blueprint flags a full terminal emulator as
// "the real cost driver" of stage 2.2/3.3 and asks for its cost to be
// estimated separately before attempting one. What's covered: cursor
// movement (CUU/CUD/CUF/CUB/CUP), erase in display/line (ED/EL), basic
// SGR (reset, bold, the 8 standard + 8 bright ANSI colors as foreground
// or background), CR/LF/backspace/tab. What's explicitly NOT covered:
// 256-color/truecolor SGR, the alternate screen buffer, mouse reporting,
// scroll regions, save/restore cursor, and any OSC payload (window title
// etc. — recognized and safely discarded rather than leaking into the
// display, but not acted on). Good enough for interactive shell use
// (prompts, ls --color, vim in its basic mode); not a full terminal.
type ansiState struct {
	grid       *widget.TextGrid
	cols, rows int
	row, col   int
	style      termStyle

	// Parser state machine.
	mode       byte // 0 = normal, 'esc' = just saw ESC, 'csi' = accumulating a CSI sequence, 'osc' = discarding an OSC sequence
	params     []int
	paramAccum int
	haveParam  bool
}

const (
	modeNormal byte = 0
	modeEsc    byte = 1
	modeCSI    byte = 2
	modeOSC    byte = 3
)

func newANSIState(grid *widget.TextGrid, cols, rows int) *ansiState {
	s := &ansiState{grid: grid, cols: cols, rows: rows, style: defaultTermStyle}
	s.resetGrid()
	return s
}

func (s *ansiState) resetGrid() {
	rows := make([]widget.TextGridRow, s.rows)
	for i := range rows {
		rows[i].Cells = make([]widget.TextGridCell, s.cols)
		for j := range rows[i].Cells {
			rows[i].Cells[j] = widget.TextGridCell{Rune: ' ', Style: defaultTermStyle}
		}
	}
	s.grid.Rows = rows
	s.row, s.col = 0, 0
}

// Feed processes one byte of protocol output.
func (s *ansiState) Feed(b byte) {
	switch s.mode {
	case modeEsc:
		switch b {
		case '[':
			s.mode = modeCSI
			s.params = s.params[:0]
			s.paramAccum, s.haveParam = 0, false
		case ']':
			s.mode = modeOSC
		default:
			// Unhandled single-byte escape (e.g. charset selection) —
			// drop it and resume, rather than printing it literally.
			s.mode = modeNormal
		}
		return
	case modeOSC:
		// OSC payloads end at BEL (0x07) or ESC \ (ST); either way,
		// discard the content. Since ESC \ needs a second byte, handle
		// only the common BEL terminator for v1 — a payload terminated
		// with ST instead will have its trailing ESC re-enter modeEsc
		// and then be dropped by the default case above, which is a
		// harmless, self-correcting fallback.
		if b == 0x07 {
			s.mode = modeNormal
		}
		return
	case modeCSI:
		s.feedCSI(b)
		return
	}

	switch b {
	case 0x1B: // ESC
		s.mode = modeEsc
	case '\r':
		s.col = 0
	case '\n':
		s.newline()
	case '\b':
		if s.col > 0 {
			s.col--
		}
	case '\t':
		s.col = (s.col/8 + 1) * 8
		if s.col >= s.cols {
			s.col = s.cols - 1
		}
	default:
		s.put(rune(b))
	}
}

// FeedString feeds a decoded string, for runes that arrive pre-decoded
// (e.g. typed input echo) rather than as a raw byte stream. Multi-byte
// UTF-8 sequences arriving via Feed are not decoded — each byte is
// treated as one cell, a known limitation of this minimal implementation
// (see the ansiState doc comment).
func (s *ansiState) FeedString(str string) {
	for _, r := range str {
		if r == '\n' {
			s.newline()
			continue
		}
		s.put(r)
	}
}

func (s *ansiState) put(r rune) {
	if s.col >= s.cols {
		s.col = 0
		s.newline()
	}
	s.grid.Rows[s.row].Cells[s.col] = widget.TextGridCell{Rune: r, Style: s.style}
	s.col++
}

func (s *ansiState) newline() {
	if s.row == s.rows-1 {
		copy(s.grid.Rows, s.grid.Rows[1:])
		last := len(s.grid.Rows) - 1
		s.grid.Rows[last].Cells = make([]widget.TextGridCell, s.cols)
		for j := range s.grid.Rows[last].Cells {
			s.grid.Rows[last].Cells[j] = widget.TextGridCell{Rune: ' ', Style: defaultTermStyle}
		}
		return
	}
	s.row++
}

func (s *ansiState) feedCSI(b byte) {
	switch {
	case b >= '0' && b <= '9':
		s.paramAccum = s.paramAccum*10 + int(b-'0')
		s.haveParam = true
		return
	case b == ';':
		s.params = append(s.params, s.paramAccum)
		s.paramAccum, s.haveParam = 0, false
		return
	}

	if s.haveParam || len(s.params) == 0 {
		s.params = append(s.params, s.paramAccum)
	}
	s.mode = modeNormal

	p := func(i, def int) int {
		if i >= len(s.params) || s.params[i] == 0 {
			return def
		}
		return s.params[i]
	}

	switch b {
	case 'A':
		s.row = max0(s.row - p(0, 1))
	case 'B':
		s.row = min(s.rows-1, s.row+p(0, 1))
	case 'C':
		s.col = min(s.cols-1, s.col+p(0, 1))
	case 'D':
		s.col = max0(s.col - p(0, 1))
	case 'H', 'f':
		s.row = clamp(p(0, 1)-1, 0, s.rows-1)
		s.col = clamp(p(1, 1)-1, 0, s.cols-1)
	case 'J':
		s.eraseDisplay(firstParam(s.params))
	case 'K':
		s.eraseLine(firstParam(s.params))
	case 'm':
		s.applySGR(s.params)
	default:
		// Unhandled CSI final byte (mode set/reset, save/restore cursor,
		// scroll region, ...): parameters already consumed above, so
		// just resume — this is what keeps unsupported sequences from
		// corrupting the display instead of merely not doing anything.
	}
}

func (s *ansiState) eraseDisplay(mode int) {
	switch mode {
	case 2, 3:
		s.resetGrid()
	case 0:
		s.eraseLine(0)
		for r := s.row + 1; r < s.rows; r++ {
			s.clearRow(r)
		}
	case 1:
		s.eraseLine(1)
		for r := 0; r < s.row; r++ {
			s.clearRow(r)
		}
	}
}

func (s *ansiState) eraseLine(mode int) {
	switch mode {
	case 0:
		for c := s.col; c < s.cols; c++ {
			s.grid.Rows[s.row].Cells[c] = widget.TextGridCell{Rune: ' ', Style: defaultTermStyle}
		}
	case 1:
		for c := 0; c <= s.col && c < s.cols; c++ {
			s.grid.Rows[s.row].Cells[c] = widget.TextGridCell{Rune: ' ', Style: defaultTermStyle}
		}
	case 2:
		s.clearRow(s.row)
	}
}

func (s *ansiState) clearRow(row int) {
	for c := 0; c < s.cols; c++ {
		s.grid.Rows[row].Cells[c] = widget.TextGridCell{Rune: ' ', Style: defaultTermStyle}
	}
}

func (s *ansiState) applySGR(params []int) {
	if len(params) == 0 {
		s.style = defaultTermStyle
		return
	}
	for _, p := range params {
		switch {
		case p == 0:
			s.style = defaultTermStyle
		case p == 1:
			s.style.bold = true
		case p == 22:
			s.style.bold = false
		case p >= 30 && p <= 37:
			s.style.fg = ansiPalette[p-30]
		case p == 39:
			s.style.fg = nil
		case p >= 40 && p <= 47:
			s.style.bg = ansiPalette[p-40]
		case p == 49:
			s.style.bg = nil
		case p >= 90 && p <= 97:
			s.style.fg = ansiPalette[p-90+8]
		case p >= 100 && p <= 107:
			s.style.bg = ansiPalette[p-100+8]
		}
	}
}

func firstParam(params []int) int {
	if len(params) == 0 {
		return 0
	}
	return params[0]
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// termStyle implements widget.TextGridStyle. fg/bg nil means "use the
// theme default", matching how the standard ANSI reset codes (39/49)
// behave.
type termStyle struct {
	bold   bool
	fg, bg color.Color
}

var defaultTermStyle = termStyle{}

func (s termStyle) Style() fyne.TextStyle {
	return fyne.TextStyle{Bold: s.bold, Monospace: true}
}

func (s termStyle) TextColor() color.Color {
	if s.fg == nil {
		return theme.ForegroundColor()
	}
	return s.fg
}

func (s termStyle) BackgroundColor() color.Color {
	return s.bg // nil is valid: widget.TextGrid treats a nil background as "no override"
}

// ansiPalette is the standard 16-color ANSI/xterm palette: indices 0-7 are
// the normal colors, 8-15 the bright variants (SGR 90-97/100-107).
var ansiPalette = [16]color.Color{
	color.NRGBA{R: 0, G: 0, B: 0, A: 255},
	color.NRGBA{R: 205, G: 0, B: 0, A: 255},
	color.NRGBA{R: 0, G: 205, B: 0, A: 255},
	color.NRGBA{R: 205, G: 205, B: 0, A: 255},
	color.NRGBA{R: 0, G: 0, B: 238, A: 255},
	color.NRGBA{R: 205, G: 0, B: 205, A: 255},
	color.NRGBA{R: 0, G: 205, B: 205, A: 255},
	color.NRGBA{R: 229, G: 229, B: 229, A: 255},
	color.NRGBA{R: 127, G: 127, B: 127, A: 255},
	color.NRGBA{R: 255, G: 0, B: 0, A: 255},
	color.NRGBA{R: 0, G: 255, B: 0, A: 255},
	color.NRGBA{R: 255, G: 255, B: 0, A: 255},
	color.NRGBA{R: 92, G: 92, B: 255, A: 255},
	color.NRGBA{R: 255, G: 0, B: 255, A: 255},
	color.NRGBA{R: 0, G: 255, B: 255, A: 255},
	color.NRGBA{R: 255, G: 255, B: 255, A: 255},
}
