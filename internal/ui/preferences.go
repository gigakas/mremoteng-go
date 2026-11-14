package ui

import "fyne.io/fyne/v2"

// Preference keys. They live in one place so the settings dialog, the startup path
// and the tests cannot drift apart on a string.
const (
	prefReopenTabs      = "session.reopen"
	prefDefaultProtocol = "new.protocol"
	prefDefaultPort     = "new.port"
	prefThemeVariant    = "ui.theme"
)

// preference defaults: a fresh install reopens the last tabs, starts a new
// connection with SSH, and follows the desktop's light or dark choice.
const (
	defaultReopenTabs      = true
	defaultThemeVariant    = "system"
	reopenTabsDescription  = "reopen the tabs that were open when the window closed"
	themeVariantSystem     = "system"
	themeVariantLight      = "light"
	themeVariantDark       = "dark"
	defaultProtocolSetting = ""
)

// appPreferences is the typed view of the Fyne preference store the window needs.
// A nil store answers with the defaults, which is what a headless test wants.
type appPreferences struct {
	prefs fyne.Preferences
}

// reopenTabs reports whether the next start should reopen the stored tabs.
func (p appPreferences) reopenTabs() bool {
	if p.prefs == nil {
		return defaultReopenTabs
	}
	return p.prefs.BoolWithFallback(prefReopenTabs, defaultReopenTabs)
}

// setReopenTabs stores the choice.
func (p appPreferences) setReopenTabs(on bool) {
	if p.prefs == nil {
		return
	}
	p.prefs.SetBool(prefReopenTabs, on)
}

// defaultProtocol is the protocol a new connection starts with.
func (p appPreferences) defaultProtocol() string {
	if p.prefs == nil {
		return defaultProtocolSetting
	}
	return p.prefs.StringWithFallback(prefDefaultProtocol, defaultProtocolSetting)
}

// defaultPort is the port a new connection starts with, or zero to use the
// protocol's own default.
func (p appPreferences) defaultPort() int {
	if p.prefs == nil {
		return 0
	}
	return p.prefs.IntWithFallback(prefDefaultPort, 0)
}

// setNewConnectionDefault stores the protocol and port a new connection starts with.
func (p appPreferences) setNewConnectionDefault(protocol string, port int) {
	if p.prefs == nil {
		return
	}
	p.prefs.SetString(prefDefaultProtocol, protocol)
	p.prefs.SetInt(prefDefaultPort, port)
}

// themeVariant is the stored colour choice: "system", "light" or "dark".
func (p appPreferences) themeVariant() string {
	if p.prefs == nil {
		return defaultThemeVariant
	}
	variant := p.prefs.StringWithFallback(prefThemeVariant, defaultThemeVariant)
	if variant != themeVariantLight && variant != themeVariantDark {
		return themeVariantSystem
	}
	return variant
}

// setThemeVariant stores the colour choice.
func (p appPreferences) setThemeVariant(variant string) {
	if p.prefs == nil {
		return
	}
	p.prefs.SetString(prefThemeVariant, variant)
}
