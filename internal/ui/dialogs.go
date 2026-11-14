package ui

import (
	"sync"

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
	closed   bool
}

// showModal wraps content in a fixed-size modal popup centred on the window
// canvas and shows it. Callers keep the returned modal so the buttons inside the
// content can hide it, and hiding restores the previous focus.
func (s *screen) showModal(content fyne.CanvasObject, size fyne.Size) *modal {
	var previous fyne.Focusable
	if canvas := s.win.Canvas(); canvas != nil {
		previous = canvas.Focused()
	}
	s.beginModal()
	popup := widget.NewModalPopUp(container.NewGridWrap(size, container.NewPadded(content)), s.win.Canvas())
	popup.Show()
	return &modal{PopUp: popup, screen: s, previous: previous}
}

// Hide closes the popup, gives the keyboard back to whatever had it before and, once
// no dialog is left, brings the embedded session windows back.
func (m *modal) Hide() {
	if m.closed {
		return
	}
	m.closed = true
	m.PopUp.Hide()
	if m.screen != nil {
		m.screen.endModal()
	}
	if m.previous == nil || m.screen == nil || m.screen.win == nil {
		return
	}
	if canvas := m.screen.win.Canvas(); canvas != nil {
		canvas.Focus(m.previous)
	}
	m.previous = nil
}

// showMessage reports something the user must not miss, in a modal with a single
// button. A status line is invisible behind the window that raised it, which is how
// the host-key and FreeRDP failures stayed unnoticed.
func (s *screen) showMessage(title, message string) {
	body := widget.NewLabel(message)
	body.Wrapping = fyne.TextWrapWord
	var popup *modal
	content := container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		body,
		container.NewHBox(widget.NewButton("Close", func() { popup.Hide() })),
	)
	popup = s.showModal(content, fyne.NewSize(500, 220))
}

// beginModal keeps the native session windows out of the way while a dialog is open.
// An embedded RDP session is an X11 child window, so X composites it above everything
// this process draws into the same window: every dialog — ours or Fyne's own — opens
// behind it and the user has to disconnect to read one. The session keeps running;
// only its window is unmapped, and the last dialog to close maps it back.
func (s *screen) beginModal() {
	if s.modalDepth == 0 {
		s.setEmbeddedSessionsVisible(false)
	}
	s.modalDepth++
}

// endModal brings the session windows back once the last dialog has closed.
func (s *screen) endModal() {
	s.modalDepth--
	if s.modalDepth > 0 {
		return
	}
	s.modalDepth = 0
	s.setEmbeddedSessionsVisible(true)
}

// modalGuard brackets a dialog this package does not own, Fyne's file chooser being
// the only one, and returns the function that ends the bracket exactly once: the
// chooser reports both a pick and a cancel through the same callback.
func (s *screen) modalGuard() func() {
	s.beginModal()
	var once sync.Once
	return func() { once.Do(s.endModal) }
}

// showConfirm asks a yes/no question in a dialog of our own instead of Fyne's, which
// draws straight into the canvas and would therefore land behind an embedded session.
func (s *screen) showConfirm(title, message string, onConfirm, onDecline func()) {
	body := widget.NewLabel(message)
	body.Wrapping = fyne.TextWrapWord
	var popup *modal
	answer := func(accepted bool) {
		if accepted {
			// Run the action first: it may open another dialog, and closing this one
			// before it would map the session for a frame in between.
			onConfirm()
			popup.Hide()
			return
		}
		popup.Hide()
		if onDecline != nil {
			onDecline()
		}
	}
	content := container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		body,
		container.NewHBox(
			widget.NewButton("Confirm", func() { answer(true) }),
			widget.NewButton("Cancel", func() { answer(false) }),
		),
	)
	popup = s.showModal(content, fyne.NewSize(460, 200))
}
