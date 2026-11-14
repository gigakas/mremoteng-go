package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// modal is a modal popup that returns the keyboard where it was when it closes.
// Fyne's popup takes the canvas focus when it opens and never gives it back, so
// after any dialog the terminal stayed deaf to typing until it was clicked
// again.
type modal struct {
	*widget.PopUp

	screen   *screen
	previous fyne.Focusable
}

// showModal wraps content in a fixed-size modal popup centred on the window
// canvas and shows it. Callers keep the returned modal so the buttons inside the
// content can hide it, and hiding restores the previous focus.
func (s *screen) showModal(content fyne.CanvasObject, size fyne.Size) *modal {
	var previous fyne.Focusable
	if canvas := s.win.Canvas(); canvas != nil {
		previous = canvas.Focused()
	}
	popup := widget.NewModalPopUp(container.NewGridWrap(size, container.NewPadded(content)), s.win.Canvas())
	popup.Show()
	return &modal{PopUp: popup, screen: s, previous: previous}
}

// Hide closes the popup and gives the keyboard back to whatever had it before.
func (m *modal) Hide() {
	m.PopUp.Hide()
	if m.previous == nil || m.screen == nil || m.screen.win == nil {
		return
	}
	if canvas := m.screen.win.Canvas(); canvas != nil {
		canvas.Focus(m.previous)
	}
	m.previous = nil
}
