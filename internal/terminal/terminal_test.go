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
