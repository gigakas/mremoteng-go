---
timestamp: 2026-11-12T02:07:03Z
agent: mimocode
files:
  - internal/ui/theme.go
  - internal/ui/theme_test.go
---

Make secondary text readable instead of translucent grey

Reported after the hover hints landed: the messages were hard to read because the font looked too faint. The cause was not the hint feature but the theme: SoberTheme never defined ColorNameDisabled, so it inherited Fyne's default, a 35 percent white, and a widget.Label with LowImportance is painted with exactly that colour. Every secondary line in the window was affected, not only the new hints: the endpoint on the session bar, the Config panel notes, the import notice and input placeholders. SoberTheme now defines ColorNameDisabled and ColorNamePlaceHolder as solid muted colours, #9aa1a9 on the dark variant and #5f6670 on the light one. Measured with the WCAG formula the dark muted text goes from 2.7:1 to 6.5:1 against the background and the light one from 2.3:1 to 5.3:1, while the foreground stays at 13.7:1 and 14.5:1, so secondary text is legible without competing with primary text. Added TestSoberTheme_MutedText_StaysReadable, which computes the ratios and fails below 4.5:1 or if the muted colour stops being weaker than the foreground, and checks both variants share the placeholder colour. Validated with ./scripts/check.sh.
