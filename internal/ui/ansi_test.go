package ui

import (
	"testing"

	"fyne.io/fyne/v2/widget"
)

func newTestANSI(cols, rows int) (*ansiState, *widget.TextGrid) {
	grid := widget.NewTextGrid()
	return newANSIState(grid, cols, rows), grid
}

func rowText(grid *widget.TextGrid, row int) string {
	var b []rune
	for _, cell := range grid.Rows[row].Cells {
		b = append(b, cell.Rune)
	}
	return string(b)
}

func feed(s *ansiState, str string) {
	for i := 0; i < len(str); i++ {
		s.Feed(str[i])
	}
}

func TestFeed_PlainText_WritesRunesLeftToRight(t *testing.T) {
	s, grid := newTestANSI(10, 3)
	feed(s, "hello")

	if got := rowText(grid, 0)[:5]; got != "hello" {
		t.Errorf("row 0 = %q, want %q", got, "hello")
	}
}

func TestFeed_CRLF_MovesToNextRowAtColumnZero(t *testing.T) {
	s, grid := newTestANSI(10, 3)
	feed(s, "ab\r\ncd")

	if got := rowText(grid, 0)[:2]; got != "ab" {
		t.Errorf("row 0 = %q, want %q", got, "ab")
	}
	if got := rowText(grid, 1)[:2]; got != "cd" {
		t.Errorf("row 1 = %q, want %q", got, "cd")
	}
}

func TestFeed_LineFeedAlone_DoesNotResetColumn(t *testing.T) {
	// Real terminal semantics: \n moves down a row but keeps the column,
	// distinct from \r\n. Getting this backwards is a classic terminal
	// emulator bug.
	s, grid := newTestANSI(10, 3)
	feed(s, "ab\ncd")

	if got := rowText(grid, 1)[:4]; got != "  cd" {
		t.Errorf("row 1 = %q, want %q (cd starting at column 2, where \\n left the cursor)", got, "  cd")
	}
}

func TestFeed_Backspace_MovesCursorLeft(t *testing.T) {
	s, grid := newTestANSI(10, 3)
	feed(s, "ab\bc")

	if got := rowText(grid, 0)[:2]; got != "ac" {
		t.Errorf("row 0 = %q, want %q", got, "ac")
	}
}

func TestFeed_CursorPosition_PlacesTextAtGivenCell(t *testing.T) {
	s, grid := newTestANSI(10, 5)
	feed(s, "\x1b[3;4Hx") // 1-indexed row 3, col 4

	if got := rowText(grid, 2)[3]; got != 'x' {
		t.Errorf("cell (row 2, col 3) = %q, want 'x'", got)
	}
}

func TestFeed_CursorMovement_UpDownLeftRight(t *testing.T) {
	s, grid := newTestANSI(10, 5)
	feed(s, "\x1b[3;3H") // start at row 2, col 2 (0-indexed)
	feed(s, "\x1b[1B")   // down 1 -> row 3
	feed(s, "\x1b[2C")   // right 2 -> col 4
	feed(s, "x")
	if got := rowText(grid, 3)[4]; got != 'x' {
		t.Errorf("after down+right, cell (row 3, col 4) = %q, want 'x'", got)
	}

	feed(s, "\x1b[1A") // up 1 -> row 2
	feed(s, "\x1b[3D") // left 3 -> col 2
	feed(s, "y")
	if got := rowText(grid, 2)[2]; got != 'y' {
		t.Errorf("after up+left, cell (row 2, col 2) = %q, want 'y'", got)
	}
}

func TestFeed_EraseDisplay_Mode2_ClearsEverything(t *testing.T) {
	s, grid := newTestANSI(10, 3)
	feed(s, "line one\r\nline two")
	feed(s, "\x1b[2J")

	for r := 0; r < 3; r++ {
		for c, ch := range rowText(grid, r) {
			if ch != ' ' {
				t.Fatalf("cell (row %d, col %d) = %q after erase-all, want space", r, c, ch)
			}
		}
	}
}

func TestFeed_EraseLine_Mode0_ClearsFromCursorToEndOfLine(t *testing.T) {
	s, grid := newTestANSI(10, 1)
	feed(s, "abcdefgh")
	feed(s, "\x1b[1;4H") // cursor to column 4 (1-indexed) = index 3
	feed(s, "\x1b[0K")

	if got := rowText(grid, 0); got != "abc       " {
		t.Errorf("row after erase-to-end = %q, want %q", got, "abc       ")
	}
}

func TestFeed_SGR_ColorAppliesToSubsequentCellsAndResetClearsIt(t *testing.T) {
	s, grid := newTestANSI(10, 1)
	feed(s, "\x1b[31mred\x1b[0mplain")

	red := ansiPalette[1]
	for i, ch := range "red" {
		cell := grid.Rows[0].Cells[i]
		if cell.Rune != ch {
			t.Fatalf("cell %d rune = %q, want %q", i, cell.Rune, ch)
		}
		if cell.Style.TextColor() != red {
			t.Errorf("cell %d (in \"red\") color = %v, want the ANSI red palette entry", i, cell.Style.TextColor())
		}
	}
	for i := 3; i < 8; i++ {
		cell := grid.Rows[0].Cells[i]
		if cell.Style.TextColor() == red {
			t.Errorf("cell %d (in \"plain\", after reset) still has the red color", i)
		}
	}
}

func TestFeed_OSCSequence_IsDiscardedNotDisplayed(t *testing.T) {
	s, grid := newTestANSI(20, 1)
	feed(s, "\x1b]0;window title\x07visible")

	if got := rowText(grid, 0)[:7]; got != "visible" {
		t.Errorf("row 0 = %q, want %q (the OSC title payload must not leak into the grid)", got, "visible")
	}
}

func TestFeed_LineWrap_ContinuesOnNextRow(t *testing.T) {
	s, grid := newTestANSI(5, 3)
	feed(s, "abcdefg")

	if got := rowText(grid, 0); got != "abcde" {
		t.Errorf("row 0 = %q, want %q", got, "abcde")
	}
	if got := rowText(grid, 1)[:2]; got != "fg" {
		t.Errorf("row 1 = %q, want %q", got, "fg")
	}
}

func TestFeed_ScrollsWhenPastLastRow(t *testing.T) {
	s, grid := newTestANSI(10, 2)
	feed(s, "one\r\ntwo\r\nthree")

	if got := rowText(grid, 0)[:3]; got != "two" {
		t.Errorf("row 0 after scroll = %q, want %q (\"one\" should have scrolled off)", got, "two")
	}
	if got := rowText(grid, 1)[:5]; got != "three" {
		t.Errorf("row 1 after scroll = %q, want %q", got, "three")
	}
}
