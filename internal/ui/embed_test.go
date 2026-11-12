package ui

import (
	"testing"
)

func TestIntersect_ClampsToVisibleBounds(t *testing.T) {
	area := rect{x: 100, y: 100, width: 500, height: 400}
	bounds := rect{x: 0, y: 0, width: 400, height: 300}
	got := intersect(area, bounds)
	if got != (rect{x: 100, y: 100, width: 300, height: 200}) {
		t.Fatalf("intersect = %+v, want the visible part only", got)
	}
	if empty := intersect(area, rect{x: 900, y: 900, width: 10, height: 10}); empty.width != 0 || empty.height != 0 {
		t.Fatalf("intersect = %+v, want an empty rect", empty)
	}
}

func TestScaleRect_ConvertsLogicalPointsToPixels(t *testing.T) {
	got := scaleRect(rect{x: 10, y: 20, width: 100, height: 50}, 2)
	if got != (rect{x: 20, y: 40, width: 200, height: 100}) {
		t.Fatalf("scaleRect = %+v, want doubled geometry", got)
	}
	if got := scaleRect(rect{x: 1, y: 1, width: 1, height: 1}, 0); got.width != 1 {
		t.Fatalf("scaleRect with zero scale = %+v, want it treated as 1", got)
	}
}
