package graphical

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/math/fixed"

	"./dasher"
	"github.com/golang/freetype/raster"
)

const (
	STROKE_WIDTH float64 = 1
	MAGIC_K      float64 = 0.5522847498
)

type Color struct {
	R uint8 `xml:"r,attr"`
	G uint8 `xml:"g,attr"`
	B uint8 `xml:"b,attr"`
	A uint8 `xml:"a,attr"`
}

func (c Color) RGBA() color.RGBA {
	return color.RGBA{c.R, c.G, c.B, c.A}
}

var WHITE = Color{255, 255, 255, 255}

type PointType int

const (
	POINT_NORMAL PointType = iota
	POINT_ROUND
)

type Point struct {
	X      float64   `xml:"x,attr"`
	Y      float64   `xml:"y,attr"`
	Locked bool      `xml:"locked,attr"`
	Type   PointType `xml:"type,attr"`
}

func (p1 Point) NorthOf(p2 Point) bool { return p1.Y < p2.Y }
func (p1 Point) SouthOf(p2 Point) bool { return p1.Y > p2.Y }
func (p1 Point) WestOf(p2 Point) bool  { return p1.X < p2.X }
func (p1 Point) EastOf(p2 Point) bool  { return p1.X > p2.X }

func P(p Point) fixed.Point26_6 {
	//TODO: handle fractional part too, but probably not needed
	return fixed.P(int(p.X), int(p.Y))
}

func ftofix(f float64) fixed.Int26_6 {
	//TODO: verify this is OK
	a := math.Trunc(f)
	b := math.Ldexp(math.Abs(f-a), 6)
	return fixed.Int26_6(a)<<6 + fixed.Int26_6(b)
}

func Stroke(img *image.RGBA, path raster.Path, color color.RGBA) {
	stroke(img, path, color, nil)
}

func stroke(img *image.RGBA, path raster.Path, color color.RGBA, cr raster.Capper) {
	g := raster.NewRasterizer(img.Rect.Max.X+1, img.Rect.Max.Y+1) //TODO: +1 or not?
	raster.Stroke(g, path, ftofix(STROKE_WIDTH), cr, nil)
	painter := raster.NewRGBAPainter(img)
	painter.SetColor(color)
	g.Rasterize(painter)
}

func Dash(img *image.RGBA, path raster.Path, color color.RGBA) {
	p := func(x, y fixed.Int26_6) fixed.Point26_6 {
		return fixed.Point26_6{x, y}
	}
	dashed := raster.Path{}
	dasher := dasher.DeBezierizer{A: &dasher.Dasher{
		Length: fixed.I(5), // FIXME(akavel): make this configurable, or auto-detect
		A:      &dashed,
	}}
	for len(path) > 0 {
		switch path[0] {
		case 0:
			dasher.Start(p(path[1], path[2]))
			path = path[4:]
		case 1:
			dasher.Add1(p(path[1], path[2]))
			path = path[4:]
		case 2:
			dasher.Add2(p(path[1], path[2]), p(path[3], path[4]))
			path = path[6:]
		case 3:
			panic("Dash: cubic paths not implemented")
		default:
			panic("Dash: unknown code of path segment")
		}
	}
	stroke(img, dashed, color, raster.ButtCapper)
}

func Fill(img *image.RGBA, path raster.Path, color color.RGBA) {
	g := raster.NewRasterizer(img.Rect.Max.X+1, img.Rect.Max.Y+1) //TODO: +1 or not?
	g.AddPath(path)
	painter := raster.NewRGBAPainter(img)
	painter.SetColor(color)
	g.Rasterize(painter)
}

func Circle(x, y, r float64) raster.Path {
	//panic(fmt.Sprint(x, y, r))
	P := func(x, y float64) fixed.Point26_6 {
		return fixed.Point26_6{ftofix(x), ftofix(y)}
	}
	p1 := P(x+r, y)
	p2 := P(x, y+r)
	p3 := P(x-r, y)
	p4 := P(x, y-r)
	kr := MAGIC_K * r
	path := raster.Path{}
	// see: http://hansmuller-flex.blogspot.com/2011/04/approximating-circular-arc-with-cubic.html
	//  or: http://www.whizkidtech.redprince.net/bezier/circle/
	// etc. -- google "drawing circle with cubic curves"
	path.Start(p1)
	path.Add3(P(x+r, y+kr), P(x+kr, y+r), p2)
	path.Add3(P(x-kr, y+r), P(x-r, y+kr), p3)
	path.Add3(P(x-r, y-kr), P(x-kr, y-r), p4)
	path.Add3(P(x+kr, y-r), P(x+r, y-kr), p1)
	return path
}
