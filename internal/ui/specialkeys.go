package ui

import (
	"errors"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Key symbols from the X11 keysym table. Only the ones a chord needs, so the file
// documents exactly what it can send.
const (
	keysymShiftL   uint32 = 0xffe1
	keysymControlL uint32 = 0xffe3
	keysymAltL     uint32 = 0xffe9
	keysymDelete   uint32 = 0xffff
	keysymEnd      uint32 = 0xff57
	keysymInsert   uint32 = 0xff63
	keysymTab      uint32 = 0xff09
	keysymEscape   uint32 = 0xff1b
)

// specialChord is a key combination the local desktop or the X server intercepts
// before the remote session can see it. These are the chords mRemoteNG offers, which
// is why they need a button at all: pressing them can never reach the session.
type specialChord struct {
	Label string
	// Keysyms are the modifiers first and the key last, which is also the order they
	// are pressed in.
	Keysyms []uint32
}

// specialChords is the menu the session bar and the View menu offer.
var specialChords = []specialChord{
	{"Ctrl+Alt+Del", []uint32{keysymControlL, keysymAltL, keysymDelete}},
	{"Ctrl+Alt+End", []uint32{keysymControlL, keysymAltL, keysymEnd}},
	{"Ctrl+Alt+Insert", []uint32{keysymControlL, keysymAltL, keysymInsert}},
	{"Alt+Tab", []uint32{keysymAltL, keysymTab}},
	{"Alt+Shift+Tab", []uint32{keysymAltL, keysymShiftL, keysymTab}},
}

// ErrNoEmbeddedSession means the chords need an embedded session: sending them relies
// on the session window living inside this application's window.
var ErrNoEmbeddedSession = errors.New("ui: sending keys needs the session embedded in the tab")

// sendSpecialKey types a chord into this panel's session.
//
// It works by focusing the embedded session window and faking the key events with
// XTEST, which is the only way to deliver a chord the local desktop would otherwise
// swallow: Ctrl+Alt+Del never reaches an application that is not the X server, and
// Alt+Tab is grabbed by the window manager.
func (p *sessionPanel) sendSpecialKey(chord specialChord) {
	if p.embedded == nil {
		p.setProblem("Special keys need the session embedded in the tab; " +
			"a separate FreeRDP window takes its own keyboard.")
		return
	}
	if err := p.embedded.SendKeys(chord.Keysyms); err != nil {
		p.setProblem("Could not send " + chord.Label + ": " + err.Error())
		return
	}
	p.setStatus("Sent " + chord.Label + " to the session.")
}

// specialKeysButton is the session-bar control that sends the first chord, which is
// the one users need and cannot type. It carries no label: a bar of five chords would
// push the session into horizontal scrolling, so the rest live in the View menu and
// the hover hint names the one this button sends.
func (p *sessionPanel) specialKeysButton() *hintButton {
	chord := specialChords[0]
	button := newHintButton(theme.ComputerIcon(), "Send "+chord.Label+" to the session", func() {
		p.sendSpecialKey(chord)
	}, p.screen.hint)
	button.Importance = widget.LowImportance
	return button
}

// phoneChords sends one of the chords from the View menu to the selected tab.
func (s *screen) sendChordToSelected(chord specialChord) {
	panel := s.selectedPanel()
	if panel == nil {
		s.setStatus("Open a session to send " + chord.Label + ".")
		return
	}
	panel.sendSpecialKey(chord)
}

// selectedPanel is the main panel of the selected tab, which is what a menu action
// without a click target acts on.
func (s *screen) selectedPanel() *sessionPanel {
	if s.tabs == nil {
		return nil
	}
	selected := s.tabs.Selected()
	for id, tab := range s.openTabs {
		if tab == selected {
			return s.panels[id]
		}
	}
	return nil
}
