package main

import (
	"fmt"

	"github.com/akavel/polyclip-go"

	"./graphical"
)

type edgeType int

const (
	edgeHorizontal edgeType = iota
	edgeVertical
	edgeSloped
)

type edge struct {
	start, end *graphical.Point
	owner      *graphical.Shape
}

func (e edge) String() string {
	return fmt.Sprintf("(%v, %v) -> (%v, %v)", e.start.X, e.start.Y, e.end.X, e.end.Y)
}

func separateCommonEdges(gg graphical.Grid, shapes []graphical.Shape) []graphical.Shape {
	offset := gg.MinimumOfCellDimensions() / 5
	edges := []edge{}

	// get all edges
	for i := range shapes {
		s := &shapes[i]
		n := len(s.Points)
		if n == 1 {
			continue
		}
		for j := 0; j < n-1; j++ {
			edges = append(edges, edge{
				start: &s.Points[j],
				end:   &s.Points[j+1],
				owner: s,
			})
		}
		if s.Closed {
			edges = append(edges, edge{
				start: &s.Points[n-1],
				end:   &s.Points[0],
				owner: s,
			})
		}
	}

	// group edges into pairs of touching edges
	pairs := [][2]edge{}

	// all-against-all touching test for the edges
	startIndex := 1 //skip some to avoid duplicate comparisons and self-to-self comparisons

	for _, edge1 := range edges {
		for _, edge2 := range edges[startIndex:] {
			if edge1.TouchesWith(edge2) {
				pairs = append(pairs, [2]edge{edge1, edge2})
				if DEBUG {
					fmt.Println(edge1, "touches with", edge2)
				}
			}
		}
		startIndex++
	}

	moved := []edge{}

	// move equivalent edges inwards
	for _, p := range pairs {
	edges:
		for _, e := range p[:] {
			for _, m := range moved {
				if m.Equals(e) {
					continue edges
				}
			}
			// e not_in moved
			e.MoveInwardsBy(offset)
			moved = append(moved, e)
		}
	}

	return shapes
}

func (e1 edge) TouchesWith(e2 edge) bool {
	switch {
	case e1.Equals(e2):
		return true

	case e1.Horizontal() && e2.Vertical():
		return false
	case e1.Vertical() && e2.Horizontal():
		return false

	case e1.DistanceFromOrigin() != e2.DistanceFromOrigin():
		return false
	}

	//covering this corner case (should produce false):
	//      ---------
	//              ---------

	first, second := e1, e2
	if first.Vertical() {
		first.ChangeAxis()
		second.ChangeAxis()
	}
	first.FixDirection()
	second.FixDirection()
	if first.start.X > second.start.X {
		first, second = second, first
	}
	if *first.end == *second.start {
		return false
	}

	// case 1:
	// ----------
	//      -----------

	// case 2:
	//         ------
	// -----------------

	switch {
	case e2.PointWithin(e1.start) || e2.PointWithin(e1.end):
		return true
	case e1.PointWithin(e2.start) || e1.PointWithin(e2.end):
		return true
	}

	return false
}

func (e1 edge) Equals(e2 edge) bool {
	switch {
	case *e1.start == *e2.start && *e1.end == *e2.end:
		return true
	case *e1.start == *e2.end && *e1.end == *e2.start:
		return true
	}
	return false
}

func (e edge) Horizontal() bool { return e.start.Y == e.end.Y }
func (e edge) Vertical() bool   { return e.start.X == e.end.X }

func (e edge) DistanceFromOrigin() float64 {
	switch e.Type() {
	case edgeSloped:
		panic("Cannot calculate distance of sloped edge from origin")
	case edgeHorizontal:
		return e.start.Y
	default: // edgeVertical
		return e.start.X
	}
}

func (e *edge) ChangeAxis() {
	tmp := e.start
	e.start = &graphical.Point{X: e.end.Y, Y: e.end.X}
	e.end = &graphical.Point{X: tmp.Y, Y: tmp.X}
}

func (e *edge) FixDirection() {
	switch {
	case e.Horizontal():
		if e.start.X > e.end.X {
			e.FlipDirection()
		}
	case e.Vertical():
		if e.start.Y > e.end.Y {
			e.FlipDirection()
		}
	default:
		panic("Cannot fix direction of sloped egde")
	}
}

func (e edge) PointWithin(p *graphical.Point) bool {
	switch {
	case e.Horizontal():
		return (p.X >= e.start.X && p.X <= e.end.X) ||
			(p.X >= e.end.X && p.X <= e.start.X)
	case e.Vertical():
		return (p.Y >= e.start.Y && p.Y <= e.end.Y) ||
			(p.Y >= e.end.Y && p.Y <= e.start.Y)
	default:
		panic("Cannot calculate is ShapePoint is within sloped edge")
	}
}

func (e edge) Type() edgeType {
	switch {
	case e.Horizontal():
		return edgeHorizontal
	case e.Vertical():
		return edgeVertical
	default:
		return edgeSloped
	}
}

func (e *edge) FlipDirection() {
	e.start, e.end = e.end, e.start
}

func (e *edge) MoveInwardsBy(offset float64) {
	t := e.Type()
	if t == edgeSloped {
		panic(fmt.Sprint("Cannot move a sloped egde inwards: ", *e))
	}

	var xoff, yoff float64

	middle := graphical.Point{
		X: (e.start.X + e.end.X) / 2,
		Y: (e.start.Y + e.end.Y) / 2,
	}
	path := e.owner.MakeIntoPath()
	if path == nil {
		return
	}
	switch t {
	case edgeHorizontal:
		up := polyclip.Point{X: middle.X, Y: middle.Y - 0.05}
		down := polyclip.Point{X: middle.X, Y: middle.Y + 0.05}
		switch {
		case path.Contains(up):
			yoff = -offset
		case path.Contains(down):
			yoff = offset
		}
	case edgeVertical:
		left := polyclip.Point{X: middle.X - 0.05, Y: middle.Y}
		right := polyclip.Point{X: middle.X + 0.05, Y: middle.Y}
		switch {
		case path.Contains(left):
			xoff = -offset
		case path.Contains(right):
			xoff = offset
		}
	}

	if DEBUG {
		fmt.Printf("Moved edge %v by %v, %v\n", e, xoff, yoff)
	}
	e.start.X += xoff
	e.start.Y += yoff
	e.end.X += xoff
	e.end.Y += yoff
}
