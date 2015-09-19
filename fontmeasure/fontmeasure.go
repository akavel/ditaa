package fontmeasure

import (
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// go:generate go-bindata -pkg rsrc -o rsrc/font.ttf.go -ignore \.(java|class|txt)$ -prefix orig-java/src/org/stathissideris/ascii2image/graphics orig-java/src/org/stathissideris/ascii2image/graphics
// go:generate go-assets-builder -p rsrc -o rsrc/font.ttf.go -s orig-java/src/org/stathissideris/ascii2image/graphics orig-java/src/org/stathissideris/ascii2image/graphics/font.ttf
//go:generate go run tools/embd.go -o embd/font.ttf.go -p embd orig-java/src/org/stathissideris/ascii2image/graphics/font.ttf

type Font struct {
	Font *truetype.Font
	DPI  float64
	Size float64
}

func (f Font) scale() fixed.Int26_6 {
	// See: freetype.Context#recalc()
	// at: https://github.com/golang/freetype/blob/41fa49aa5b23cc7c4082c9aaaf2da41e195602d9/freetype.go#L263
	// also a comment from the same file:
	// "scale is the number of 26.6 fixed point units in 1 em"
	// (where 26.6 means 26 bits integer and 6 fractional)
	// also from docs:
	// "If the device space involves pixels, 64 units
	// per pixel is recommended, since that is what
	// the bytecode hinter uses [...]".
	return fixed.Int26_6(f.Size * f.DPI * (64.0 / 72.0))
}

func (f Font) Baseline() int {
	return int(f.Font.Bounds(f.scale()).Max.Y >> 6)
	// or use f.Font.VMetric() for some glyph?
}

func (f Font) Advance() int {
	b := f.Font.Bounds(f.scale())
	return int((b.Max.Y - b.Min.Y) >> 6) // or -1 in inner parens?
	// or use f.Font.VMetric() for some glyph?
}

func (f Font) Ascent() int {
	// ok or not?
	b := f.Font.Bounds(f.scale())
	return int(b.Max.Y >> 6)
}

func (f Font) WidthFor(s string) int {
	face := truetype.NewFace(f.Font, &truetype.Options{
		Size: f.Size,
		DPI:  f.DPI,
		// TODO(akavel): Hinting: font.HintingFull, // ?
	})
	drawer := font.Drawer{Face: face}
	advance := drawer.MeasureString(s)
	return int(advance >> 6)
}

func (f Font) ZHeight() int {
	z := f.Font.Index('Z')
	glyph := truetype.GlyphBuf{}
	// TODO(akavel): font.HintingFull ?
	glyph.Load(f.Font, f.scale(), z, font.HintingNone)
	// TODO(akavel): or, use MeasureString("Z")?
	return int((glyph.Bounds.Max.Y - glyph.Bounds.Min.Y) >> 6)
}

func prepFont(font *truetype.Font) Font {
	// Note: that's the default value used in the truetype package
	const dpi = 72
	return Font{font, dpi, 12.0}
}

func GetFontForHeight(font *truetype.Font, h int) *Font {
	measure := prepFont(font)
	// TODO(akavel): original code used 'ascent' (reporting that it's distance between the baseline and the tallest character); are we implementing it ok?
	fontH := measure.Ascent()
	direction := 1.0
	if fontH > h {
		direction = -1.0
	}
	measure.Size += direction
	for measure.Size > 0 {
		fontH = measure.Ascent()
		if direction > 0 {
			if fontH > h {
				measure.Size -= 0.5
				return &measure
			}
		} else {
			if fontH < h {
				return &measure
			}
		}
		measure.Size += 0.5 * direction
	}
	return nil // TODO(akavel): does it make sense? maybe panic?
}

func prepCtx(font *Font) *freetype.Context {
	ctx := freetype.NewContext()
	ctx.SetDPI(font.DPI)
	ctx.SetFont(font.Font)
	ctx.SetFontSize(font.Size)
	return ctx
}

func GetFontForWidth(font *truetype.Font, w int, s string) *Font {
	// fmt.Println("MCDBG GetFontForWidth w=", w, "s=", s)
	measure := prepFont(font)
	fontW := measure.WidthFor(s)
	direction := 1.0
	if fontW > w {
		direction = -1.0
	}
	measure.Size += direction
	for measure.Size > 0 {
		fontW = measure.WidthFor(s)
		if direction > 0 {
			if fontW > w {
				measure.Size -= 1
				return &measure
			}
		} else {
			if fontW < w {
				return &measure
			}
		}
		measure.Size += direction
	}
	return nil
}
