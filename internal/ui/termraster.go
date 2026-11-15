package ui

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/mRemoteNG/mremoteng-go/internal/terminal"
)

// This file draws the terminal screen as one image.
//
// The first version used widget.TextGrid, which keeps a canvas.Rectangle and a
// canvas.Text for every cell: an 80x24 screen is 3840 objects that Fyne walks,
// measures and issues GL calls for on every frame, and profiling four live
// sessions put over 90% of the CPU there. A canvas.Raster is a single object
// and a single texture, so the cost of a frame is the cost of filling an image.
//
// Glyphs come from the theme's monospace font through golang.org/x/image/font.
// A rune the font has no glyph for (CJK, emoji, symbols) is not drawn into the
// image: it is overlaid as an ordinary canvas.Text on its cell, so Fyne's own
// font fallback still applies to it.

// frameCell is one cell of the visible screen after scrollback, selection and
// cursor inversion have been resolved. fg and bg are emulator colour ids.
type frameCell struct {
	r      rune
	fg, bg int32
}

// termFont is the parsed monospace font and the set of runes it can draw.
type termFont struct {
	name string
	font *opentype.Font
	mu   sync.Mutex
	buf  sfnt.Buffer
	has  map[rune]bool
}

var (
	termFontsMu sync.Mutex
	termFonts   = map[string]*termFont{}
)

// monospaceFont returns the parsed font for the resource, or nil when it cannot
// be parsed (every glyph then falls back to Fyne's text rendering).
func monospaceFont(res fyne.Resource) *termFont {
	if res == nil {
		return nil
	}
	termFontsMu.Lock()
	defer termFontsMu.Unlock()
	if known, ok := termFonts[res.Name()]; ok {
		return known
	}
	parsed, err := opentype.Parse(res.Content())
	var entry *termFont
	if err == nil {
		entry = &termFont{name: res.Name(), font: parsed, has: map[rune]bool{}}
	}
	termFonts[res.Name()] = entry
	return entry
}

// covers reports whether the font has a glyph for r.
func (f *termFont) covers(r rune) bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if known, ok := f.has[r]; ok {
		return known
	}
	index, err := f.font.GlyphIndex(&f.buf, r)
	covered := err == nil && index != 0
	f.has[r] = covered
	return covered
}

// glyph is a rasterised character: its mask and where it sits relative to the
// top-left corner of its cell.
type glyph struct {
	dr    image.Rectangle
	mask  image.Image
	maskp image.Point
}

// glyphSet caches the glyphs of one font at one pixel size.
type glyphSet struct {
	face    font.Face
	px      float64
	name    string
	ascent  int
	descent int
	glyphs  map[rune]glyph
}

func newGlyphSet(f *termFont, px float64) *glyphSet {
	if f == nil || px < 1 {
		return nil
	}
	face, err := opentype.NewFace(f.font, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil
	}
	metrics := face.Metrics()
	return &glyphSet{
		face: face, px: px, name: f.name,
		ascent: metrics.Ascent.Ceil(), descent: metrics.Descent.Ceil(),
		glyphs: map[rune]glyph{},
	}
}

func (g *glyphSet) glyph(r rune) glyph {
	if cached, ok := g.glyphs[r]; ok {
		return cached
	}
	dr, mask, maskp, _, ok := g.face.Glyph(fixed.P(0, g.ascent), r)
	cell := glyph{}
	if ok && mask != nil {
		// The face reuses the mask's storage on its next call, so keep a copy.
		own := image.NewAlpha(image.Rect(0, 0, dr.Dx(), dr.Dy()))
		draw.Draw(own, own.Bounds(), mask, maskp, draw.Src)
		cell = glyph{dr: dr, mask: own}
	}
	g.glyphs[r] = cell
	return cell
}

// monospace is the text style every terminal cell uses.
var monospace = fyne.TextStyle{Monospace: true}

// font returns the terminal's monospace font as the current theme resolves it.
func (t *termWidget) font() *termFont {
	return monospaceFont(t.zoom.Font(monospace))
}

// glyphsFor returns the glyph cache for a pixel size, rebuilding it when the
// size or the theme's font changed (a zoom step, a monitor with another scale).
func (t *termWidget) glyphsFor(f *termFont, px float64) *glyphSet {
	if t.glyphs != nil && t.glyphs.px == px && f != nil && t.glyphs.name == f.name {
		return t.glyphs
	}
	t.glyphs = newGlyphSet(f, px)
	return t.glyphs
}

// pixelScale is how many image pixels one device-independent unit covers.
func (t *termWidget) pixelScale(width int) float64 {
	if dp := t.raster.Size().Width; dp > 0 {
		return float64(width) / float64(dp)
	}
	return 1
}

// paintFrame is the raster generator: it draws the current frame into an image
// of w by h pixels.
func (t *termWidget) paintFrame(w, h int) image.Image {
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	img := t.imageOf(w, h)
	draw.Draw(img, img.Bounds(), image.NewUniform(rgba(paletteColor(terminal.DefaultBG))), image.Point{}, draw.Src)
	if t.frameCols == 0 || t.frameRows == 0 {
		return img
	}
	scale := t.pixelScale(w)
	cell := t.cellSize()
	pixels := cellPixels{width: float64(cell.Width) * scale, height: float64(cell.Height) * scale}
	t.paintBackgrounds(img, pixels)
	t.paintGlyphs(img, pixels, scale)
	return img
}

// cellPixels is the size of one cell in image pixels.
type cellPixels struct{ width, height float64 }

// rect is the pixel rectangle of a cell. Both edges are rounded from the same
// formula, so neighbouring cells meet without a gap or an overlap.
func (c cellPixels) rect(col, row int) image.Rectangle {
	return image.Rect(
		int(float64(col)*c.width+0.5), int(float64(row)*c.height+0.5),
		int(float64(col+1)*c.width+0.5), int(float64(row+1)*c.height+0.5),
	)
}

// imageOf reuses the image between frames: the texture upload copies it, so
// nothing keeps a reference once the generator returns.
func (t *termWidget) imageOf(w, h int) *image.RGBA {
	if t.img == nil || t.img.Bounds().Dx() != w || t.img.Bounds().Dy() != h {
		t.img = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	return t.img
}

// paintBackgrounds fills the cells that are not on the default background. The
// image is already cleared to the default, so those cells need nothing.
func (t *termWidget) paintBackgrounds(img *image.RGBA, pixels cellPixels) {
	fills := map[int32]*image.Uniform{}
	for y := 0; y < t.frameRows; y++ {
		for x := 0; x < t.frameCols; x++ {
			bg := t.frame[y*t.frameCols+x].bg
			if bg == terminal.DefaultBG {
				continue
			}
			fill, ok := fills[bg]
			if !ok {
				fill = image.NewUniform(rgba(paletteColor(bg)))
				fills[bg] = fill
			}
			draw.Draw(img, pixels.rect(x, y), fill, image.Point{}, draw.Src)
		}
	}
}

// paintGlyphs draws every character the font covers. It runs after the
// backgrounds so a glyph that leans into the next cell is not painted over.
func (t *termWidget) paintGlyphs(img *image.RGBA, pixels cellPixels, scale float64) {
	face := t.font()
	glyphs := t.glyphsFor(face, float64(t.textSize())*scale)
	if glyphs == nil {
		return
	}
	inks := map[int32]*image.Uniform{}
	for y := 0; y < t.frameRows; y++ {
		for x := 0; x < t.frameCols; x++ {
			c := t.frame[y*t.frameCols+x]
			if c.r <= ' ' || !face.covers(c.r) {
				continue
			}
			g := glyphs.glyph(c.r)
			if g.mask == nil {
				continue
			}
			ink, ok := inks[c.fg]
			if !ok {
				ink = image.NewUniform(rgba(paletteColor(c.fg)))
				inks[c.fg] = ink
			}
			origin := pixels.rect(x, y).Min
			draw.DrawMask(img, g.dr.Add(origin), ink, image.Point{}, g.mask, g.maskp, draw.Over)
		}
	}
}

func rgba(c color.Color) color.RGBA {
	return color.RGBAModel.Convert(c).(color.RGBA)
}

// syncOverlays keeps one canvas.Text on every cell whose rune the monospace
// font cannot draw, so Fyne's font fallback renders it.
func (t *termWidget) syncOverlays() {
	cell := t.cellSize()
	font := t.font()
	used := 0
	for y := 0; y < t.frameRows; y++ {
		for x := 0; x < t.frameCols; x++ {
			c := t.frame[y*t.frameCols+x]
			if c.r <= ' ' || font.covers(c.r) {
				continue
			}
			if used == len(t.overlays) {
				t.overlays = append(t.overlays, &canvas.Text{TextStyle: monospace})
			}
			text := t.overlays[used]
			text.Text = string(c.r)
			text.Color = paletteColor(c.fg)
			text.TextSize = t.textSize()
			text.Move(fyne.NewPos(float32(x)*cell.Width, float32(y)*cell.Height))
			text.Refresh()
			used++
		}
	}
	if t.renderer == nil {
		return
	}
	objects := make([]fyne.CanvasObject, 0, used+1)
	objects = append(objects, t.raster)
	for _, text := range t.overlays[:used] {
		objects = append(objects, text)
	}
	t.renderer.objects = objects
}
