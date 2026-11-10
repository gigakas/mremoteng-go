package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

<<<<<<< HEAD
// Palette: a calm dark surface set with one accent, and a light variant that
// keeps the same hierarchy.
var (
	darkBackground  = color.NRGBA{R: 0x1b, G: 0x1d, B: 0x21, A: 0xff}
	darkPanel       = color.NRGBA{R: 0x23, G: 0x26, B: 0x2b, A: 0xff}
	darkInput       = color.NRGBA{R: 0x2c, G: 0x30, B: 0x36, A: 0xff}
	darkHover       = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x0d}
	darkSeparator   = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x12}
	darkForeground  = color.NRGBA{R: 0xe6, G: 0xe8, B: 0xeb, A: 0xff}
	lightBackground = color.NRGBA{R: 0xf4, G: 0xf5, B: 0xf7, A: 0xff}
	lightPanel      = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	lightInput      = color.NRGBA{R: 0xf0, G: 0xf1, B: 0xf4, A: 0xff}
	lightHover      = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x0a}
	lightSeparator  = color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x12}
	lightForeground = color.NRGBA{R: 0x1f, G: 0x23, B: 0x28, A: 0xff}
	accentColor     = color.NRGBA{R: 0x3d, G: 0x7e, B: 0xff, A: 0xff}
	accentSoft      = color.NRGBA{R: 0x3d, G: 0x7e, B: 0xff, A: 0x33}
)

// soberTheme keeps the default icons and fonts but applies a tighter type scale
// and a restrained palette, so the dense tree and Config panel read as a tool.
type soberTheme struct {
	base fyne.Theme
}

// SoberTheme returns the application theme.
func SoberTheme() fyne.Theme {
	return soberTheme{base: theme.DefaultTheme()}
}

func (t soberTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	dark := variant != theme.VariantLight
	switch name {
	case theme.ColorNameBackground:
		return pick(dark, darkBackground, lightBackground)
	case theme.ColorNameOverlayBackground, theme.ColorNameMenuBackground:
		return pick(dark, darkPanel, lightPanel)
	case theme.ColorNameInputBackground:
		return pick(dark, darkInput, lightInput)
	case theme.ColorNameForeground:
		return pick(dark, darkForeground, lightForeground)
	case theme.ColorNameHover:
		return pick(dark, darkHover, lightHover)
	case theme.ColorNameSeparator:
		return pick(dark, darkSeparator, lightSeparator)
	case theme.ColorNamePrimary:
		return accentColor
	case theme.ColorNameSelection:
		return accentSoft
	default:
		return t.base.Color(name, variant)
	}
}

func (t soberTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t soberTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t soberTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 12
	case theme.SizeNamePadding:
		return 3
	case theme.SizeNameInnerPadding:
		return 5
	case theme.SizeNameInlineIcon:
		return 18
	case theme.SizeNameInputRadius:
		return 6
	case theme.SizeNameSelectionRadius:
		return 6
	default:
		return t.base.Size(name)
	}
}

func pick(dark bool, darkValue, lightValue color.Color) color.Color {
	if dark {
		return darkValue
	}
	return lightValue
=======
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
>>>>>>> 876e6198187aa7ca8eeacb02d85b84d8328d4e68
}
