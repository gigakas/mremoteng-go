package ui

import (
	"testing"

	"github.com/BurntSushi/xgb/xproto"
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

// TestKeycodeForKeysym_FindsTheKeyInTheMapping covers the layout-independent part of
// sending a chord: the keycode comes from the X keyboard mapping, never from a
// hardcoded US table.
func TestKeycodeForKeysym_FindsTheKeyInTheMapping(t *testing.T) {
	// Two keysyms per keycode: unshifted then shifted, as X reports them.
	mapping := []xproto.Keysym{
		0x61, 0x41, // keycode 8: a / A
		0xffe3, 0xffe3, // keycode 9: Control_L
		0xffe9, 0xffe9, // keycode 10: Alt_L
		0xffff, 0xffff, // keycode 11: Delete
	}
	keycode, known := keycodeForKeysym(mapping, 2, 8, 0xffff)
	if !known || keycode != 11 {
		t.Errorf("Delete = %d, %v; want keycode 11", keycode, known)
	}
	if keycode, known := keycodeForKeysym(mapping, 2, 8, 0xffe3); !known || keycode != 9 {
		t.Errorf("Control_L = %d, %v; want keycode 9", keycode, known)
	}
	if _, known := keycodeForKeysym(mapping, 2, 8, 0xff0d); known {
		t.Error("a keysym outside the mapping was found")
	}
	if _, known := keycodeForKeysym(mapping, 0, 8, 0xffff); known {
		t.Error("an empty mapping produced a keycode")
	}
}
