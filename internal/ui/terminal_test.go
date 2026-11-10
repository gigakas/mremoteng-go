package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

func TestNewTerminal_DefaultsToSSHPTYSize(t *testing.T) {
	term := ui.NewTerminal()
	cols, rows := term.CharSize()
	if cols != 80 || rows != 24 {
		t.Errorf("CharSize() = (%d, %d), want (80, 24) to match the SSH backend's PTY defaults", cols, rows)
	}
}

func TestTerminal_Write_FeedsTheANSIInterpreter(t *testing.T) {
	term := ui.NewTerminal()
	if _, err := term.Write([]byte("hello\r\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Write's own correctness (ANSI interpretation) is covered exhaustively
	// by ansi_test.go; this just checks the widget wires Write through to
	// it and reports success.
}

func TestTerminal_TypedRune_CallsOnInputWithUTF8Bytes(t *testing.T) {
	term := ui.NewTerminal()
	var got []byte
	term.OnInput = func(b []byte) { got = b }

	term.TypedRune('é')

	if string(got) != "é" {
		t.Errorf("OnInput got %q, want %q", got, "é")
	}
}

func TestTerminal_TypedKey_MapsSpecialKeysToControlSequences(t *testing.T) {
	term := ui.NewTerminal()
	var got []byte
	term.OnInput = func(b []byte) { got = b }

	cases := []struct {
		key  fyne.KeyName
		want string
	}{
		{fyne.KeyReturn, "\r"},
		{fyne.KeyBackspace, "\x7f"},
		{fyne.KeyTab, "\t"},
		{fyne.KeyEscape, "\x1b"},
		{fyne.KeyUp, "\x1b[A"},
		{fyne.KeyDown, "\x1b[B"},
		{fyne.KeyLeft, "\x1b[D"},
		{fyne.KeyRight, "\x1b[C"},
	}
	for _, c := range cases {
		got = nil
		term.TypedKey(&fyne.KeyEvent{Name: c.key})
		if string(got) != c.want {
			t.Errorf("TypedKey(%s): OnInput got %q, want %q", c.key, got, c.want)
		}
	}
}

func TestTerminal_TypedKey_UnmappedKeyDoesNotCallOnInput(t *testing.T) {
	term := ui.NewTerminal()
	called := false
	term.OnInput = func(b []byte) { called = true }

	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyF1})

	if called {
		t.Error("OnInput was called for an unmapped key")
	}
}

func TestTerminal_ImplementsFyneWidgetInterfaces(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	win := test.NewWindow(nil)
	defer win.Close()

	term := ui.NewTerminal()
	win.SetContent(term)

	var (
		_ fyne.Widget    = term
		_ fyne.Focusable = term
		_ fyne.Tappable  = term
	)
	// None of these should panic, including Tapped, which requests focus
	// via the canvas the widget is now attached to.
	term.FocusGained()
	term.FocusLost()
	term.Tapped(&fyne.PointEvent{})
}
