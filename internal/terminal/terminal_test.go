package terminal

import (
	"fmt"
	"strings"
	"testing"
)

func rowText(snapshot Snapshot, row int) string {
	var builder strings.Builder
	for _, cell := range snapshot.Cells[row] {
		if cell.Rune == 0 {
			builder.WriteRune(' ')
			continue
		}
		builder.WriteRune(cell.Rune)
	}
	return strings.TrimRight(builder.String(), " ")
}

func TestEmulator_WritePlainText_RendersOnFirstRow(t *testing.T) {
	emulator := New(20, 4)
	if _, err := emulator.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	snapshot := emulator.Snapshot()
	if got := rowText(snapshot, 0); got != "hello" {
		t.Errorf("row 0 = %q, want %q", got, "hello")
	}
	if snapshot.CursorX != 5 || snapshot.CursorY != 0 {
		t.Errorf("cursor = (%d,%d), want (5,0)", snapshot.CursorX, snapshot.CursorY)
	}
}

func TestEmulator_CarriageReturnAndNewline_OverwritesAndAdvances(t *testing.T) {
	emulator := New(20, 4)
	_, _ = emulator.Write([]byte("hello\rHE\r\nnext"))
	snapshot := emulator.Snapshot()
	if got := rowText(snapshot, 0); got != "HEllo" {
		t.Errorf("row 0 = %q, want %q", got, "HEllo")
	}
	if got := rowText(snapshot, 1); got != "next" {
		t.Errorf("row 1 = %q, want %q", got, "next")
	}
}

func TestEmulator_SGROverride_ReportsForegroundColor(t *testing.T) {
	emulator := New(20, 4)
	_, _ = emulator.Write([]byte("\x1b[31mred\x1b[0m plain"))
	snapshot := emulator.Snapshot()
	if got := snapshot.Cells[0][0].FG; got != 1 {
		t.Errorf("red cell FG = %d, want 1", got)
	}
	if got := snapshot.Cells[0][4].FG; got != DefaultFG {
		t.Errorf("plain cell FG = %d, want DefaultFG", got)
	}
}

func TestEmulator_EraseLine_ClearsCells(t *testing.T) {
	emulator := New(20, 4)
	_, _ = emulator.Write([]byte("hello world\r\x1b[K"))
	snapshot := emulator.Snapshot()
	if got := rowText(snapshot, 0); got != "" {
		t.Errorf("row 0 = %q, want it cleared", got)
	}
}

func TestEmulator_Resize_ChangesSnapshotSize(t *testing.T) {
	emulator := New(10, 3)
	emulator.Resize(40, 12)
	snapshot := emulator.Snapshot()
	if snapshot.Cols != 40 || snapshot.Rows != 12 {
		t.Fatalf("snapshot = %dx%d, want 40x12", snapshot.Cols, snapshot.Rows)
	}
	if len(snapshot.Cells) != 12 || len(snapshot.Cells[0]) != 40 {
		t.Errorf("cells = %dx%d, want 12x40", len(snapshot.Cells), len(snapshot.Cells[0]))
	}
}

func TestEmulator_CursorVisibility_DefaultsToVisible(t *testing.T) {
	emulator := New(10, 3)
	if !emulator.Snapshot().CursorOn {
		t.Error("cursor is hidden on a fresh emulator")
	}
}

func TestEmulator_Scrollback_KeepsLinesThatLeaveTheScreen(t *testing.T) {
	emulator := New(20, 3)
	for i := 0; i < 10; i++ {
		_, _ = emulator.Write([]byte(fmt.Sprintf("line %d\r\n", i)))
	}
	if emulator.HistoryLen() == 0 {
		t.Fatal("no scrollback was recorded")
	}
	text := emulator.BufferText()
	if !strings.Contains(text, "line 0") {
		t.Errorf("buffer = %q, want the first line retained in history", text)
	}
	if !strings.Contains(text, "line 9") {
		t.Errorf("buffer = %q, want the last line on screen", text)
	}
	lines := emulator.Lines()
	if len(lines) != emulator.HistoryLen()+3 {
		t.Errorf("lines = %d, want history plus the three screen rows", len(lines))
	}
}

func TestEmulator_WriteLargeBlock_RecordsTheWholeScrollback(t *testing.T) {
	// One write, many screens: a compiled build or a cat of a large file arrives
	// as a single block. The scrollback must record every line that left the
	// screen, not just the ones a small step could align.
	emulator := New(20, 3)
	var block strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&block, "line %03d\r\n", i)
	}
	if _, err := emulator.Write([]byte(block.String())); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := emulator.HistoryLen(); got == 0 {
		t.Fatal("a single large write recorded no scrollback")
	}
	// A three-row screen keeps the last line and the one above it, so the lines
	// that left the screen are N - rows + 1.
	if got, want := emulator.HistoryLen(), 200-3+1; got != want {
		t.Errorf("history = %d, want %d scrolled lines", got, want)
	}
	text := emulator.BufferText()
	if !strings.Contains(text, "line 000") {
		t.Errorf("buffer does not hold the first line: %q", text)
	}
	if !strings.Contains(text, "line 199") {
		t.Errorf("buffer does not hold the last line: %q", text)
	}
}

func TestEmulator_WriteFittingOnScreen_RecordsNoHistory(t *testing.T) {
	emulator := New(20, 3)
	if _, err := emulator.Write([]byte("first\r\nsecond\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := emulator.HistoryLen(); got != 0 {
		t.Errorf("history = %d, want nothing recorded when the output fits", got)
	}
}

func TestEmulator_WriteHugeBlock_KeepsHistoryBounded(t *testing.T) {
	emulator := New(20, 3)
	var block strings.Builder
	// Numbered lines: a run of identical lines makes every screen look the same,
	// which the tracker reads as a redraw and correctly ignores.
	for i := 0; i < historyLimit+500; i++ {
		fmt.Fprintf(&block, "line %04d\r\n", i)
	}
	if _, err := emulator.Write([]byte(block.String())); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := emulator.HistoryLen(); got != historyLimit {
		t.Errorf("history = %d, want it capped at %d", got, historyLimit)
	}
}

func TestEmulator_History_ReturnsScrollbackWithoutVisibleScreen(t *testing.T) {
	emulator := New(20, 3)
	var block strings.Builder
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&block, "line %d\r\n", i)
	}
	if _, err := emulator.Write([]byte(block.String())); err != nil {
		t.Fatalf("write: %v", err)
	}
	history := emulator.History()
	if len(history) != emulator.HistoryLen() {
		t.Fatalf("History() = %d lines, HistoryLen() = %d", len(history), emulator.HistoryLen())
	}
	if len(history) != 10-3+1 {
		t.Errorf("history = %d lines, want the eight that left the screen", len(history))
	}
	for _, line := range history {
		if strings.Contains(line, "line 9") {
			t.Errorf("history contains %q, which is still on screen", line)
		}
	}
	if got, want := len(emulator.Lines()), len(history)+3; got != want {
		t.Errorf("Lines() = %d, want history plus the three screen rows (%d)", got, want)
	}
}

func TestEmulator_Scrollback_IsBoundedAndClearScreenDoesNotDump(t *testing.T) {
	emulator := New(20, 3)
	for i := 0; i < historyLimit+50; i++ {
		_, _ = emulator.Write([]byte("x\r\n"))
	}
	if emulator.HistoryLen() > historyLimit {
		t.Fatalf("history = %d, want at most %d", emulator.HistoryLen(), historyLimit)
	}
}

func TestEmulator_RedrawWithoutScrolling_DoesNotDuplicateHistory(t *testing.T) {
	emulator := New(20, 3)
	_, _ = emulator.Write([]byte("prompt$ \r\n"))
	before := emulator.HistoryLen()
	for i := 0; i < 5; i++ {
		_, _ = emulator.Write([]byte("\rprompt$ "))
	}
	if got := emulator.HistoryLen(); got != before {
		t.Fatalf("history = %d, want %d after redraws", got, before)
	}
}

func TestEmulator_BufferText_HasNoRepeatedLinesFromOneScroll(t *testing.T) {
	emulator := New(20, 3)
	for i := 0; i < 6; i++ {
		_, _ = emulator.Write([]byte(fmt.Sprintf("row %d\r\n", i)))
	}
	text := emulator.BufferText()
	for i := 0; i < 4; i++ {
		line := fmt.Sprintf("row %d", i)
		if strings.Count(text, line+"\n") > 1 {
			t.Fatalf("buffer repeats %q: %q", line, text)
		}
	}
}

func TestEmulator_ScrolledLines_KeepTheirColours(t *testing.T) {
	emulator := New(10, 3)
	for _, chunk := range []string{"\x1b[31mred\x1b[0m\r\n", "b\r\n", "c\r\n", "d\r\n", "e"} {
		_, _ = emulator.Write([]byte(chunk))
	}

	cells := emulator.HistoryCells()
	if len(cells) != emulator.HistoryLen() || len(cells) == 0 {
		t.Fatalf("history cells = %d, history lines = %d", len(cells), emulator.HistoryLen())
	}
	first := cells[0][0]
	if first.Rune != 'r' || first.FG == DefaultFG {
		t.Errorf("first scrolled cell = %+v, want a red 'r'", first)
	}
}

func TestEmulator_EraseScrollback_DropsHistory(t *testing.T) {
	emulator := New(10, 3)
	for _, chunk := range []string{"a\r\n", "b\r\n", "c\r\n", "d\r\n", "e"} {
		_, _ = emulator.Write([]byte(chunk))
	}
	if emulator.HistoryLen() == 0 {
		t.Fatal("setup produced no history")
	}

	_, _ = emulator.Write([]byte("\x1b[H\x1b[2J\x1b[3J"))

	if emulator.HistoryLen() != 0 || len(emulator.HistoryCells()) != 0 {
		t.Errorf("history = %d lines, %d cells, want none after ESC[3J", emulator.HistoryLen(), len(emulator.HistoryCells()))
	}
}
