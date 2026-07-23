package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// defaultTermCols/Rows match the SSH backend's own PTY defaults
// (internal/protocol/ssh's defaultCols/defaultRows), so a freshly opened
// terminal doesn't immediately mismatch what the remote end was told.
const (
	defaultTermCols = 80
	defaultTermRows = 24
)

// Terminal is a minimal ANSI/VT100 terminal view: a widget.TextGrid driven
// by an ansiState byte-stream interpreter, with keyboard input relayed to
// whatever consumes it (a protocol.TerminalProtocol's Write, in normal
// use). See ansiState's doc comment for exactly what escape-sequence
// subset is supported — this is not a full xterm implementation.
type Terminal struct {
	widget.BaseWidget

	grid  *widget.TextGrid
	state *ansiState

	// OnInput, if set, receives raw bytes for every keystroke — printable
	// runes as UTF-8, special keys as their control byte or CSI escape
	// sequence. Left nil by default; the session-tab assembly point wires
	// it to the active protocol.TerminalProtocol's Write.
	OnInput func([]byte)
}

// NewTerminal creates a Terminal at the default size. Call Write to feed
// it protocol output.
func NewTerminal() *Terminal {
	t := &Terminal{grid: widget.NewTextGrid()}
	t.state = newANSIState(t.grid, defaultTermCols, defaultTermRows)
	t.ExtendBaseWidget(t)
	return t
}

// CharSize reports the terminal's character-cell dimensions, for a
// protocol.TerminalProtocol backend's initial Resize call (width, height
// — character cells, per protocol.Protocol.Resize's documented,
// backend-defined unit for terminal protocols). Named CharSize, not
// Size, because widget.BaseWidget already defines Size() fyne.Size (the
// widget's pixel size on screen) — a different, unrelated concept.
func (t *Terminal) CharSize() (cols, rows int) { return t.state.cols, t.state.rows }

// CreateRenderer implements fyne.Widget.
func (t *Terminal) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.grid)
}

// Write implements io.Writer: feeds raw protocol output (e.g. an SSH
// session's stdout) into the ANSI interpreter. Safe to call from any
// goroutine only in the sense that ansiState itself has no internal
// locking — callers (the session-tab assembly point) are responsible for
// not calling Write concurrently with itself, and for hopping onto
// Fyne's main goroutine (fyne.Do) before calling it, since it mutates
// widget state.
func (t *Terminal) Write(p []byte) (int, error) {
	for _, b := range p {
		t.state.Feed(b)
	}
	t.grid.Refresh()
	return len(p), nil
}

// Tapped implements fyne.Tappable: clicking the terminal requests
// keyboard focus for it, the same as clicking a text field.
func (t *Terminal) Tapped(*fyne.PointEvent) {
	if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
		c.Focus(t)
	}
}

// FocusGained/FocusLost implement fyne.Focusable. The terminal has no
// visual focus indicator yet (a v1 gap — a cursor-cell highlight or
// border would be the natural addition, not attempted here).
func (t *Terminal) FocusGained() {}
func (t *Terminal) FocusLost()   {}

// TypedRune implements fyne.Focusable: printable character input.
func (t *Terminal) TypedRune(r rune) {
	if t.OnInput != nil {
		t.OnInput([]byte(string(r)))
	}
}

// TypedKey implements fyne.Focusable: non-printable key input, mapped to
// the control byte or CSI escape sequence a real terminal would send.
// Keys with no defined mapping here are silently ignored, not an error —
// this is a v1 subset (see the package's terminal-widget scope note),
// not a full keyboard-to-terminal mapping.
func (t *Terminal) TypedKey(e *fyne.KeyEvent) {
	if t.OnInput == nil {
		return
	}
	if seq, ok := terminalKeySequences[e.Name]; ok {
		t.OnInput([]byte(seq))
	}
}

var terminalKeySequences = map[fyne.KeyName]string{
	fyne.KeyReturn:    "\r",
	fyne.KeyEnter:     "\r",
	fyne.KeyBackspace: "\x7f",
	fyne.KeyTab:       "\t",
	fyne.KeyEscape:    "\x1b",
	fyne.KeyUp:        "\x1b[A",
	fyne.KeyDown:      "\x1b[B",
	fyne.KeyRight:     "\x1b[C",
	fyne.KeyLeft:      "\x1b[D",
	fyne.KeyHome:      "\x1b[H",
	fyne.KeyEnd:       "\x1b[F",
	fyne.KeyDelete:    "\x1b[3~",
	fyne.KeyPageUp:    "\x1b[5~",
	fyne.KeyPageDown:  "\x1b[6~",
}

var _ fyne.Widget = (*Terminal)(nil)
var _ fyne.Focusable = (*Terminal)(nil)
var _ fyne.Tappable = (*Terminal)(nil)
