package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"os"

	"code.google.com/p/wozniakk-freetype-go/freetype"
	"code.google.com/p/wozniakk-freetype-go/freetype/truetype"
)

var (
	fontArg = flag.String("font", "font.ttf", "file with TrueType font")
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func hline(img draw.Image, x0, y0, x1 int, rgb int32) {
	draw.Draw(img, image.Rect(x0, y0, x1, y0+1), image.NewUniform(color.NRGBA{uint8(rgb >> 16), uint8(rgb >> 8), uint8(rgb), 255}), image.Pt(0, 0), draw.Src)
}

func run() error {
	flag.Parse()
	if *fontArg == "" {
		flag.Usage()
		os.Exit(1)
	}

	fontfile, err := os.Open(*fontArg)
	if err != nil {
		return err
	}
	defer fontfile.Close()

	w, err := os.OpenFile("test-fontmeasure.png", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer w.Close()

	fontdata, err := ioutil.ReadAll(fontfile)
	font, err := freetype.ParseFont(fontdata)
	if err != nil {
		return err
	}

	img := image.NewNRGBA(image.Rect(0, 0, 320, 200))
	draw.Draw(img, image.Rect(0, 0, 320, 200), image.White, image.Pt(0, 0), draw.Src)

	// fontmetrics := Font{DPI: 72, Size: 12}
	fontmetrics := Font{DPI: 72, Size: 20}
	bounds := font.Bounds(fontmetrics.scale())
	i := font.Index('f')
	vmetric := font.VMetric(fontmetrics.scale(), i)
	fmt.Printf("%#v\n%#v\n", bounds, vmetric)

	origin := freetype.Pt(10, 100)
	intorigin := image.Pt(int(origin.X>>8), int(origin.Y>>8))
	hline(img, intorigin.X, intorigin.Y, 320, 0x8080ff)
	_ = bounds
	// hline(img, 0, intorigin.Y-int(vmetric.AdvanceHeight>>6), 320, 0xffff80)
	// hline(img, 0, intorigin.Y-int(vmetric.TopSideBearing>>6), 320, 0xff8080)
	hline(img, 0, intorigin.Y-int(bounds.YMax>>6), 320, 0x80ff80)
	hline(img, 0, intorigin.Y-int(bounds.YMin>>6), 320, 0xffff80)

	// glyph := truetype.NewGlyphBuf()
	// // i = font.Index('Z')
	// // glyph.Load(font, fontmetrics.scale(), zIndex, nil)
	// glyph.Load(font, fontmetrics.scale(), i, nil)
	// // glyph.Load(font, fontmetrics.scale(), font.Index('g'), nil)
	// hline(img, 0, intorigin.Y-int(glyph.B.YMax>>6), 320, 0x80ff80)
	// hline(img, 0, intorigin.Y-int(glyph.B.YMin>>6), 320, 0xffff80)

	ctx := freetype.NewContext()
	ctx.SetFont(font)
	// ctx.SetDPI(10)
	ctx.SetDPI(fontmetrics.DPI)
	ctx.SetFontSize(fontmetrics.Size)
	ctx.SetSrc(image.Black)
	ctx.SetDst(img)
	ctx.SetClip(image.Rect(0, 0, 320, 200))
	ctx.DrawString("foogarŻÓŹX", origin)

	err = png.Encode(w, img)
	if err != nil {
		return err
	}
	return nil
}

type Font struct {
	Font truetype.Font
	DPI  float64
	Size float64
}

func (f Font) scale() int32 {
	// See: freetype.Context#recalc()
	// at: https://code.google.com/p/freetype-go/source/browse/freetype/freetype.go#242
	// also a comment from the same file:
	// "scale is the number of 26.6 fixed point units in 1 em"
	// (where 26.6 means 26 bits integer and 6 fractional)
	// also from docs:
	// "If the device space involves pixels, 64 units
	// per pixel is recommended, since that is what
	// the bytecode hinter uses [...]".
	return int32(f.Size * f.DPI * (64.0 / 72.0))
}
