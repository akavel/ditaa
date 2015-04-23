package graphical

import (
	"fmt"
	"math"

	"code.google.com/p/jamslam-freetype-go/freetype/raster"
	"github.com/akavel/polyclip-go"
)

type Grid struct {
	W     int `xml:"width"`
	H     int `xml:"height"`
	CellW int `xml:"cellWidth"`
	CellH int `xml:"cellHeight"`
}

type Cell struct {
	X, Y int
}

func (g Grid) CellFor(p Point) Cell {
	return Cell{int(p.X) / g.CellW, int(p.Y) / g.CellH}
}

func (g Grid) CellMinX(cell Cell) float64 { return float64(cell.X * g.CellW) }
func (g Grid) CellMaxX(cell Cell) float64 { return float64((cell.X + 1) * g.CellW) }
func (g Grid) CellMinY(cell Cell) float64 { return float64(cell.Y * g.CellH) }
func (g Grid) CellMaxY(cell Cell) float64 { return float64((cell.Y + 1) * g.CellH) }
func (g Grid) CellMidX(cell Cell) float64 { return float64(cell.X*g.CellW + g.CellW/2) }
func (g Grid) CellMidY(cell Cell) float64 { return float64(cell.Y*g.CellH + g.CellH/2) }

func (g Grid) MinimumOfCellDimensions() float64 { return math.Min(float64(g.CellW), float64(g.CellH)) }

type ShapeType int

const (
	TYPE_SIMPLE ShapeType = iota
	TYPE_ARROWHEAD
	TYPE_POINT_MARKER
	TYPE_DOCUMENT
	TYPE_STORAGE
	TYPE_IO
	TYPE_DECISION
	TYPE_MANUAL_OPERATION // upside-down trapezoid
	TYPE_TRAPEZOID        // rightside-up trapezoid
	TYPE_ELLIPSE
	TYPE_CUSTOM ShapeType = 9999
)

type Shape struct {
	Type        ShapeType `xml:"type"`
	FillColor   *Color    `xml:"fillColor"`
	StrokeColor Color     `xml:"strokeColor"`
	Closed      bool      `xml:"isClosed"`
	Dashed      bool      `xml:"isStrokeDashed"`
	Points      []Point   `xml:"points>point"`
}

func NewShape(points ...Point) *Shape {
	return &Shape{
		Points:      points,
		StrokeColor: Color{A: 255}, // black by default
	}
}

func (s *Shape) CalcArea() float64 {
	// See http://mathworld.wolfram.com/PolygonArea.html
	if len(s.Points) == 0 {
		return 0
	}
	area := float64(0)
	for i := range s.Points {
		p1, p2 := s.Points[i], s.Points[(i+1)%len(s.Points)]
		area += float64(p1.X * p2.Y)
		area -= float64(p2.X * p1.Y)
	}
	return math.Abs(area * 0.5)
}

func (s *Shape) DropsShadow() bool {
	return s.Closed && s.Type != TYPE_ARROWHEAD && s.Type != TYPE_POINT_MARKER && !s.Dashed
}

func (s *Shape) MakeMarkerPaths(g Grid) (outer, inner raster.Path) {
	if len(s.Points) != 1 {
		return nil, nil
	}
	center := s.Points[0]
	diameter := 0.7 * math.Min(float64(g.CellW), float64(g.CellH))
	return Circle(float64(center.X), float64(center.Y), (diameter+STROKE_WIDTH)*0.5),
		Circle(float64(center.X), float64(center.Y), (diameter-STROKE_WIDTH)*0.5)
}

func (s *Shape) MakeIntoPath() polyclip.Contour {
	if len(s.Points) < 2 {
		return nil
	}

	c := polyclip.Contour{}
	for _, p := range s.Points {
		c.Add(polyclip.Point{X: p.X, Y: p.Y})
	}
	return c
}

func (s Shape) Contains(p Point) bool {
	path := s.MakeIntoPath()
	if path == nil {
		return false
	}
	return path.Contains(polyclip.Point{p.X, p.Y})
}

// FIXME(akavel): buggy, fix this... use full polyclip.Construct()
func (s Shape) Intersects(rect Rect) bool {
	for _, p := range s.Points {
		if rect.Contains(p) {
			return true
		}
	}
	for _, p := range []Point{
		rect.Min,
		{X: rect.Min.X, Y: rect.Max.Y},
		rect.Max,
		{X: rect.Max.X, Y: rect.Min.Y},
	} {
		if s.Contains(p) {
			return true
		}
	}
	return false
}

func (s1 Shape) Equals(s2 Shape) bool {
	if len(s1.Points) != len(s2.Points) {
		return false
	}
	pts1 := map[Point]struct{}{}
	for _, p := range s1.Points {
		pts1[p] = struct{}{}
	}
	for _, p := range s2.Points {
		if _, ok := pts1[p]; !ok {
			return false
		}
	}
	return true
}

func (s *Shape) SmallerThan(s2 *Shape) bool {
	bounds := Bounds(s.Points)
	bounds2 := Bounds(s2.Points)

	return bounds.Area() < bounds2.Area()
}

type Rect struct{ Min, Max Point }

func (r Rect) Area() float64 {
	return (r.Max.Y - r.Min.Y) * (r.Max.X - r.Min.X)
}

func (r Rect) Contains(p Point) bool {
	return p.X >= r.Min.X && p.X <= r.Max.X && p.Y >= r.Min.Y && p.Y <= r.Max.Y
}

func Bounds(pp []Point) Rect {
	if len(pp) == 0 {
		return Rect{}
	}
	r := Rect{pp[0], pp[0]}
	for _, p := range pp {
		if p.X < r.Min.X {
			r.Min.X = p.X
		}
		if p.X > r.Max.X {
			r.Max.X = p.X
		}
		if p.Y < r.Min.Y {
			r.Min.Y = p.Y
		}
		if p.Y > r.Max.Y {
			r.Max.Y = p.Y
		}
	}
	return r
}

func specPoints(bb Rect) (p1, p2, p3, p4 Point) {
	p1 = Point{X: bb.Min.X, Y: bb.Min.Y}
	p2 = Point{X: bb.Max.X, Y: bb.Min.Y}
	p3 = Point{X: bb.Max.X, Y: bb.Max.Y}
	p4 = Point{X: bb.Min.X, Y: bb.Max.Y}
	return
}

func (s *Shape) makeDocumentPath() raster.Path {
	bb := Bounds(s.Points)
	p1, p2, p3, p4 := specPoints(bb)
	pmid := Point{X: 0.5 * (bb.Min.X + bb.Max.X), Y: bb.Max.Y}

	path := raster.Path{}
	path.Start(P(p1))
	path.Add1(P(p2))
	path.Add1(P(p3))

	controlDX := (bb.Max.X - bb.Min.X) / 6
	controlDY := (bb.Max.Y - bb.Min.Y) / 8
	path.Add2(P(Point{X: pmid.X + controlDX, Y: pmid.Y - controlDY}), P(pmid))
	path.Add2(P(Point{X: pmid.X - controlDX, Y: pmid.Y + controlDY}), P(p4))
	path.Add1(P(p1))
	return path
}

func (s *Shape) makeIOPath(g Grid /*, opt Options*/) raster.Path {
	if len(s.Points) != 4 {
		return nil
	}
	bb := Bounds(s.Points)
	p1, p2, p3, p4 := specPoints(bb)
	//TODO: handle opt.FixedSlope
	offset := float64(g.CellW) * 0.5

	path := raster.Path{}
	path.Start(P(Point{X: p1.X + offset, Y: p1.Y}))
	path.Add1(P(Point{X: p2.X + offset, Y: p2.Y}))
	path.Add1(P(Point{X: p3.X - offset, Y: p3.Y}))
	path.Add1(P(Point{X: p4.X - offset, Y: p4.Y}))
	path.Add1(P(Point{X: p1.X + offset, Y: p1.Y})) // close path
	return path
}

func (s *Shape) makeTrapezoidPath(g Grid /*, opt Options*/, inverted bool) raster.Path {
	if len(s.Points) != 4 {
		return nil
	}
	bb := Bounds(s.Points)
	//TODO: handle opt.FixedSlope
	offset := float64(g.CellW) * 0.5
	if inverted {
		offset = -offset
	}
	ul := Point{X: bb.Min.X + offset, Y: bb.Min.Y}
	ur := Point{X: bb.Max.X - offset, Y: bb.Min.Y}
	br := Point{X: bb.Max.X + offset, Y: bb.Max.Y}
	bl := Point{X: bb.Min.X - offset, Y: bb.Max.Y}
	//pmid := Point{X:0.5*(bb.Min.X+bb.Max.X), Y:bb.Max.Y}

	path := raster.Path{}
	path.Start(P(ul))
	path.Add1(P(ur))
	path.Add1(P(br))
	path.Add1(P(bl))
	path.Add1(P(ul)) // close path
	return path
}

func (s *Shape) makeDecisionPath() raster.Path {
	if len(s.Points) != 4 {
		return nil
	}
	bb := Bounds(s.Points)
	pmid := Point{X: 0.5 * (bb.Min.X + bb.Max.X), Y: 0.5 * (bb.Min.Y + bb.Max.Y)}
	left := Point{X: bb.Min.X, Y: pmid.Y}
	right := Point{X: bb.Max.X, Y: pmid.Y}
	top := Point{X: pmid.X, Y: bb.Min.Y}
	bottom := Point{X: pmid.X, Y: bb.Max.Y}

	path := raster.Path{}
	path.Start(P(left))
	path.Add1(P(top))
	path.Add1(P(right))
	path.Add1(P(bottom))
	path.Add1(P(left)) // close path
	return path
}

func (s *Shape) makeStoragePath(g Grid) raster.Path {
	if len(s.Points) != 4 {
		return nil
	}
	bb := Bounds(s.Points)
	p1, p2, p3, p4 := specPoints(bb)

	//control point offset X, and Y
	offx := (bb.Max.X - bb.Min.X) / 6
	offytop := float64(g.CellH) / 2
	offybottom := float64(g.CellH) * 10 / 14

	path := raster.Path{}
	// FIXME(akavel): build with cubic Bezier curves and stroke with http://stackoverflow.com/q/408457
	// //top of cylinder
	// path.Start(P(p1))
	// path.Add3(P(Point{X: p1.X + offx, Y: p1.Y + offytop}), P(Point{X: p2.X - offx, Y: p2.Y + offytop}), P(p2))
	// path.Add3(P(Point{X: p2.X - offx, Y: p2.Y - offytop}), P(Point{X: p1.X + offx, Y: p1.Y - offytop}), P(p1))
	// //side of cylinder
	// path.Add1(P(p4))
	// path.Add3(P(Point{X: p4.X + offx, Y: p4.Y + offybottom}), P(Point{X: p3.X - offx, Y: p3.Y + offybottom}), P(p3))
	// path.Add1(P(p2))
	// return path

	Pxy := func(x, y float64) raster.Point { return P(Point{X: x, Y: y}) }
	//top of cylinder
	path.Start(P(p1))
	path.Add2(Pxy(p1.X+offx, p1.Y+offytop), Pxy((p1.X+p2.X)/2, p1.Y+offytop))
	path.Add2(Pxy(p2.X-offx, p2.Y+offytop), P(p2))
	path.Add2(Pxy(p2.X-offx, p2.Y-offytop), Pxy((p1.X+p2.X)/2, p2.Y-offytop))
	path.Add2(Pxy(p1.X+offx, p1.Y-offytop), P(p1))
	//side and bottom of cylinder
	path.Add1(P(p4))
	path.Add2(Pxy(p4.X+offx, p4.Y+offybottom), Pxy((p4.X+p3.X)/2, p4.Y+offybottom))
	path.Add2(Pxy(p3.X-offx, p3.Y+offybottom), P(p3))
	path.Add1(P(p2))
	return path
}

func getCellEdgePointBetween(pointInCell, otherPoint Point, g Grid) Point {
	if pointInCell == otherPoint {
		panic("the two points cannot be the same")
	}
	cell := g.CellFor(pointInCell)
	switch {
	case otherPoint.NorthOf(pointInCell):
		return Point{X: pointInCell.X, Y: float64(g.CellMinY(cell))}
	case otherPoint.SouthOf(pointInCell):
		return Point{X: pointInCell.X, Y: float64(g.CellMaxY(cell))}
	case otherPoint.WestOf(pointInCell):
		return Point{X: float64(g.CellMinX(cell)), Y: pointInCell.Y}
	case otherPoint.EastOf(pointInCell):
		return Point{X: float64(g.CellMaxX(cell)), Y: pointInCell.Y}
	}
	panic("should not reach")
}

func (s *Shape) MakeIntoRenderPath(g Grid /*, opt Options*/) raster.Path {
	if s.Type == TYPE_POINT_MARKER {
		panic("please handle markers separately")
		return nil
		//return s.makeMarkerPath(g)
	}
	if len(s.Points) == 4 {
		switch s.Type {
		case TYPE_DOCUMENT:
			return s.makeDocumentPath()
		case TYPE_IO:
			return s.makeIOPath(g /*, opt*/)
		case TYPE_MANUAL_OPERATION:
			return s.makeTrapezoidPath(g /*, opt*/, true)
		case TYPE_TRAPEZOID:
			return s.makeTrapezoidPath(g /*, opt*/, false)
		case TYPE_DECISION:
			return s.makeDecisionPath()
		case TYPE_STORAGE:
			return s.makeStoragePath(g)
		case TYPE_ELLIPSE:
			// FIXME(akavel): build with cubic Bezier curves and stroke with http://stackoverflow.com/q/408457
			_ = fmt.Sprintf
			//panic(fmt.Sprintf("niy for type %d", s.Type))
			//TODO: fixme
			return nil
		}
	}
	return s.makeOtherPath(g)
}

func (s *Shape) makeOtherPath(g Grid) raster.Path {
	if len(s.Points) < 2 {
		return nil
	}
	path := raster.Path{}
	point, prev, next := s.Points[0], s.Points[len(s.Points)-1], s.Points[1]
	switch point.Type {
	case POINT_NORMAL:
		path.Start(P(point))
	case POINT_ROUND:
		entry := getCellEdgePointBetween(point, prev, g)
		exit := getCellEdgePointBetween(point, next, g)
		path.Start(P(entry))
		path.Add2(P(point), P(exit))
	}
	for i := 1; i < len(s.Points); i++ {
		prev = point
		point = s.Points[i]
		if i < len(s.Points)-1 {
			next = s.Points[i+1]
		} else {
			next = s.Points[0]
		}
		switch point.Type {
		case POINT_NORMAL:
			path.Add1(P(point))
		case POINT_ROUND:
			entry := getCellEdgePointBetween(point, prev, g)
			exit := getCellEdgePointBetween(point, next, g)
			path.Add1(P(entry))
			path.Add2(P(point), P(exit))
		}
	}
	if s.Closed && len(s.Points) > 2 {
		prev = point
		point = s.Points[0]
		switch point.Type {
		case POINT_NORMAL:
			path.Add1(P(point))
		case POINT_ROUND:
			entry := getCellEdgePointBetween(point, prev, g)
			path.Add1(P(entry))
		}
	}
	return path
}
