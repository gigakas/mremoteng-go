package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/mRemoteNG/mremoteng-go/internal/terminal"
)

// rasterTerm returns a terminal sized like a window of the given dp size,
// ready to paint.
func rasterTerm(t *testing.T, cols, rows int, size fyne.Size) *termWidget {
	t.Helper()
	app := test.NewApp()
	t.Cleanup(app.Quit)
	term := newTermWidget(cols, rows, func([]byte) {}, nil)
	term.CreateRenderer()
	term.raster.Resize(size)
	return term
}

func paint(term *termWidget, scale float64) *image.RGBA {
	size := term.raster.Size()
	w, h := int(float64(size.Width)*scale), int(float64(size.Height)*scale)
	return term.paintFrame(w, h).(*image.RGBA)
}

// cellRegion is the pixel rectangle of a cell for an image drawn at scale.
func cellRegion(term *termWidget, scale float64, col, row int) image.Rectangle {
	cell := term.cellSize()
	return cellPixels{width: float64(cell.Width) * scale, height: float64(cell.Height) * scale}.rect(col, row)
}

func differsFrom(img *image.RGBA, region image.Rectangle, background color.RGBA) int {
	count := 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			if img.RGBAAt(x, y) != background {
				count++
			}
		}
	}
	return count
}

func TestPaintFrame_TextOnDefaultBackground_InksOnlyItsOwnCell(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	term.write([]byte("W"))
	img := paint(term, 1)
	background := rgba(paletteColor(terminal.DefaultBG))

	if got := differsFrom(img, cellRegion(term, 1, 0, 0), background); got < 10 {
		t.Errorf("cell (0,0) has %d inked pixels, want the glyph drawn", got)
	}
	if got := differsFrom(img, cellRegion(term, 1, 8, 3), background); got != 0 {
		t.Errorf("an empty cell has %d inked pixels, want none", got)
	}
}

func TestPaintFrame_ColouredBackground_FillsTheWholeCell(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	term.write([]byte("\x1b[41m \x1b[0mx"))
	img := paint(term, 1)
	red := rgba(paletteColor(1))

	region := cellRegion(term, 1, 0, 0)
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			if img.RGBAAt(x, y) != red {
				t.Fatalf("pixel (%d,%d) = %v, want the cell filled with %v", x, y, img.RGBAAt(x, y), red)
			}
		}
	}
}

func TestPaintFrame_HigherScale_ProducesAMatchingImage(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	term.write([]byte("W"))

	img := paint(term, 2)

	if img.Bounds().Dx() != 800 || img.Bounds().Dy() != 400 {
		t.Fatalf("image = %v, want 800x400 for a 400x200 widget at scale 2", img.Bounds())
	}
	background := rgba(paletteColor(terminal.DefaultBG))
	small := differsFrom(paint(term, 1), cellRegion(term, 1, 0, 0), background)
	large := differsFrom(img, cellRegion(term, 2, 0, 0), background)
	if large < small*2 {
		t.Errorf("glyph covers %d pixels at scale 2 and %d at scale 1, want it to grow with the scale", large, small)
	}
}

func TestPaintFrame_RuneMissingFromTheFont_IsLeftToFyneText(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	term.write([]byte("a漢"))
	img := paint(term, 1)
	background := rgba(paletteColor(terminal.DefaultBG))

	if len(term.overlays) == 0 || term.overlays[0].Text != "漢" {
		t.Fatalf("overlays = %d, want one text for the rune the font cannot draw", len(term.overlays))
	}
	if got := differsFrom(img, cellRegion(term, 1, 1, 0), background); got != 0 {
		t.Errorf("the raster inked %d pixels of a cell that belongs to a Fyne overlay", got)
	}
	if term.renderer.objects[len(term.renderer.objects)-1] != term.overlays[0] {
		t.Error("the overlay is not part of the renderer's objects")
	}
}

func TestPaintFrame_OnlyCoveredRunes_NeedNoOverlay(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	term.write([]byte("plain ascii and box ─│┌"))

	if len(term.overlays) != 0 && term.renderer.objects[len(term.renderer.objects)-1] != term.raster {
		t.Errorf("overlays = %d for text the font covers, want none in use", len(term.overlays))
	}
}

func TestPaintFrame_ZeroSize_ReturnsAUsableImage(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	if img := term.paintFrame(0, 0); img == nil {
		t.Fatal("paintFrame returned nil for an empty area")
	}
}

// TestPaintFrame_SameGlyphTwice_DrawsIdenticalPixels guards the glyph cache: the
// font face reuses the storage of the mask it returns, so a cache that kept the
// mask as is would corrupt every glyph drawn after another one.
func TestPaintFrame_SameGlyphTwice_DrawsIdenticalPixels(t *testing.T) {
	term := rasterTerm(t, 20, 5, fyne.NewSize(400, 200))
	term.write([]byte("WiMiW"))
	img := paint(term, 1)

	window := func(col int) []color.RGBA {
		origin := cellRegion(term, 1, col, 0).Min
		var pixels []color.RGBA
		for y := 0; y < 10; y++ {
			for x := 0; x < 6; x++ {
				pixels = append(pixels, img.RGBAAt(origin.X+x, origin.Y+y))
			}
		}
		return pixels
	}
	first, last := window(0), window(4)
	for i := range first {
		if first[i] != last[i] {
			t.Fatalf("the two W cells differ at pixel %d: %v against %v", i, first[i], last[i])
		}
	}
}
