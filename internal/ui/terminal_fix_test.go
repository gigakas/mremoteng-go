package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestTermWidget_CellSize_TracksTheFontSize(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(80, 24, func([]byte) {}, nil)
	_ = test.WidgetRenderer(term)

	small := term.cellSize()
	if small.Width <= 0 || small.Height <= 0 {
		t.Fatalf("cell size = %v, want a measured cell", small)
	}
	// The cache must not go stale when the font changes.
	term.adjustZoom(6)
	large := term.cellSize()
	if large.Height <= small.Height {
		t.Errorf("cell height after zoom = %.1f, want more than %.1f", large.Height, small.Height)
	}
}

func TestTermWidget_Dragged_ThrottlesTheRepaintButDragEndAlwaysPaints(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(80, 24, func([]byte) {}, nil)
	window := test.NewWindow(term)
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(600, 400))

	start := time.Now()
	term.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(10, 10)}, Dragged: fyne.NewDelta(1, 1)})
	if term.lastDragPaint.IsZero() {
		t.Error("the first drag event did not paint")
	}
	first := term.lastDragPaint

	// A burst inside the interval must not repaint again.
	for i := 0; i < 5; i++ {
		term.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(20, 20)}, Dragged: fyne.NewDelta(1, 1)})
	}
	if !term.lastDragPaint.Equal(first) {
		t.Errorf("drag burst repainted again after %v, want it throttled", time.Since(start))
	}

	// Releasing always paints, so the final highlight is never lost.
	term.lastDragPaint = time.Now().Add(-2 * dragPaintInterval)
	term.DragEnd()
	if term.selecting {
		t.Error("DragEnd left the widget in selection mode")
	}
}
