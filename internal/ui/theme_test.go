package ui

import (
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// luminance is the WCAG relative luminance of a colour.
func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	channel := func(v uint32) float64 {
		s := float64(v) / 65535
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}

// contrastRatio is the WCAG contrast between two colours, from 1 to 21.
func contrastRatio(a, b color.Color) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// TestSoberTheme_MutedText_StaysReadable guards the reported problem: hints and
// secondary labels were rendered with Fyne's translucent disabled colour, which
// on this palette was barely visible. Muted text must clear WCAG AA for body
// text while staying visibly weaker than the foreground, so it still reads as
// secondary.
func TestSoberTheme_MutedText_StaysReadable(t *testing.T) {
	th := SoberTheme()
	cases := []struct {
		name    string
		variant fyne.ThemeVariant
	}{
		{"dark", theme.VariantDark},
		{"light", theme.VariantLight},
	}
	for _, tc := range cases {
		muted := th.Color(theme.ColorNameDisabled, tc.variant)
		foreground := th.Color(theme.ColorNameForeground, tc.variant)
		background := th.Color(theme.ColorNameBackground, tc.variant)

		if got := contrastRatio(muted, background); got < 4.5 {
			t.Errorf("%s muted text contrast = %.1f:1, want at least 4.5:1", tc.name, got)
		}
		if contrastRatio(muted, background) >= contrastRatio(foreground, background) {
			t.Errorf("%s muted text is not weaker than the foreground", tc.name)
		}
		// Placeholders share the muted colour, so they must not regress either.
		if muted != th.Color(theme.ColorNamePlaceHolder, tc.variant) {
			t.Errorf("%s placeholder does not use the muted colour", tc.name)
		}
	}
}
