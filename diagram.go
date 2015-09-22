package main

import (
	"fmt"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"

	"github.com/akavel/ditaa/embd"
	"github.com/akavel/ditaa/fontmeasure"
	"github.com/akavel/ditaa/graphical"
	"github.com/akavel/ditaa/text"
)

var baseFont = func() *truetype.Font {
	f, err := freetype.ParseFont(embd.File_font_ttf)
	if err != nil {
		panic(err)
	}
	return f
}()

type Diagram struct {
	G graphical.Diagram
}

/*
An outline of the inner workings of this very important (and monstrous)
constructor is presented here. Boundary processing is the first step
of the process:

  1. Copy the grid into a work grid and remove all type-on-line
     and point markers from the work grid
  2. Split grid into distinct shapes by plotting the grid
     onto an AbstractionGrid and its getDistinctShapes() method.
  3. Find all the possible boundary sets of each of the
     distinct shapes. This can produce duplicate shapes (if the boundaries
     are the same when filling from the inside and the outside).
  4. Remove duplicate boundaries.
  5. Remove obsolete boundaries. Obsolete boundaries are the ones that are
     the sum of their parts when plotted as filled shapes. (see method
     removeObsoleteShapes())
  6. Seperate the found boundary sets to open, closed or mixed
     (See CellSet class on how its done).
  7. Are there any closed boundaries?
        * YES. Subtract all the closed boundaries from each of the
          open ones. That should convert the mixed shapes into open.
        * NO. In this (harder) case, we use the method
          breakTrulyMixedBoundaries() of CellSet to break boundaries
          into open and closed shapes (would work in any case, but it's
          probably slower than the other method). This method is based
          on tracing from the lines' ends and splitting when we get to
          an intersection.
  8. If we had to eliminate any mixed shapes, we seperate the found
     boundary sets again to open, closed or mixed.

At this stage, the boundary processing is all complete and we
proceed with using those boundaries to create the shapes:

  1. Create closed shapes.
  2. Create open shapes. That's when the line end corrections are
     also applied, concerning the positioning of the ends of lines
     see methods connectEndsToAnchors() and moveEndsToCellEdges() of
     DiagramShape.
  3. Assign color codes to closed shapes.
  4. Assign extended markup tags to closed shapes.
  5. Create arrowheads.
  6. Create point markers.

Finally, the text processing occurs: [pending]

*/
func NewDiagram(grid *text.Grid) *Diagram {

	workGrid := text.CopyGrid(grid)
	workGrid.ReplaceTypeOnLine()
	workGrid.ReplacePointMarkersOnLine()

	if DEBUG {
		fmt.Print(workGrid.DEBUG())
	}

	boundaries := getAllBoundaries(workGrid)
	boundarySetsStep1 := getDistinctShapes(text.NewAbstractionGrid(workGrid, boundaries))

	if DEBUG {
		fmt.Println("******* Distinct shapes found using AbstractionGrid *******")
		for _, cells := range boundarySetsStep1 {
			cells.PrintAsGrid()
		}
		fmt.Println("******* Same set of shapes after processing them by filling *******")
	}

	//Find all the boundaries by using the special version of the filling method
	//(fills in a different buffer than the buffer it reads from)
	w, h := grid.Width(), grid.Height()
	boundarySetsStep2 := []*text.CellSet{}
	for _, cells := range boundarySetsStep1 {
		//the fill buffer keeps track of which cells have been
		//filled already
		fillBuffer := text.NewGrid(3*w, 3*h)

		for yi := 0; yi < 3*h; yi++ {
			for xi := 0; xi < 3*w; xi++ {
				if !fillBuffer.IsBlankXY(text.Cell{xi, yi}) {
					continue
				}

				copyGrid := text.NewGrid(0, 0)
				copyGrid.Rows = text.NewAbstractionGrid(workGrid, cells).Rows

				boundaries := findBoundariesExpandingFrom(copyGrid, text.Cell{xi, yi})
				if len(boundaries.Set) == 0 {
					continue //i'm not sure why these occur
				}
				boundarySetsStep2 = append(boundarySetsStep2, makeScaledOneThirdEquivalent(boundaries))

				copyGrid.Rows = text.NewAbstractionGrid(workGrid, cells).Rows
				filled := copyGrid.FillContinuousArea(text.Cell{xi, yi}, '*')
				text.FillCellsWith(fillBuffer.Rows, filled, '*')
				text.FillCellsWith(fillBuffer.Rows, boundaries, '-')

				if DEBUG {
					makeScaledOneThirdEquivalent(boundaries).PrintAsGrid()
					fmt.Println("-----------------------------------")
				}
			}
		}
	}

	boundarySetsStep2 = removeDuplicateSets(boundarySetsStep2)
	//TODO: debug print to verify duplicates removed

	if DEBUG {
		fmt.Println("******* First evaluation of openess *******")
	}
	open, closed, mixed := categorizeBoundaries(boundarySetsStep2, workGrid)

	hadToEliminateMixed := false
	if len(mixed) > 0 && len(closed) > 0 {
		// mixed shapes can be eliminated by
		// subtracting all the closed shapes from them
		hadToEliminateMixed = true
		//subtract from each of the mixed sets all the closed sets
		for _, set := range mixed {
			for _, closedSet := range closed {
				set.SubtractSet(closedSet)
			}
			// this is necessary because some mixed sets produce
			// several distinct open sets after you subtract the
			// closed sets from them
			if set.Type(workGrid) == text.SET_OPEN {
				boundarySetsStep2 = remove(boundarySetsStep2, set)
				boundarySetsStep2 = append(boundarySetsStep2, breakIntoDistinctBoundaries2(set, workGrid)...)
			}
		}
	} else if len(mixed) > 0 && len(closed) == 0 {
		// no closed shape exists, will have to
		// handle mixed shape on its own
		// an example of this case is the following:
		// +-----+
		// |  A  |C                 B
		// +  ---+-------------------
		// |     |
		// +-----+
		hadToEliminateMixed = true
		for _, set := range mixed {
			boundarySetsStep2 = remove(boundarySetsStep2, set)
			boundarySetsStep2 = append(boundarySetsStep2, breakTrulyMixedBoundaries(set, workGrid)...)
		}
	}

	if hadToEliminateMixed {
		open, closed, mixed = categorizeBoundaries(boundarySetsStep2, workGrid)
	}

	closed = removeObsoleteShapes(workGrid, closed)

	//TODO: handle allCornersRound commandline option
	allCornersRound := false

	d := Diagram{}
	d.G.Grid = graphical.Grid{
		CellW: CELL_WIDTH,
		CellH: CELL_HEIGHT,
		W:     len(grid.Rows[0]) * CELL_WIDTH,
		H:     len(grid.Rows) * CELL_HEIGHT,
	}
	//closedShapes := []interface{}{}
	for _, set := range closed {
		shape := createClosedComponentFromBoundaryCells(workGrid, set, d.G.Grid, allCornersRound)
		if shape == nil {
			continue
		}
		//switch shape := shape.(type) {
		//case graphical.Shape:
		d.G.Shapes = append(d.G.Shapes, *shape)
		//closedShapes = append(closedShapes, *shape)
		//case CompositeShape:
		//	d.compositeShapes = append(d.compositeShapes, shape)
		//}
	}

	//TODO: handle opt.performSeparationOfCommonEdges
	if true { // TODO: enabled by default, disabled if opt.performSeparationOfCommonEdges != default true
		// FIXME(akavel): as of now, we have only closed shapes here, but this might change with compositeShapes
		d.G.Shapes = separateCommonEdges(d.G.Grid, d.G.Shapes)
		if DEBUG {
			fmt.Println("closed shapes:")
			fmt.Printf("%#v\n", d.G.Shapes)
		}
	}

	//make open shapes
	for _, set := range open {
		switch len(set.Set) {
		case 1: //single cell "shape"
			c := set.SomeCell()
			if grid.CellContainsDashedLineChar(c) {
				break
			}
			shape := NewSmallLine(workGrid, c, d.G.Grid)
			if shape != nil {
				d.G.Shapes = append(d.G.Shapes, *shape)
				ConnectEndsToAnchors(shape, workGrid, d.G.Grid)
			}
		default: //normal shape
			if DEBUG {
				fmt.Println(set.GetCellsAsString())
			}
			shapes := createOpenFromBoundaryCells(workGrid, set, d.G.Grid, allCornersRound)
			for i := range shapes {
				if !shapes[i].Closed {
					ConnectEndsToAnchors(&shapes[i], workGrid, d.G.Grid)
				}
			}
			d.G.Shapes = append(d.G.Shapes, shapes...)
		}
	}

	//assign color codes to shapes
	//TODO: text on line should not change its color

	for _, pair := range grid.FindColorCodes() {
		c := graphical.Cell(pair.Cell)
		p := graphical.Point{X: d.G.Grid.CellMidX(c), Y: d.G.Grid.CellMidY(c)}
		containingShape := FindSmallestShapeContaining(p, d.G.Shapes)
		if containingShape != nil {
			color := pair.Color
			containingShape.FillColor = &color
		}
	}

	//assign markup to shapes
	for _, pair := range grid.FindMarkupTags() {
		cell := graphical.Cell(pair.Cell)
		p := graphical.Point{X: d.G.Grid.CellMidX(cell), Y: d.G.Grid.CellMidY(cell)}
		containingShape := FindSmallestShapeContaining(p, d.G.Shapes)
		if containingShape == nil {
			continue
		}
		shapeCodes := map[string]graphical.ShapeType{
			"d":  graphical.TYPE_DOCUMENT,
			"s":  graphical.TYPE_STORAGE,
			"io": graphical.TYPE_IO,
			"c":  graphical.TYPE_DECISION,
			"mo": graphical.TYPE_MANUAL_OPERATION,
			"tr": graphical.TYPE_TRAPEZOID,
			"o":  graphical.TYPE_ELLIPSE,
		}
		typ, ok := shapeCodes[pair.Tag]
		if ok {
			// TODO(akavel): translate full loadCustomShape() with getFromCustomShapes()
			containingShape.Type = typ
		} else {
			containingShape.Type = graphical.TYPE_CUSTOM
			// TODO(akavel): implement lines below:
			// def := options.processingOptions.getFromCustomShapes(pair.Tag)
			// containingShape.Definition = def
		}
	}

	//make arrowheads
	for _, c := range workGrid.FindArrowheads() {
		s := createArrowhead(workGrid, c, d.G.Grid)
		if s != nil {
			d.G.Shapes = append(d.G.Shapes, *s)
		} else {
			fmt.Fprintf(os.Stderr, "Could not create arrowhead shape. Unexpected error.")
		}
	}

	//make point markers
	for _, c := range grid.GetPointMarkersOnLine() {
		cell := graphical.Cell(c)
		d.G.Shapes = append(d.G.Shapes, graphical.Shape{
			Points: []graphical.Point{
				{X: d.G.Grid.CellMidX(cell), Y: d.G.Grid.CellMidY(cell)},
			},
			Type:        graphical.TYPE_POINT_MARKER,
			FillColor:   &graphical.WHITE,
			StrokeColor: graphical.Color{0, 0, 0, 255},
		})
	}

	d.G.Shapes = removeDuplicateShapes(d.G.Shapes)

	//copy again
	workGrid = text.CopyGrid(grid)
	workGrid.RemoveNonText()

	// ****** handle text *******
	//break up text into groups
	textGroupGrid := text.CopyGrid(workGrid)
	gaps := textGroupGrid.GetAllBlanksBetweenCharacters()
	//kludge
	for c := range gaps.Set {
		textGroupGrid.Set(c, '|')
	}
	nonBlank := textGroupGrid.GetAllNonBlank()
	textGroups := breakIntoDistinctBoundaries(nonBlank)
	if DEBUG {
		fmt.Println(len(textGroups), "text groups found")
	}

	font := fontmeasure.GetFontForHeight(baseFont, d.G.Grid.CellH)

	for _, textGroupCellSet := range textGroups {
		isolationGrid := text.NewGrid(w, h)
		text.CopySelectedCells(isolationGrid, textGroupCellSet, workGrid)
		strings := isolationGrid.FindStrings()
		for _, pair := range strings {
			cell := graphical.Cell(pair.C)
			s := pair.S
			if DEBUG {
				fmt.Println("Found string", s)
			}
			lastCell := graphical.Cell{cell.X + len(s) - 1, cell.Y}

			minX := d.G.Grid.CellMinX(cell)
			y := d.G.Grid.CellMaxY(cell)
			maxX := d.G.Grid.CellMaxX(lastCell)

			textObject := graphical.Label{
				Text:     s,
				FontSize: font.Size,
				X:        int(minX + 0.5),
				Y:        int(y + 0.5),
				Color:    graphical.Color{A: 255},
			}
			if float64(font.WidthFor(s)) > maxX-minX { // does not fit horizontally
				lessWideFont := fontmeasure.GetFontForWidth(baseFont, int(maxX-minX+0.5), s)
				textObject.FontSize = lessWideFont.Size
			}

			textObject.CenterVerticallyBetween(int(d.G.Grid.CellMinY(cell)), int(d.G.Grid.CellMaxY(cell)), font)

			//TODO: if the strings start with bullets they should be aligned to the left

			// position text correctly
			otherStart := isolationGrid.OtherStringsStartInTheSameColumn(text.Cell(cell))
			otherEnd := isolationGrid.OtherStringsEndInTheSameColumn(text.Cell(lastCell))
			if otherStart == 0 && otherEnd == 0 {
				textObject.CenterHorizontallyBetween(int(minX), int(maxX), font)
			} else if otherEnd > 0 && otherStart == 0 {
				textObject.AlignRightEdgeTo(int(maxX), font)
			} else if otherEnd > 0 && otherStart > 0 {
				if otherEnd > otherStart {
					textObject.AlignRightEdgeTo(int(maxX), font)
				} else if otherEnd == otherStart {
					textObject.CenterHorizontallyBetween(int(minX), int(maxX), font)
				}
			}
			d.G.Labels = append(d.G.Labels, textObject)
		}
	}
	// for _, l := range d.G.Labels {
	// 	fmt.Printf("%#v\n", l)
	// }

	if DEBUG {
		fmt.Println("Positioned text")
	}

	//correct the color of the text objects according
	//to the underlying color
	//[MC] TODO

	for i := range d.G.Labels {
		label := &d.G.Labels[i]
		// FIXME(akavel): fix all usages of DPI/dpi
		tmpFont := &fontmeasure.Font{Font: baseFont, DPI: 72}
		shape := FindSmallestShapeIntersecting(label.BoundsFor(tmpFont), d.G.Shapes)
		if shape == nil || shape.FillColor == nil || !IsDark(*shape.FillColor) {
			continue
		}
		label.Color = graphical.WHITE
	}

	//set outline to true for test within custom shapes
	//[MC] TODO

	return &d
}

func removeDuplicateShapes(shapes []graphical.Shape) []graphical.Shape {
	origShapes := []graphical.Shape{}
	for _, s := range shapes {
		isOrig := true
		for _, o := range origShapes {
			if s.Equals(o) {
				isOrig = false
				break
			}
		}
		if isOrig {
			origShapes = append(origShapes, s)
		}
	}

	return origShapes
}

func createClosedComponentFromBoundaryCells(grid *text.Grid, cells *text.CellSet, gg graphical.Grid, allCornersRound bool) *graphical.Shape {
	if cells.Type(grid) == text.SET_OPEN {
		panic("CellSet is open and cannot be handled by this method")
	}
	if len(cells.Set) < 2 {
		return nil
	}

	shape := graphical.NewShape()
	shape.Closed = true
	for c := range cells.Set {
		if grid.CellContainsDashedLineChar(c) {
			shape.Dashed = true
			break
		}
	}

	workGrid := text.NewGrid(grid.Width(), grid.Height())
	text.CopySelectedCells(workGrid, cells, grid)

	start := cells.SomeCell()
	if workGrid.IsCorner(start) {
		shape.Points = append(shape.Points, makePointForCell(start, workGrid, gg, allCornersRound))
	}
	prev := start
	nextCells := workGrid.FollowCell(prev, nil)
	if len(nextCells.Set) == 0 {
		return nil
	}
	cell := nextCells.SomeCell()
	if workGrid.IsCorner(cell) {
		shape.Points = append(shape.Points, makePointForCell(cell, workGrid, gg, allCornersRound))
	}

	for cell != start {
		nextCells = workGrid.FollowCell(cell, &prev)
		if len(nextCells.Set) > 1 {
			return nil
		}
		if len(nextCells.Set) == 1 {
			prev = cell
			cell = nextCells.SomeCell()
			if cell != start && workGrid.IsCorner(cell) {
				shape.Points = append(shape.Points, makePointForCell(cell, workGrid, gg, allCornersRound))
			}
		}
	}

	return shape
}

func removeObsoleteShapes(grid *text.Grid, sets []*text.CellSet) []*text.CellSet {
	if DEBUG {
		fmt.Println("******* Removing obsolete shapes *******")
		fmt.Println("******* Sets before *******")
		for _, set := range sets {
			set.PrintAsGrid()
		}
	}

	filleds := []*text.CellSet{}

	//make filled versions of all the boundary sets
	for _, set := range sets {
		set = getFilledEquivalent(set, grid)
		if set == nil {
			return sets
		}
		filleds = append(filleds, set)
	}

	toRemove := map[int]bool{}
	for _, set := range filleds {
		//find the other sets that have common cells with set
		common := []*text.CellSet{set}
		for _, set2 := range filleds {
			if set != set2 && set.HasCommonCells(set2) {
				common = append(common, set2)
			}
		}
		//it only makes sense for more than 2 sets
		if len(common) == 2 {
			continue
		}

		//find largest set
		largest := set
		for _, set2 := range common {
			if len(set2.Set) > len(largest.Set) {
				largest = set2
			}
		}

		//see if largest is sum of others
		common = remove(common, largest)

		//make the sum set of the small sets on a grid
		bb := largest.Bounds()
		gridOfSmalls := text.NewGrid(bb.Max.X+2, bb.Max.Y+2)
		for _, set2 := range common {
			text.FillCellsWith(gridOfSmalls.Rows, set2, '*')
		}
		gridLargest := text.NewGrid(bb.Max.X+2, bb.Max.Y+2)
		text.FillCellsWith(gridLargest.Rows, largest, '*')

		idx := indexof(filleds, largest)
		if gridLargest.Equals(*gridOfSmalls) {
			toRemove[idx] = true
		}
	}

	setsToRemove := []*text.CellSet{}
	for i := range toRemove {
		setsToRemove = append(setsToRemove, sets[i])
	}

	for _, set := range setsToRemove {
		sets = remove(sets, set)
	}
	return sets
}

func getFilledEquivalent(cells *text.CellSet, grid *text.Grid) *text.CellSet {
	if cells.Type(grid) == text.SET_OPEN {
		result := text.NewCellSet()
		result.AddAll(cells)
		return result
	}
	bb := cells.Bounds()
	grid = text.NewGrid(bb.Max.X+2, bb.Max.Y+2)
	text.FillCellsWith(grid.Rows, cells, '*')

	//find a cell that has a blank both on the east and the west
	// NOTE(akavel): or bottom-right cell, apparently - bug?
	c := text.Cell{0, 0}
	for it := grid.Iter(); it.Next(); {
		c = it.Cell()
		if grid.IsBlank(c) || !grid.IsBlank(c.East()) || !grid.IsBlank(c.West()) {
			continue
		}
		break
	}
	// found
	c = c.East()
	if grid.IsOutOfBounds(c) {
		newcells := text.NewCellSet()
		newcells.AddAll(cells)
		return newcells
	}
	grid.FillContinuousArea(c, '*')
	return grid.GetAllNonBlank()
}

func indexof(vec []*text.CellSet, elem *text.CellSet) int {
	for i := range vec {
		if vec[i] == elem {
			return i
		}
	}
	return -1
}

func categorizeBoundaries(sets []*text.CellSet, grid *text.Grid) (open, closed, mixed []*text.CellSet) {
	//split boundaries to open, closed and mixed
	for _, set := range sets {
		switch set.Type(grid) {
		case text.SET_CLOSED:
			if DEBUG {
				fmt.Println("Closed boundaries:")
				set.PrintAsGrid()
			}
			closed = append(closed, set)
		case text.SET_OPEN:
			if DEBUG {
				fmt.Println("Open boundaries:")
				set.PrintAsGrid()
			}
			open = append(open, set)
		case text.SET_MIXED:
			if DEBUG {
				fmt.Println("Mixed boundaries:")
				set.PrintAsGrid()
			}
			mixed = append(mixed, set)
		}
	}
	return
}

func remove(vec []*text.CellSet, elem *text.CellSet) []*text.CellSet {
	// remove 'set' from vector, if found
	for i := range vec {
		if vec[i] == elem {
			return append(vec[:i], vec[i+1:]...)
		}
	}
	return vec
}

func removeDuplicateSets(list []*text.CellSet) []*text.CellSet {
	uniques := []*text.CellSet{}
	for _, set := range list {
		original := true
		for _, u := range uniques {
			if set.Equals(u) {
				original = false
				break
			}
		}
		if original {
			uniques = append(uniques, set)
		}
	}
	return uniques
}

func makeScaledOneThirdEquivalent(cells *text.CellSet) *text.CellSet {
	bb := cells.Bounds()
	gridBig := text.NewGrid(bb.Max.X+2, bb.Max.Y+2)
	text.FillCellsWith(gridBig.Rows, cells, '*')

	gridSmall := text.NewGrid((bb.Max.X+2)/3, (bb.Max.Y+2)/3)
	for it := gridBig.Iter(); it.Next(); {
		c := it.Cell()
		if !gridBig.IsBlank(c) {
			gridSmall.Set(text.Cell{c.X / 3, c.Y / 3}, '*')
		}
	}
	return gridSmall.GetAllNonBlank()
}

func findBoundariesExpandingFrom(grid *text.Grid, seed text.Cell) *text.CellSet {
	boundaries := text.NewCellSet()
	if grid.IsOutOfBounds(seed) {
		return boundaries
	}
	oldChar := grid.Get(seed)
	newChar := rune(1) //TODO: kludge
	stack := []text.Cell{seed}
	expand := func(c text.Cell) {
		switch grid.Get(c) {
		case oldChar:
			stack = append(stack, c)
		case '*':
			boundaries.Add(c)
		}
	}
	for len(stack) > 0 {
		var c text.Cell
		c, stack = stack[len(stack)-1], stack[:len(stack)-1]
		grid.Set(c, newChar)
		expand(c.North())
		expand(c.South())
		expand(c.East())
		expand(c.West())
	}
	return boundaries
}

func getDistinctShapes(g *text.AbstractionGrid) []*text.CellSet {
	result := []*text.CellSet{}

	tg := text.Grid{Rows: g.Rows}
	nonBlank := tg.GetAllNonBlank()

	distinct := breakIntoDistinctBoundaries(nonBlank)
	for _, set := range distinct {
		temp := text.EmptyAbstractionGrid(g.Width(), g.Height())
		text.FillCellsWith(temp.Rows, set, '*')
		result = append(result, temp.GetAsTextGrid().GetAllNonBlank())
	}
	return result
}

func breakIntoDistinctBoundaries(cells *text.CellSet) []*text.CellSet {
	result := []*text.CellSet{}
	bb := cells.Bounds()
	boundaryGrid := text.NewGrid(bb.Max.X+2, bb.Max.Y+2)
	text.FillCellsWith(boundaryGrid.Rows, cells, '*')

	for c := range cells.Set {
		if boundaryGrid.IsBlankXY(c) {
			continue
		}
		boundarySet := boundaryGrid.FillContinuousArea(c, ' ')
		result = append(result, boundarySet)
	}
	return result
}

func breakIntoDistinctBoundaries2(cells *text.CellSet, grid *text.Grid) []*text.CellSet {
	return getDistinctShapes(text.NewAbstractionGrid(grid, cells))
}

/*
Breaks that:

	+-----+
	|     |
	+  ---+-------------------
	|     |
	+-----+

into the following 3:

	+-----+
	|     |
	+     +
	|     |
	+-----+

	   ---
	       -------------------

Returns a list of boundaries that are either open or closed but not mixed.
*/
func breakTrulyMixedBoundaries(cells *text.CellSet, grid *text.Grid) []*text.CellSet {
	result := []*text.CellSet{}
	visitedEnds := text.NewCellSet()
	workGrid := text.NewGrid(grid.Width(), grid.Height())
	text.CopySelectedCells(workGrid, cells, grid)
	for start := range cells.Set {
		if !workGrid.IsLinesEnd(start) || visitedEnds.Contains(start) {
			continue
		}
		set := text.NewCellSet()
		set.Add(start)

		prev := start
		nexts := workGrid.FollowCell(prev, nil)
		if len(nexts.Set) == 0 {
			panic("This shape is either open but multipart or has only one cell, and cannot be processed by this method")
		}
		cell := nexts.SomeCell()
		set.Add(cell)

		finished := false
		if workGrid.IsLinesEnd(cell) {
			visitedEnds.Add(cell)
			finished = true
		}

		for !finished {
			nexts = workGrid.FollowCell(cell, &prev)
			switch len(nexts.Set) {
			case 0: // do nothing
			case 1:
				set.Add(cell)
				prev = cell
				cell = nexts.SomeCell()
				if workGrid.IsLinesEnd(cell) {
					visitedEnds.Add(cell)
					finished = true
				}
			default:
				finished = true
			}
		}
		result = append(result, set)
	}

	//substract all boundary sets from this CellSet
	whatsLeft := text.NewCellSet()
	whatsLeft.AddAll(cells)
	for _, set := range result {
		whatsLeft.SubtractSet(set)
	}
	result = append(result, whatsLeft)
	return result
}

func getAllBoundaries(g *text.Grid) *text.CellSet {
	set := text.NewCellSet()
	for y, row := range g.Rows {
		for x := range row {
			c := text.Cell{x, y}
			if g.IsBoundary(c) {
				set.Add(c)
			}
		}
	}
	return set
}

func FindSmallestShapeContaining(p graphical.Point, shapes []graphical.Shape) *graphical.Shape {
	var containingShape *graphical.Shape
	for i := range shapes {
		shape := &shapes[i]
		if !shape.Contains(p) {
			continue
		}
		if containingShape == nil {
			containingShape = shape
			continue
		}
		if shape.SmallerThan(containingShape) {
			containingShape = shape
		}
	}
	return containingShape
}

func FindSmallestShapeIntersecting(rect graphical.Rect, shapes []graphical.Shape) *graphical.Shape {
	var intersectingShape *graphical.Shape
	for i := range shapes {
		shape := &shapes[i]
		if !shape.Intersects(rect) {
			continue
		}
		if intersectingShape == nil {
			intersectingShape = shape
			continue
		}
		if shape.SmallerThan(intersectingShape) {
			intersectingShape = shape
		}
	}
	return intersectingShape
}

func IsDark(c graphical.Color) bool {
	brightness := c.R
	if c.G > brightness {
		brightness = c.G
	}
	if c.B > brightness {
		brightness = c.B
	}
	return brightness < 200
}
