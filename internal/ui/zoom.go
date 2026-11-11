package ui

import (
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// zoomTheme overrides only the text size, so the terminal can zoom without
// touching the rest of the window.
type zoomTheme struct {
	base fyne.Theme
	size float32
}

func (t zoomTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.base.Color(name, variant)
}

func (t zoomTheme) Font(style fyne.TextStyle) fyne.Resource { return t.base.Font(style) }

func (t zoomTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return t.base.Icon(name) }

func (t zoomTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return t.size
	}
	return t.base.Size(name)
}

// controlPressed reports whether Ctrl is held, by asking the X server: Fyne's
// scroll events carry no modifier information.
func controlPressed() bool {
	if os.Getenv("DISPLAY") == "" {
		return false
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return false
	}
	defer conn.Close()
	setup := xproto.Setup(conn)
	root := setup.DefaultScreen(conn).Root
	reply, err := xproto.QueryPointer(conn, root).Reply()
	if err != nil {
		return false
	}
	return reply.Mask&xproto.ModMaskControl != 0
}
