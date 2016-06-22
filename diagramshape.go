package main

import (
	"fmt"

	"./graphical"
)

func NewSmallLine(grid *TextGrid, c Cell, gg graphical.Grid) *graphical.Shape {
	cc := graphical.Cell(c)
	switch {
	case grid.IsHorizontalLine(c):
		return graphical.NewShape(
			graphical.Point{X: gg.CellMinX(cc), Y: gg.CellMidY(cc)},
			graphical.Point{X: gg.CellMaxX(cc) - 1, Y: gg.CellMidY(cc)},
		)
	case grid.IsVerticalLine(c):
		return graphical.NewShape(
			graphical.Point{X: gg.CellMidX(cc), Y: gg.CellMinY(cc)},
			graphical.Point{X: gg.CellMidX(cc), Y: gg.CellMaxY(cc) - 1},
		)
	}
	return nil
}

func ConnectEndsToAnchors(s *graphical.Shape, grid *TextGrid, gg graphical.Grid) {
	if s.Closed {
		return
	}
	n := len(s.Points)
	// println(n)
	for _, line := range []struct{ end, next *graphical.Point }{
		{&s.Points[0], &s.Points[1]},
		{&s.Points[n-1], &s.Points[n-2]},
	} {
		var x, y float64
		switch {
		case line.next.NorthOf(*line.end):
			x, y = line.end.X, line.end.Y+float64(gg.CellH)
		case line.next.SouthOf(*line.end):
			x, y = line.end.X, line.end.Y-float64(gg.CellH)
		case line.next.WestOf(*line.end):
			x, y = line.end.X+float64(gg.CellW), line.end.Y
		case line.next.EastOf(*line.end):
			x, y = line.end.X-float64(gg.CellW), line.end.Y
		}
		anchor := gg.CellFor(graphical.Point{X: x, Y: y})
		c := Cell(anchor)
		if grid.IsArrowhead(c) || grid.IsCorner(c) || grid.IsIntersection(c) {
			line.end.X, line.end.Y = gg.CellMidX(anchor), gg.CellMidY(anchor)
			line.end.Locked = true
		}
	}
}

func createOpenFromBoundaryCells(grid *TextGrid, cells *CellSet, gg graphical.Grid, allCornersRound bool) []graphical.Shape {
	if cells.Type(grid) != SET_OPEN {
		panic("CellSet is closed and cannot be handled by this method")
	}
	if len(cells.Set) == 0 {
		return []graphical.Shape{}
	}

	shapes := []graphical.Shape{}
	workGrid := NewTextGrid(grid.Width(), grid.Height())
	CopySelectedCells(workGrid, cells, grid)

	// if DEBUG {
	// 	fmt.Println("Making composite shape from grid:")
	// 	workGrid.printDebug()
	// }

	visited := NewCellSet()
	for c := range cells.Set {
		// fmt.Println("cell", c)
		if workGrid.IsLinesEnd(c) {
			// fmt.Println("- is lines end")
			nextCells := workGrid.FollowCell(c, nil)
			// fmt.Println("- nextCells", nextCells)
			shapes = append(shapes, growEdgesFromCell(workGrid, gg, allCornersRound, nextCells.SomeCell(), c, visited)...)
			break
		}
	}

	//dashed shapes should "infect" the rest of the shapes
	dashedShapeExists := false
	for _, s := range shapes {
		if s.Dashed {
			dashedShapeExists = true
			break
		}
	}
	if dashedShapeExists {
		for i := range shapes {
			shapes[i].Dashed = true
		}
	}

	return shapes
}

// func callfromline() int {
// 	_, _, line, _ := runtime.Caller(1)
// 	return line
// }

func growEdgesFromCell(grid *TextGrid, gg graphical.Grid, allCornersRound bool, c, prev Cell, visited *CellSet) []graphical.Shape {
	result := []graphical.Shape{}
	visited.Add(prev)
	shape := graphical.NewShape(makePointForCell(prev, grid, gg, allCornersRound))
	// if DEBUG {
	// 	fmt.Printf("point at %s (call from line: %d)", prev, callfromline())
	// }
	if grid.CellContainsDashedLineChar(prev) {
		shape.Dashed = true
	}

	for finished := false; !finished; {
		visited.Add(c)
		if grid.IsPointCell(c) {
			shape.Points = append(shape.Points, makePointForCell(c, grid, gg, allCornersRound))
		}
		if grid.CellContainsDashedLineChar(c) {
			shape.Dashed = true
		}
		if grid.IsLinesEnd(c) {
			finished = true
		}
		nextCells := grid.FollowCell(c, &prev)
		if len(nextCells.Set) == 1 {
			prev = c
			c = nextCells.SomeCell()
		} else { // 3- or 4- way intersection
			finished = true
			for nextCell := range nextCells.Set {
				result = append(result, growEdgesFromCell(grid, gg, allCornersRound, nextCell, c, visited)...)
			}
		}
	}

	result = append(result, *shape)
	return result
}

func makePointForCell(c Cell, grid *TextGrid, gg graphical.Grid, allCornersRound bool) graphical.Point {
	var typ graphical.PointType
	switch {
	case grid.IsCorner(c) && allCornersRound:
		typ = graphical.POINT_ROUND
	case grid.IsNormalCorner(c):
		typ = graphical.POINT_NORMAL
	case grid.IsRoundCorner(c):
		typ = graphical.POINT_ROUND
	case grid.IsLinesEnd(c) || grid.IsIntersection(c):
		typ = graphical.POINT_NORMAL
	default:
		panic(fmt.Sprint("Cannot make point for cell", c))
	}
	return graphical.Point{
		X:    gg.CellMidX(graphical.Cell(c)),
		Y:    gg.CellMidY(graphical.Cell(c)),
		Type: typ,
	}
}

func createArrowhead(grid *TextGrid, c Cell, gg graphical.Grid) *graphical.Shape {
	if !grid.IsArrowhead(c) {
		return nil
	}

	BLACK := graphical.Color{A: 255}
	s := graphical.Shape{
		Closed:      true,
		FillColor:   &BLACK,
		StrokeColor: BLACK,
		Type:        graphical.TYPE_ARROWHEAD,
	}
	cc := graphical.Cell(c)
	switch {
	case grid.IsNorthArrowhead(c):
		s.Points = []graphical.Point{
			{X: gg.CellMidX(cc), Y: gg.CellMinY(cc)},
			{X: gg.CellMinX(cc), Y: gg.CellMaxY(cc)},
			{X: gg.CellMaxX(cc), Y: gg.CellMaxY(cc)},
		}
	case grid.IsSouthArrowhead(c):
		s.Points = []graphical.Point{
			{X: gg.CellMinX(cc), Y: gg.CellMinY(cc)},
			{X: gg.CellMidX(cc), Y: gg.CellMaxY(cc)},
			{X: gg.CellMaxX(cc), Y: gg.CellMinY(cc)},
		}
	case grid.IsWestArrowhead(c):
		s.Points = []graphical.Point{
			{X: gg.CellMaxX(cc), Y: gg.CellMinY(cc)},
			{X: gg.CellMinX(cc), Y: gg.CellMidY(cc)},
			{X: gg.CellMaxX(cc), Y: gg.CellMaxY(cc)},
		}
	case grid.IsEastArrowhead(c):
		s.Points = []graphical.Point{
			{X: gg.CellMinX(cc), Y: gg.CellMinY(cc)},
			{X: gg.CellMaxX(cc), Y: gg.CellMidY(cc)},
			{X: gg.CellMinX(cc), Y: gg.CellMaxY(cc)},
		}
	default:
		return nil
	}
	return &s
}
