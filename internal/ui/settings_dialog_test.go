package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/mRemoteNG/mremoteng-go/internal/profiles"
)

func TestAppPreferences_FreshInstall_HasTheDocumentedDefaults(t *testing.T) {
	empty := appPreferences{}
	if !empty.reopenTabs() {
		t.Error("a fresh install should reopen the last tabs")
	}
	if empty.themeVariant() != themeVariantSystem {
		t.Errorf("theme = %q, want the desktop's choice", empty.themeVariant())
	}
	if empty.defaultProtocol() != "" || empty.defaultPort() != 0 {
		t.Errorf("new connection defaults = %q/%d, want empty", empty.defaultProtocol(), empty.defaultPort())
	}
}

func TestApplySettings_StoresAndValidatesTheValues(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	prefs := appPreferences{prefs: app.Preferences()}

	err := applySettings(prefs, settingsValues{Theme: themeVariantLight, Protocol: string(profiles.RDP), Port: 3390, ReopenTabs: false})
	if err != nil {
		t.Fatalf("applySettings: %v", err)
	}
	if prefs.themeVariant() != themeVariantLight {
		t.Errorf("theme = %q, want light", prefs.themeVariant())
	}
	if prefs.defaultProtocol() != string(profiles.RDP) || prefs.defaultPort() != 3390 {
		t.Errorf("defaults = %q/%d, want RDP/3390", prefs.defaultProtocol(), prefs.defaultPort())
	}
	if prefs.reopenTabs() {
		t.Error("the reopen preference was not stored")
	}

	cases := map[string]settingsValues{
		"unknown protocol": {Theme: themeVariantSystem, Protocol: "FTP"},
		"port too high":    {Theme: themeVariantSystem, Protocol: string(profiles.SSH), Port: 70000},
		"port below range": {Theme: themeVariantSystem, Protocol: string(profiles.SSH), Port: -1},
	}
	for name, values := range cases {
		before := prefs.themeVariant()
		if err := applySettings(prefs, values); err == nil {
			t.Errorf("%s was accepted", name)
		}
		if prefs.themeVariant() != before {
			t.Errorf("%s changed the stored theme", name)
		}
	}
}

func TestParseSettingsPort_EmptyMeansTheProtocolDefault(t *testing.T) {
	if port, err := parseSettingsPort("  "); err != nil || port != 0 {
		t.Errorf("empty port = %d, %v; want the protocol default", port, err)
	}
	if port, err := parseSettingsPort("2222"); err != nil || port != 2222 {
		t.Errorf("port = %d, %v", port, err)
	}
	for _, bad := range []string{"ssh", "0", "65536", "-22"} {
		if _, err := parseSettingsPort(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestNewConnectionProfile_UsesTheStoredDefaults(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	app.Preferences().SetString(prefDefaultProtocol, string(profiles.RDP))
	app.Preferences().SetInt(prefDefaultPort, 3390)
	s := newTestScreenWithPreferences(t, &fakeStore{}, func(prefs fyne.Preferences) {
		prefs.SetString(prefDefaultProtocol, string(profiles.RDP))
		prefs.SetInt(prefDefaultPort, 3390)
	})

	profile := s.newConnectionProfile()
	if profile.Protocol != profiles.RDP || profile.Port != 3390 {
		t.Errorf("new connection = %+v, want RDP on the stored port", profile)
	}

	// An unknown stored protocol falls back to SSH rather than leaving the form empty.
	plain := newTestScreenWithPreferences(t, &fakeStore{}, func(prefs fyne.Preferences) {
		prefs.SetString(prefDefaultProtocol, "FTP")
	})
	if got := plain.newConnectionProfile(); got.Protocol != profiles.SSH || got.Port != 22 {
		t.Errorf("new connection = %+v, want SSH on its default port", got)
	}
}

func TestSoberThemeVariant_PinsTheColours(t *testing.T) {
	light := SoberThemeVariant(themeVariantLight)
	if got := light.Color(theme.ColorNameBackground, theme.VariantDark); got != lightBackground {
		t.Errorf("light theme background = %v, want the light surface whatever the desktop says", got)
	}
	dark := SoberThemeVariant(themeVariantDark)
	if got := dark.Color(theme.ColorNameBackground, theme.VariantLight); got != darkBackground {
		t.Errorf("dark theme background = %v, want the dark surface", got)
	}
	system := SoberThemeVariant(themeVariantSystem)
	if got := system.Color(theme.ColorNameBackground, theme.VariantLight); got != lightBackground {
		t.Errorf("system theme ignored the variant: %v", got)
	}
	if got := system.Color(theme.ColorNameBackground, theme.VariantDark); got != darkBackground {
		t.Errorf("system theme ignored the variant: %v", got)
	}
}

func TestSoberThemeForPreferences_ReadsTheStoredChoice(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	app.Preferences().SetString(prefThemeVariant, themeVariantLight)

	themed := SoberThemeForPreferences(app.Preferences())
	if got := themed.Color(theme.ColorNameBackground, theme.VariantDark); got != lightBackground {
		t.Errorf("startup theme = %v, want the stored light choice applied before the first paint", got)
	}
	if strings.TrimSpace(prefThemeVariant) == "" {
		t.Error("the preference key is empty")
	}
}
