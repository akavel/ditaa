// +build none

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"

	"github.com/golang/freetype/raster"
	"golang.org/x/image/math/fixed"
)

func main() {
	port := flag.String("p", ":8888", "port")
	flag.Parse()
	fmt.Println("listening on", *port)

	http.HandleFunc("/", serveImg)
	log.Fatal(http.ListenAndServe(*port, nil))
}

func serveImg(wr http.ResponseWriter, _ *http.Request) {
	wr.Header().Set("content-type", "image/png")
	w, h := 101, 101
	path := raster.Path{}
	p := DeBezierizer{Line: Liner(&path)}
	// p := &path // reference implementation
	p.Start(fixed.P(1, 1))
	p.Add1(fixed.P(100, 10))
	p.Add2(fixed.P(100, 50), fixed.P(25, 100))
	// p.Start(fixed.P(25, 100))
	p.Add2(fixed.P(13, 0), fixed.P(1, 100))
	r := raster.NewRasterizer(w, h)
	raster.Stroke(r, path, 2<<6, nil, nil)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	painter := raster.NewRGBAPainter(img)
	painter.SetColor(color.NRGBA{0, 0, 255, 255})
	r.Rasterize(painter)
	err := png.Encode(wr, img)
	if err != nil {
		panic(err)
	}
}

func Liner(a raster.Adder) func(p0, p1 fixed.Point26_6) {
	started := false
	start := fixed.Point26_6{}
	return func(p0, p1 fixed.Point26_6) {
		if !started || start != p0 {
			a.Start(p0)
			started = true
		}
		a.Add1(p1)
		start = p1
	}
}

type DeBezierizer struct {
	P0   fixed.Point26_6
	Line func(p0, p1 fixed.Point26_6)
}

// var depth = 0

func (d *DeBezierizer) Start(p0 fixed.Point26_6) { d.P0 = p0 }
func (d *DeBezierizer) Add1(p1 fixed.Point26_6)  { d.Line(d.P0, p1); d.P0 = p1 }
func (d *DeBezierizer) Add2(p1, p2 fixed.Point26_6) {
	// ind := strings.Repeat(" ", depth)
	// fmt.Printf("%sAdd2(%v, %v, %v)\n", ind, d.P0, p1, p2)
	if !curvy(d.P0, p1, p2) {
		// fmt.Printf("%s Add1(%v, %v)\n", ind, d.P0, p2)
		d.Add1(p2)
		return
	}
	ps := split(d.P0, p1, p2)
	// depth++
	d.Add2(ps[1], ps[2])
	d.Add2(ps[3], ps[4])
	// depth--
	// if depth == 0 {
	// 	fmt.Println()
	// }
}

func curvy(p0, p1, p2 fixed.Point26_6) bool {
	// FIXME(akavel): make sure if this func makes any sense; improve if needed
	vec01 := p1.Sub(p0)
	vec12 := p2.Sub(p1)
	d01 := pLen(vec01)
	d12 := pLen(vec12)
	const minLen = fixed.Int26_6(1 << 6)
	if d01 < minLen || d12 < minLen {
		return false
	}
	n01 := pNorm(vec01, 1<<6)
	n12 := pNorm(vec12, 1<<6)
	dot := pDot(n01, n12)
	const minDot = fixed.Int52_12((1 << 12) / 8)
	// fmt.Printf("%s dot=%v\n", strings.Repeat(" ", depth), dot)
	if dot < minDot {
		return false
	}
	return true
}

func split(p0, p1, p2 fixed.Point26_6) [5]fixed.Point26_6 {
	// based on: http://stackoverflow.com/a/8405756/98528
	x0, y0 := p0.X, p0.Y
	x1, y1 := p1.X, p1.Y
	x2, y2 := p2.X, p2.Y

	x01 := (x1-x0)/2 + x0
	y01 := (y1-y0)/2 + y0

	x12 := (x2-x1)/2 + x1
	y12 := (y2-y1)/2 + y1

	x012 := (x12-x01)/2 + x01
	y012 := (y12-y01)/2 + y01

	return [5]fixed.Point26_6{
		{x0, y0},
		{x01, y01},
		{x012, y012},
		{x12, y12},
		{x2, y2},
	}
}

// pLen returns the length of the vector p.
func pLen(p fixed.Point26_6) fixed.Int26_6 {
	// TODO(nigeltao): use fixed point math.
	x := float64(p.X)
	y := float64(p.Y)
	return fixed.Int26_6(math.Sqrt(x*x + y*y))
}

// pNorm returns the vector p normalized to the given length, or zero if p is
// degenerate.
func pNorm(p fixed.Point26_6, length fixed.Int26_6) fixed.Point26_6 {
	d := pLen(p)
	if d == 0 {
		return fixed.Point26_6{}
	}
	s, t := int64(length), int64(d)
	x := int64(p.X) * s / t
	y := int64(p.Y) * s / t
	return fixed.Point26_6{fixed.Int26_6(x), fixed.Int26_6(y)}
}

// midpoint returns the midpoint of two Points.
func midpoint(a, b fixed.Point26_6) fixed.Point26_6 {
	return fixed.Point26_6{(a.X + b.X) / 2, (a.Y + b.Y) / 2}
}

// pDot returns the dot product p·q.
func pDot(p fixed.Point26_6, q fixed.Point26_6) fixed.Int52_12 {
	px, py := int64(p.X), int64(p.Y)
	qx, qy := int64(q.X), int64(q.Y)
	return fixed.Int52_12(px*qx + py*qy)
}
