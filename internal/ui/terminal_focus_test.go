package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func TestTermWidget_ImplementsMouseable_DriverCallsMouseDown(t *testing.T) {
	var widget fyne.CanvasObject = &termWidget{}
	if _, ok := widget.(desktop.Mouseable); !ok {
		t.Fatal("termWidget is not a desktop.Mouseable: Fyne never calls MouseDown, so a click that is not a clean tap leaves the terminal without the keyboard")
	}
}

func TestTermWidget_MouseDown_TakesKeyboardFocus(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	first := newTermWidget(40, 10, nil, nil)
	second := newTermWidget(40, 10, nil, nil)
	window := test.NewWindow(container.NewWithoutLayout(first, second))
	t.Cleanup(window.Close)
	first.Resize(fyne.NewSize(300, 200))
	second.Resize(fyne.NewSize(300, 200))
	window.Canvas().Focus(first)

	second.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})

	if window.Canvas().Focused() != fyne.Focusable(second) {
		t.Error("pressing the mouse on a terminal must give it the keyboard")
	}
}
