package ui

import "fyne.io/fyne/v2"

// X11 keysyms for the keys that never produce text. The values are the ones in
// X11/keysymdef.h, which is what the RFB KeyEvent message carries.
var vncSpecialKeys = map[fyne.KeyName]uint32{
	fyne.KeyReturn: 0xff0d, fyne.KeyEnter: 0xff8d, fyne.KeyEscape: 0xff1b,
	fyne.KeyBackspace: 0xff08, fyne.KeyTab: 0xff09, fyne.KeyDelete: 0xffff,
	fyne.KeyInsert: 0xff63, fyne.KeyHome: 0xff50, fyne.KeyEnd: 0xff57,
	fyne.KeyPageUp: 0xff55, fyne.KeyPageDown: 0xff56,
	fyne.KeyLeft: 0xff51, fyne.KeyUp: 0xff52, fyne.KeyRight: 0xff53, fyne.KeyDown: 0xff54,
	fyne.KeyF1: 0xffbe, fyne.KeyF2: 0xffbf, fyne.KeyF3: 0xffc0, fyne.KeyF4: 0xffc1,
	fyne.KeyF5: 0xffc2, fyne.KeyF6: 0xffc3, fyne.KeyF7: 0xffc4, fyne.KeyF8: 0xffc5,
	fyne.KeyF9: 0xffc6, fyne.KeyF10: 0xffc7, fyne.KeyF11: 0xffc8, fyne.KeyF12: 0xffc9,
	"Menu": 0xff67, "PrintScreen": 0xff61, "CapsLock": 0xffe5,
}

// vncModifierKeys are the modifier keys, with the keysym each one sends. Ctrl, Alt
// and Super also change how the next letter is sent, so they are told apart from
// Shift, which only affects the text Fyne already types.
var vncModifierKeys = map[fyne.KeyName]struct {
	keysym  uint32
	changes bool
}{
	"LeftShift": {0xffe1, false}, "RightShift": {0xffe2, false},
	"LeftControl": {0xffe3, true}, "RightControl": {0xffe4, true},
	"LeftAlt": {0xffe9, true}, "RightAlt": {0xffea, true},
	"LeftSuper": {0xffeb, true}, "RightSuper": {0xffec, true},
}

// vncKeysym is the keysym for a key by name. Printable keys come back lower case
// (the key, not the character) and only matter while a modifier is held.
func vncKeysym(name fyne.KeyName) (keysym uint32, printable, ok bool) {
	if sym, found := vncSpecialKeys[name]; found {
		return sym, false, true
	}
	if mod, found := vncModifierKeys[name]; found {
		return mod.keysym, false, true
	}
	if name == fyne.KeySpace {
		return 0x20, true, true
	}
	runes := []rune(string(name))
	if len(runes) != 1 {
		return 0, false, false
	}
	r := runes[0]
	if r >= 'A' && r <= 'Z' {
		r += 'a' - 'A'
	}
	return vncRuneKeysym(r), true, true
}

// vncRuneKeysym maps a character to its keysym: Latin-1 is the character itself,
// and every other code point is 0x01000000 plus the code point.
func vncRuneKeysym(r rune) uint32 {
	if r < 0x100 {
		return uint32(r)
	}
	return 0x01000000 | uint32(r)
}
