---
timestamp: 2026-07-23T22:33:18Z
agent: claude-code
files:
  - auditory/phase3-stage6-20260724-claude-code.md
  - blueprint/phase-3-ui.md
  - cmd/mremoteng/main.go
  - internal/ui/optionsdialog.go
  - internal/ui/theme.go
  - internal/ui/theme_test.go
---

Phase 3 stage 3.6: theming

Add internal/ui/theme.go: ApplyTheme(a, name) maps settings.Settings.Theme (system/light/dark) to a Fyne theme. system uses theme.DefaultTheme() unchanged (tracks OS preference); light/dark wrap it in a small forcedVariantTheme decorator that overrides Color to substitute a fixed variant regardless of what Fyne's renderer asks for -- chosen over the deprecated theme.DarkTheme()/LightTheme() functions, whose own doc comments point at exactly this custom-theme pattern instead. optionsdialog.go's theme choice list is now the shared ui.ThemeChoices. Wired in cmd/mremoteng/main.go at startup and inside the Options dialog's save callback so a theme change applies immediately.
