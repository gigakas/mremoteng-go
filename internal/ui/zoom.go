package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
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

// zoomLevels documents the discrete steps used by the shortcuts and menu.
