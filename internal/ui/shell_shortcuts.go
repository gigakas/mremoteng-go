package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// This file holds the window-level keyboard shortcuts that act on the focused
// terminal.

// toggleLog shows or hides the activity console with its own status-bar icon.
// registerControlShortcuts registers Ctrl+<key> so Fyne delivers the combination
// to the focused terminal instead of dropping it: key events carry no modifier
// information, and only a few control combinations are built in.
func (s *screen) registerControlShortcuts() {
	canvas := s.win.Canvas()
	if canvas == nil {
		return
	}
	for key := range controlCodes {
		if handledByFyne[key] {
			continue
		}
		key := key
		canvas.AddShortcut(&desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierControl},
			func(fyne.Shortcut) { s.sendControlToFocused(key) })
	}
	// Shift-Tab is the one shifted key a terminal tells apart from its plain form, and it
	// cannot arrive as a typed key: key events carry no modifier state, so Tab alone is
	// completion. A registered chord is the only way to see the shift.
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierShift},
		func(fyne.Shortcut) { s.sendKeySequenceToFocused("\x1b[Z") })
	// Zoom is keys only: Fyne scroll events carry no modifiers, and reading the
	// Ctrl state from X misreported it on Wayland/XWayland, so a plain wheel
	// zoomed instead of scrolling and froze the console.
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyEqual, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { s.zoomFocusedTerminal(2) })
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyMinus, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { s.zoomFocusedTerminal(-2) })
	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key0, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { s.resetFocusedTerminalZoom() })
}

// sendKeySequenceToFocused writes raw bytes to the focused terminal. It is how keys the
// terminal has to synthesise (the shift-tab sequence) reach the shell.
func (s *screen) sendKeySequenceToFocused(sequence string) {
	term, ok := s.focusedTerminal()
	if !ok || sequence == "" {
		return
	}
	term.sendBytes([]byte(sequence))
}

// zoomFocusedTerminal changes the font size of the focused terminal.
func (s *screen) zoomFocusedTerminal(delta float32) {
	if term, ok := s.focusedTerminal(); ok {
		term.adjustZoom(delta)
	}
}

// resetFocusedTerminalZoom returns the focused terminal to the default size.
func (s *screen) resetFocusedTerminalZoom() {
	if term, ok := s.focusedTerminal(); ok {
		term.adjustZoom(theme.TextSize() - term.textSize())
	}
}

// focusedTerminal returns the terminal that owns the keyboard.
func (s *screen) focusedTerminal() (*termWidget, bool) {
	canvas := s.win.Canvas()
	if canvas == nil {
		return nil, false
	}
	term, ok := canvas.Focused().(*termWidget)
	return term, ok
}

// sendControlToFocused delivers a control byte to the focused terminal, if any.
func (s *screen) sendControlToFocused(key fyne.KeyName) {
	canvas := s.win.Canvas()
	if canvas == nil {
		return
	}
	term, ok := canvas.Focused().(*termWidget)
	if !ok {
		return
	}
	if code, known := controlCodes[key]; known {
		term.sendBytes([]byte{code})
	}
}
