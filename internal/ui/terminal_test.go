package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	telnetclient "github.com/mRemoteNG/mremoteng-go/internal/protocol/telnet"
	"strings"
	"testing"
)

func TestTermWidget_TypedInput_SendsTerminalSequences(t *testing.T) {
	var sent []byte
	term := newTermWidget(40, 10, func(data []byte) { sent = append(sent, data...) }, nil)
	term.TypedRune('a')
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	want := "a\r\x1b[A\x7f\t"
	if string(sent) != want {
		t.Fatalf("sent = %q, want %q", sent, want)
	}
}

func TestTermWidget_AcceptsTab_KeepsCompletionInsideTerminal(t *testing.T) {
	var tabbable fyne.Tabbable = newTermWidget(40, 10, nil, nil)
	if !tabbable.AcceptsTab() {
		t.Fatal("AcceptsTab = false, Fyne will move focus instead of completing")
	}
}

func TestTermWidget_Write_PaintsScreen(t *testing.T) {
	term := newTermWidget(40, 5, nil, nil)
	term.CreateRenderer()
	term.write([]byte("hello"))
	if len(term.grid.Rows) == 0 || len(term.grid.Rows[0].Cells) < 5 {
		t.Fatalf("grid rows = %d", len(term.grid.Rows))
	}
	var row strings.Builder
	for _, cell := range term.grid.Rows[0].Cells {
		row.WriteRune(cell.Rune)
	}
	if !strings.HasPrefix(row.String(), "hello") {
		t.Errorf("row 0 = %q, want it to start with hello", row.String())
	}
}

func TestTermWidget_Selection_CopiesOnlyHighlightedCells(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("hello world\r\nsecond line\r\nthird"))
	// Select from the middle of the first line to the start of the second.
	term.selStart = cellPos{col: 6, row: 0}
	term.selEnd = cellPos{col: 5, row: 1}
	if !term.selected(6, 0) || term.selected(5, 0) {
		t.Error("selection does not cover the expected first-line cells")
	}
	if !term.selected(5, 1) || term.selected(6, 1) {
		t.Error("selection end is wrong on the second line")
	}
	text := term.screenText()
	if text != "world\nsecond" {
		t.Fatalf("selected text = %q, want only the highlighted range", text)
	}

	// A reversed drag normalises.
	term.selStart = cellPos{col: 5, row: 1}
	term.selEnd = cellPos{col: 6, row: 0}
	if got := term.screenText(); got != "world\nsecond" {
		t.Fatalf("reversed selection = %q", got)
	}
}

func TestTermWidget_DragWithoutMouseDown_SelectsRange(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("hello world\r\nsecond line\r\n"))
	term.Resize(fyne.NewSize(300, 120))
	cell := fyne.MeasureText("M", theme.TextSize(), fyne.TextStyle{Monospace: true})
	// A single drag event from the left edge across ten cells (no MouseDown).
	term.Dragged(&fyne.DragEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(cell.Width*10, cell.Height*0.5)},
		Dragged:    fyne.Delta{DX: cell.Width * 10},
	})
	term.DragEnd()
	text := term.screenText()
	if !strings.HasPrefix(text, "hello worl") || strings.Contains(text, "second") {
		t.Fatalf("drag selection = %q, want the first line range", text)
	}
}

func TestTermWidget_DragPastTopEdge_ScrollsWhileSelecting(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 3, nil, nil)
	term.CreateRenderer()
	for i := 0; i < 12; i++ {
		term.write([]byte("line " + string(rune('a'+i)) + "\r\n"))
	}
	term.Resize(fyne.NewSize(300, 120))
	history := term.emulator.HistoryLen()
	if history == 0 {
		t.Fatal("no history to scroll through")
	}

	// Start a selection inside the view, then drag above the top edge.
	inner := fyne.NewPos(10, 10)
	term.selStart = term.cellAt(inner)
	term.selEnd = term.selStart
	term.selecting = true
	term.Dragged(&fyne.DragEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(10, -5)},
		Dragged:    fyne.Delta{DY: -15},
	})
	if term.scrollOffset == 0 {
		t.Fatal("dragging past the top did not scroll the view")
	}
	if term.selEnd.row >= term.selStart.row {
		t.Errorf("selection end row = %d, want a row above the start", term.selEnd.row)
	}
	if text := term.screenText(); text == "" || !strings.Contains(text, "\n") {
		t.Errorf("selected text = %q, want a multi-line selection from the scrollback", text)
	}
}

func TestTermWidget_ControlShortcuts_SendControlBytes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	term := newTermWidget(20, 4, func(data []byte) { sent = append(sent, data...) }, nil)

	term.TypedShortcut(&fyne.ShortcutSelectAll{})
	if len(sent) != 1 || sent[0] != 0x01 {
		t.Fatalf("Ctrl-A sent %v, want 0x01", sent)
	}
	sent = nil
	term.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierControl})
	if len(sent) != 1 || sent[0] != 0x0c {
		t.Fatalf("Ctrl-L sent %v, want 0x0c", sent)
	}
	sent = nil
	term.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyL})
	if len(sent) != 0 {
		t.Errorf("Ctrl-less shortcut sent %v, want nothing", sent)
	}
}

func TestControlCodes_CoverTheShellCombinations(t *testing.T) {
	want := map[fyne.KeyName]byte{
		fyne.KeyA: 0x01, fyne.KeyD: 0x04, fyne.KeyE: 0x05, fyne.KeyK: 0x0b,
		fyne.KeyL: 0x0c, fyne.KeyR: 0x12, fyne.KeyU: 0x15, fyne.KeyW: 0x17, fyne.KeyZ: 0x1a,
	}
	for key, code := range want {
		if got := controlByte(key); got != code {
			t.Errorf("controlByte(%s) = %#x, want %#x", key, got, code)
		}
	}
}

func TestTermWidget_ArrowKeysNavigateHistoryWhenScrolled(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	term := newTermWidget(20, 3, func(data []byte) { sent = append(sent, data...) }, nil)
	term.CreateRenderer()
	for i := 0; i < 10; i++ {
		term.write([]byte("line " + string(rune('a'+i)) + "\r\n"))
	}
	term.Resize(fyne.NewSize(300, 120))

	// At the bottom the arrow goes to the shell (command history).
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if string(sent) != "\x1b[A" {
		t.Fatalf("sent = %q, want the up sequence at the bottom", sent)
	}
	sent = nil

	// PageUp browses the scrollback without reaching the shell.
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyPageUp})
	if len(sent) != 0 {
		t.Errorf("PageUp sent %q, want local scrolling", sent)
	}
	if term.scrollOffset == 0 {
		t.Fatal("PageUp did not scroll the view")
	}

	before := term.scrollOffset
	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if len(sent) != 0 {
		t.Errorf("up sent %q while scrolled, want local scrolling only", sent)
	}
	if term.scrollOffset != before+1 {
		t.Errorf("offset = %d, want one line further back", term.scrollOffset)
	}

	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
	if term.scrollOffset != 0 {
		t.Errorf("End did not return to the bottom: %d", term.scrollOffset)
	}
	if len(sent) != 0 {
		t.Errorf("End sent %q, want it consumed locally", sent)
	}
}

func TestTermWidget_AdjustZoom_ClampsAndUpdatesTheme(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	start := term.textSize()

	term.adjustZoom(2)
	if term.textSize() != start+2 {
		t.Fatalf("size = %.0f, want %.0f", term.textSize(), start+2)
	}
	if term.zoom == nil || term.zoom.Size(theme.SizeNameText) != start+2 {
		t.Errorf("zoom theme text size = %v, want %.0f", term.zoom, start+2)
	}
	for i := 0; i < 50; i++ {
		term.adjustZoom(1)
	}
	if term.textSize() != maxFontSize {
		t.Errorf("size = %.0f, want the maximum %d", term.textSize(), maxFontSize)
	}
	for i := 0; i < 60; i++ {
		term.adjustZoom(-1)
	}
	if term.textSize() != minFontSize {
		t.Errorf("size = %.0f, want the minimum %d", term.textSize(), minFontSize)
	}
}

func TestZoomTheme_OverridesOnlyTextSize(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	z := zoomTheme{base: theme.DefaultTheme(), size: 20}
	if got := z.Size(theme.SizeNameText); got != 20 {
		t.Fatalf("text size = %.0f, want 20", got)
	}
	if got, want := z.Size(theme.SizeNamePadding), theme.DefaultTheme().Size(theme.SizeNamePadding); got != want {
		t.Errorf("padding = %.0f, want it untouched (%0.f)", got, want)
	}
}

func TestTermWidget_Zoom_ScalesBothAxesWithoutRebuildingTheWidget(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	// test.WidgetRenderer goes through Fyne's own renderer cache, like the
	// real driver does; calling term.CreateRenderer() directly here would
	// bypass that cache, so the t.Refresh() inside rebuildGrid would find
	// nothing cached and build yet another renderer (and grid) itself,
	// masking whether the zoom step did.
	test.WidgetRenderer(term)
	before := term.grid
	term.Resize(fyne.NewSize(400, 200))
	oldCols, oldRows := term.cols, term.rows

	term.adjustZoom(6)
	term.fit(fyne.NewSize(400, 200))
	term.renderer.Layout(fyne.NewSize(400, 200))

	// Fyne's TextGrid re-measures its cell size from the theme on every
	// Refresh, so a zoom step reuses the same grid instance instead of
	// tearing down and rebuilding the widget (which is what made each zoom
	// step visibly stall on the GPU painter).
	if term.grid != before {
		t.Fatal("zoom replaced the grid instance instead of refreshing it in place")
	}
	if term.cols >= oldCols || term.rows >= oldRows {
		t.Errorf("grid = %dx%d, want fewer columns and rows at a larger font (%dx%d)", term.cols, term.rows, oldCols, oldRows)
	}
	if term.cols < 5 || term.rows < 2 {
		t.Errorf("grid collapsed to %dx%d", term.cols, term.rows)
	}
}

func TestTermWidget_CtrlCInterruptsAndCtrlVPastes(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	interrupts := 0
	term := newTermWidget(20, 4, func(data []byte) { sent = append(sent, data...) }, nil)
	term.onInterrupt = func() { interrupts++ }

	term.TypedShortcut(&fyne.ShortcutCopy{})
	if interrupts != 1 {
		t.Fatalf("Ctrl-C interrupts = %d, want 1", interrupts)
	}
	if len(sent) != 0 {
		t.Errorf("Ctrl-C sent %v, want no bytes from the shortcut path", sent)
	}
	term.sendBytes([]byte{0x03})
	if len(sent) != 1 || sent[0] != 0x03 {
		t.Errorf("sent = %v, want the interrupt byte", sent)
	}
}

func TestTermWidget_Scrollback_MovesThroughHistory(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 3, nil, nil)
	term.CreateRenderer()
	for i := 0; i < 10; i++ {
		term.write([]byte("line " + string(rune('0'+i)) + "\r\n"))
	}
	if term.emulator.HistoryLen() == 0 {
		t.Fatal("no history recorded")
	}
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: 2}})
	if term.scrollOffset != 2 {
		t.Fatalf("offset = %d, want 2", term.scrollOffset)
	}
	for i := 0; i < 50; i++ {
		term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: 5}})
	}
	if term.scrollOffset != term.emulator.HistoryLen() {
		t.Errorf("offset = %d, want the history length", term.scrollOffset)
	}
	term.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: -100}})
	if term.scrollOffset != 0 {
		t.Errorf("offset = %d, want it clamped to zero", term.scrollOffset)
	}
}

func TestTermWidget_RightClickCopiesSelection(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("pick me\r\nother\r\n"))
	var copied string
	term.onCopy = func(text string) { copied = text }
	term.selStart = cellPos{col: 0, row: 0}
	term.selEnd = cellPos{col: 6, row: 0}
	term.TappedSecondary(&fyne.PointEvent{})
	if copied != "pick me" {
		t.Fatalf("right click copied %q, want the selection", copied)
	}
	if _, _, hasSelection := term.selectionRange(); hasSelection {
		t.Error("right click left the selection active")
	}
}

func TestCopyTerminalScreen_PutsTextInClipboard(t *testing.T) {
	s := newTestScreen(t, &fakeStore{})
	clipboard := &fakeClipboard{}
	s.clipboard = func() fyne.Clipboard { return clipboard }
	stream := &fakeStream{}
	panel := s.newSessionPanel(sampleProfile(t, "T", "", "TELNET", "t.example.com", 23))
	panel.startStream(stream, telnetclient.ModeTelnet)
	panel.term.write([]byte("copied text\r\n"))
	panel.copyTerminalScreen()
	if !strings.Contains(clipboard.content, "copied text") {
		t.Fatalf("clipboard = %q, want the terminal text", clipboard.content)
	}
	if !strings.Contains(s.status.Text, "copied") {
		t.Errorf("status = %q, want a copy confirmation", s.status.Text)
	}
}

func TestTermWidget_DoubleTapSelectsLine(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("first\r\nsecond\r\n"))
	term.Resize(fyne.NewSize(200, 80))
	term.DoubleTapped(&fyne.PointEvent{Position: fyne.NewPos(2, 2)})
	text := term.screenText()
	if !strings.HasPrefix(text, "first") || strings.Contains(text, "second") {
		t.Fatalf("double tap selection = %q, want the first line", text)
	}
}

func TestTerminalScreenText_ReturnsVisibleLines(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(20, 4, nil, nil)
	term.CreateRenderer()
	term.write([]byte("first\r\nsecond   \r\n"))
	text := term.screenText()
	if !strings.Contains(text, "first") || !strings.Contains(text, "second") {
		t.Fatalf("screenText = %q, want both lines", text)
	}
	if strings.HasSuffix(text, "\n") {
		t.Errorf("screenText = %q, want trailing blank lines trimmed", text)
	}
}

func TestTermWidget_EnterWhileScrolledBack_ReturnsToTheLiveLine(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var sent []byte
	term := newTermWidget(20, 3, func(data []byte) { sent = append(sent, data...) }, nil)
	term.CreateRenderer()
	for i := 0; i < 12; i++ {
		term.write([]byte("line " + string(rune('a'+i)) + "\r\n"))
	}
	term.Resize(fyne.NewSize(300, 120))
	term.scrollOffset = 3

	term.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})

	if term.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0 after Enter", term.scrollOffset)
	}
	if string(sent) != "\r" {
		t.Errorf("sent = %q, want the Enter byte", sent)
	}
}
