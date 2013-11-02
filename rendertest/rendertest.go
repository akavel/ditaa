package main

import (
	"bufio"
	"code.google.com/p/freetype-go/freetype/raster"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	sources      = "../orig-java/tests/xmls"
	results      = "imgs"
	STROKE_WIDTH = 1 << 8
)

type Grid struct {
	W     int `xml:"width"`
	H     int `xml:"height"`
	CellW int `xml:"cellWidth"`
	CellH int `xml:"cellHeight"`
}

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

type Color struct {
	R uint8 `xml:"r,attr"`
	G uint8 `xml:"g,attr"`
	B uint8 `xml:"b,attr"`
	A uint8 `xml:"a,attr"`
}

func (c Color) RGBA() color.RGBA {
	return color.RGBA{c.R, c.G, c.B, c.A}
}

var WHITE = color.RGBA{255, 255, 255, 255}

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

type Shape struct {
	Type        ShapeType `xml:"type"`
	FillColor   *Color    `xml:"fillColor"`
	StrokeColor Color     `xml:"strokeColor"`
	Closed      bool      `xml:"isClosed"`
	Dashed      bool      `xml:"isStrokeDashed"`
	Points      []Point   `xml:"points>point"`
}

type Diagram struct {
	XMLName xml.Name `xml:"diagram"`
	Grid    Grid     `xml:"grid"`
	Shapes  []Shape  `xml:"shapes>shape"`
	//TODO: []Text
}

func P(p Point) raster.Point {
	//TODO: handle fractional part too, but probably not needed
	return raster.Point{
		raster.Fix32(int(p.X)) << 8,
		raster.Fix32(int(p.Y)) << 8,
	}
}

func Circle(x, y, r float64) raster.Path {
	//panic(fmt.Sprint(x, y, r))
	F := func(f float64) raster.Fix32 {
		//TODO: verify this is OK
		a := math.Trunc(f)
		b := math.Ldexp(math.Abs(f-a), 8)
		return raster.Fix32(a)<<8 + raster.Fix32(b)
	}
	P := func(x, y float64) raster.Point {
		return raster.Point{F(x), F(y)}
	}
	p1 := P(x+r, y)
	p2 := P(x, y+r)
	p3 := P(x-r, y)
	p4 := P(x, y-r)
	path := raster.Path{}
	path.Start(p1)
	path.Add2(p1, p2)
	path.Add2(p2, p3)
	path.Add2(p3, p4)
	path.Add2(p4, p1)
	return path
}

func (s *Shape) makeMarkerPath(g Grid) raster.Path {
	if len(s.Points) != 1 {
		return nil
	}
	center := s.Points[0]
	diameter := 0.7 * math.Min(float64(g.CellW), float64(g.CellH))
	return Circle(float64(center.X), float64(center.Y), diameter*0.5)
}

func (s *Shape) MakeIntoRenderPath(g Grid, opt Options) raster.Path {
	if s.Type == TYPE_POINT_MARKER {
		return s.makeMarkerPath(g)
	}
	if len(s.Points) == 4 {
		switch s.Type {
		case TYPE_DOCUMENT, TYPE_STORAGE, TYPE_IO, TYPE_DECISION, TYPE_MANUAL_OPERATION, TYPE_TRAPEZOID, TYPE_ELLIPSE:
			//panic(fmt.Sprintf("niy for type %d", s.Type))
			//TODO: fixme
			return nil
		}
	}
	if len(s.Points) < 2 {
		return nil
	}
	path := raster.Path{}
	point, prev, next := s.Points[0], s.Points[len(s.Points)-1], s.Points[1]
	_, _ = prev, next
	//var entry, exit *Point
	switch point.Type {
	case POINT_NORMAL:
		path.Start(P(point))
	case POINT_ROUND:
		//TODO: fixme
		path.Start(P(point))
		//panic("niy")
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
			//TODO: fixme
			path.Add1(P(point))
			//panic("niy")
		}
	}
	if s.Closed && len(s.Points) > 2 {
		path.Add1(P(s.Points[0])) //FIXME: other for POINT_ROUND?
	}
	return path
}

type Options struct{}

func LoadDiagram(path string) (*Diagram, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("loading diagram from '%s': %s", path, err)
	}
	defer r.Close()

	diagram := Diagram{}
	err = xml.NewDecoder(bufio.NewReader(r)).Decode(&diagram)
	if err != nil {
		return nil, fmt.Errorf("decoding diagram from '%s': %s", path, err)
	}
	//panic(fmt.Sprintf("%s: %#v", path, diagram))
	return &diagram, nil
}

func RenderDiagram(img *image.RGBA, diagram *Diagram, opt Options) error {
	for y := 0; y < diagram.Grid.H; y++ {
		for x := 0; x < diagram.Grid.W; x++ {
			img.SetRGBA(x, y, WHITE)
		}
	}

	//TODO: antialiasing options
	//TODO: drop shadows
	//TODO: special handling of storage shapes
	//TODO: sorting of shapes (largest first)
	//TODO: render rest of shapes + collect point markers
	pointMarkers := []Shape{}
	for _, shape := range diagram.Shapes {
		switch shape.Type {
		case TYPE_POINT_MARKER:
			pointMarkers = append(pointMarkers, shape)
			continue
		case TYPE_STORAGE:
			continue
		case TYPE_CUSTOM:
			//TODO: render custom shape
			continue
		}
		if len(shape.Points) == 0 {
			continue
		}

		path := shape.MakeIntoRenderPath(diagram.Grid, opt)

		// fill
		if path != nil && shape.Closed && !shape.Dashed {
			g := raster.NewRasterizer(diagram.Grid.W, diagram.Grid.H)
			g.AddPath(path)
			painter := raster.NewRGBAPainter(img)
			if shape.FillColor != nil {
				painter.SetColor(shape.FillColor.RGBA())
			} else {
				painter.SetColor(WHITE)
			}
			g.Rasterize(painter)
		}

		// draw
		if shape.Type != TYPE_ARROWHEAD {
			//TODO: support dashed lines
			g := raster.NewRasterizer(diagram.Grid.W, diagram.Grid.H)
			//g.AddPath(path)
			raster.Stroke(g, path, STROKE_WIDTH, nil, nil)
			painter := raster.NewRGBAPainter(img)
			painter.SetColor(shape.StrokeColor.RGBA())
			g.Rasterize(painter)
		}
	}
	//TODO: render point markers
	for _, shape := range pointMarkers {
		path := shape.MakeIntoRenderPath(diagram.Grid, opt)
		g := raster.NewRasterizer(diagram.Grid.W, diagram.Grid.H)
		g.AddPath(path)
		painter := raster.NewRGBAPainter(img)
		painter.SetColor(WHITE)
		g.Rasterize(painter)
		g.Clear()
		raster.Stroke(g, path, STROKE_WIDTH, nil, nil)
		painter.SetColor(shape.StrokeColor.RGBA())
		g.Rasterize(painter)
	}
	//TODO: handle text
	return nil
}

func runRender(src, dst string) error {
	diagram, err := LoadDiagram(src)
	if err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, diagram.Grid.W, diagram.Grid.H))
	err = RenderDiagram(img, diagram, Options{})
	if err != nil {
		return err
	}
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	wbuf := bufio.NewWriter(w)
	err = png.Encode(wbuf, img)
	if err != nil {
		return err
	}
	err = wbuf.Flush()
	return err
}

func run() error {
	fnames := []string{}

	os.Mkdir(results, 0666)

	//todo: load files from ../orig-java/tests/xmls/*.xml, then try to render them into some output dir, and link them all on one html page
	err := filepath.Walk(sources, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".xml" {
			return nil
		}
		fnames = append(fnames, info.Name())
		return runRender(path, filepath.Join(results, info.Name()+".png"))
	})

	if err != nil {
		return err
	}

	return err
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
