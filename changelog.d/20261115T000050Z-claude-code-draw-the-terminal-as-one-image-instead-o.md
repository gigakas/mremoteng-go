---
timestamp: 2026-11-15T00:00:50Z
agent: claude-code
files:
  - internal/ui/terminal.go
  - internal/ui/termraster.go
  - internal/ui/termraster_test.go
  - internal/ui/terminal_test.go
  - internal/ui/zoom_test.go
  - go.mod
  - go.sum
---

Draw the terminal as one image instead of one object per cell

widget.TextGrid keeps a Rectangle and a Text for every cell, 3840 objects on an 80x24 screen, and Fyne walks, measures and issues GL calls for each on every frame. The terminal now renders its frame into a single canvas.Raster, with glyphs rasterised from the theme's monospace font through golang.org/x/image/font and cached per rune and pixel size (the face reuses its mask storage, so the cache keeps copies). Runes the font has no glyph for, such as CJK and emoji, stay ordinary canvas.Text overlays on their cell so Fyne's font fallback still applies. golang.org/x/image was already in the module graph through Fyne and is now a direct dependency. Measured with four real SSH sessions: full-screen redraws at 10 Hz went from 10% to 4% of a core, a 100 Hz scrolling flood from 85% to 36%, and 300 Hz from 106% to 63%. Text is slightly lighter than before because glyphs are antialiased in greyscale; the screenshots otherwise match.
