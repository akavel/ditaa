package dasher

import (
	"math"

	"golang.org/x/image/math/fixed"
)

type Add1er interface {
	Start(p fixed.Point26_6)
	Add1(p fixed.Point26_6)
}

type Dasher struct {
	Length fixed.Int26_6 // Length of a dash segment
	P0     fixed.Point26_6
	A      Add1er

	carry fixed.Int26_6
	gap   bool // currently drawing the gap or dash
}

func (d *Dasher) Start(p fixed.Point26_6) {
	// fmt.Printf("\n%v\n", p)
	d.P0 = p
	d.carry = 0
	d.gap = false
	d.A.Start(p)
}
func (d *Dasher) Add1(p1 fixed.Point26_6) {
	// fmt.Printf("%v\n", p1)
	vec01 := p1.Sub(d.P0)
	len01 := pLen(vec01)
	carry := d.carry
	for i := 1; ; i++ {
		advance := fixed.Int26_6(i)*d.Length - carry
		if advance > len01 { // FIXME(akavel): > or >= ?
			d.carry = len01 - (advance - d.Length)
			break
		}

		numerator := int64(i)*int64(d.Length) - int64(carry)
		denominator := int64(len01)
		p1 := fixed.Point26_6{
			X: d.P0.X + scale(vec01.X, numerator, denominator),
			Y: d.P0.Y + scale(vec01.Y, numerator, denominator),
		}
		if d.gap {
			d.A.Start(p1)
		} else {
			d.A.Add1(p1)
		}
		d.gap = !d.gap
	}
	// draw final dash fragment to p1 if required
	if !d.gap {
		d.A.Add1(p1)
	}
	d.P0 = p1
}

func scale(x fixed.Int26_6, numerator, denominator int64) fixed.Int26_6 {
	return fixed.Int26_6((int64(x) * numerator) / denominator)
}

type DeBezierizer struct {
	P0 fixed.Point26_6
	A  Add1er
}

// var depth = 0

func (d *DeBezierizer) Start(p0 fixed.Point26_6) {
	d.A.Start(p0)
	d.P0 = p0
}
func (d *DeBezierizer) Add1(p1 fixed.Point26_6) {
	d.A.Add1(p1)
	d.P0 = p1
}
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
	if dot < 0 {
		dot = -dot
	}
	// fmt.Printf("%s dot=%v\n", strings.Repeat(" ", depth), dot)
	const minDot = fixed.Int52_12((1 << 12) / 32)
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
