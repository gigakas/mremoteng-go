package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ThemeChoices lists the values ApplyTheme understands. ShowOptionsDialog
// (stage 3.5) offers exactly these in its Theme select — kept here rather
// than duplicated there now that this stage owns what the values actually
// do.
var ThemeChoices = []string{"system", "light", "dark"}

// ApplyTheme sets a's active theme from one of ThemeChoices. "system"
// (and any unrecognized value, so a corrupt or future-version settings
// file degrades gracefully) uses Fyne's own theme.DefaultTheme, which
// already tracks the OS light/dark preference on its own. "light"/"dark"
// force a variant regardless of OS preference, via forcedVariantTheme —
// not theme.DarkTheme()/LightTheme() directly, since both are documented
// as deprecated in favor of exactly this "set a custom theme" pattern.
func ApplyTheme(a fyne.App, name string) {
	switch name {
	case "light":
		a.Settings().SetTheme(&forcedVariantTheme{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
	case "dark":
		a.Settings().SetTheme(&forcedVariantTheme{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	default:
		a.Settings().SetTheme(theme.DefaultTheme())
	}
}

// forcedVariantTheme wraps another fyne.Theme, overriding every Color
// lookup to use a fixed variant instead of whatever the caller (Fyne's
// own rendering code, always passing the current system/app variant)
// asks for. Font/Icon/Size are unaffected — Fyne's built-in themes don't
// vary those by variant, so passing them through to the wrapped theme
// unchanged matches the built-in themes' own behavior.
type forcedVariantTheme struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (t *forcedVariantTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return t.Theme.Color(name, t.variant)
}
