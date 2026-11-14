package ui

import "fyne.io/fyne/v2"

// runOnUI runs f on the Fyne goroutine. Every callback that a worker goroutine
// (a pty reader, a dialer, a timer) hands back to the UI goes through it.
//
// It is a variable only so the tests can queue the callbacks and run them on the
// test goroutine: Fyne's test driver runs fyne.Do inline on the calling
// goroutine, which makes worker callbacks race with the test body, something the
// real driver's single UI goroutine never does.
var runOnUI = fyne.Do
