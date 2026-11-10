package ui_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/mRemoteNG/mremoteng-go/internal/ui"
)

func TestApplyTheme_Light_ForcesLightColorsRegardlessOfVariantAsked(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	ui.ApplyTheme(a, "light")
	got := a.Settings().Theme()

	light := theme.DefaultTheme().Color(theme.ColorNameBackground, theme.VariantLight)
	dark := theme.DefaultTheme().Color(theme.ColorNameBackground, theme.VariantDark)
	if light == dark {
		t.Fatal("test setup problem: default theme's light/dark background colors are equal")
	}

	if got := got.Color(theme.ColorNameBackground, theme.VariantDark); got != light {
		t.Errorf("Color(Background, VariantDark) after ApplyTheme(light) = %v, want the light color %v (forced, ignoring the requested variant)", got, light)
	}
	if got := got.Color(theme.ColorNameBackground, theme.VariantLight); got != light {
		t.Errorf("Color(Background, VariantLight) after ApplyTheme(light) = %v, want %v", got, light)
	}
}

func TestApplyTheme_Dark_ForcesDarkColorsRegardlessOfVariantAsked(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	ui.ApplyTheme(a, "dark")
	got := a.Settings().Theme()

	dark := theme.DefaultTheme().Color(theme.ColorNameBackground, theme.VariantDark)

	if got := got.Color(theme.ColorNameBackground, theme.VariantLight); got != dark {
		t.Errorf("Color(Background, VariantLight) after ApplyTheme(dark) = %v, want the dark color %v (forced, ignoring the requested variant)", got, dark)
	}
}

func TestApplyTheme_SystemOrUnrecognized_UsesFynesDefaultTheme(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	for _, name := range []string{"system", "", "not-a-real-theme"} {
		ui.ApplyTheme(a, name)
		if got := a.Settings().Theme(); got != theme.DefaultTheme() {
			t.Errorf("ApplyTheme(%q): Settings().Theme() = %v, want theme.DefaultTheme()", name, got)
		}
	}
}
