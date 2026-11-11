package ui

import (
	"fmt"
	"image/color"
	"os"

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

// traceZoom records zoom decisions when MRE_TRACE_ZOOM is set, so a zoom that
// does nothing can be traced to the shortcut, the size change or the repaint
// without a debugger.
func traceZoom(format string, args ...any) {
	if os.Getenv("MRE_TRACE_ZOOM") == "" {
		return
	}
	file, err := os.OpenFile("/tmp/mremoteng-zoom.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	fmt.Fprintf(file, format+"\n", args...)
}
