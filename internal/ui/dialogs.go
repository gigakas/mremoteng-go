package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showModal wraps content in a fixed-size modal popup centred on the window
// canvas and shows it. Callers keep the returned popup so the buttons inside
// the content can hide it.
func (s *screen) showModal(content fyne.CanvasObject, size fyne.Size) *widget.PopUp {
	popup := widget.NewModalPopUp(container.NewGridWrap(size, container.NewPadded(content)), s.win.Canvas())
	popup.Show()
	return popup
}
