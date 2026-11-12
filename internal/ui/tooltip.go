package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// hintDelay is how long the pointer has to rest on a button before its hint
// appears. Short enough to feel immediate, long enough not to flash while the
// pointer crosses the bar.
var hintDelay = 300 * time.Millisecond

// hintButton is a button that explains itself.
//
// Fyne has no tooltip support, and its popups cover the whole canvas with a
// background, which steals the hover from the button that opened them and makes
// a floating tooltip flicker. This repository already answers the same problem
// for tree rows by reporting in the status bar, so an icon button does the same:
// after hintDelay it hands its one line of text to the screen, and it clears it
// on leave. Every widget.Button behaviour is kept.
type hintButton struct {
	widget.Button

	tip  string
	hint func(string)
	// gen invalidates a pending hint when the pointer leaves before the delay.
	gen int
}

// newHintButton builds a label-less button with a hover hint and a tap action.
func newHintButton(icon fyne.Resource, tip string, tapped func(), hint func(string)) *hintButton {
	b := &hintButton{tip: tip, hint: hint}
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// MouseIn starts the delay; MouseOut cancels it and clears the hint. Both chain
// to widget.Button so the built-in hover highlight keeps working.
func (b *hintButton) MouseIn(event *desktop.MouseEvent) {
	b.Button.MouseIn(event)
	b.schedule()
}

func (b *hintButton) MouseMoved(event *desktop.MouseEvent) {
	b.Button.MouseMoved(event)
}

func (b *hintButton) MouseOut() {
	b.Button.MouseOut()
	b.cancel()
	b.emit("")
}

func (b *hintButton) schedule() {
	if b.hint == nil {
		return
	}
	b.gen++
	gen := b.gen
	time.AfterFunc(hintDelay, func() {
		fyne.Do(func() {
			if gen != b.gen {
				return
			}
			b.emit(b.tip)
		})
	})
}

func (b *hintButton) cancel() {
	b.gen++
}

func (b *hintButton) emit(text string) {
	if b.hint != nil {
		b.hint(text)
	}
}
