package main

type AbstractionGrid struct {
	Rows [][]rune
}

func EmptyAbstractionGrid(w, h int) *AbstractionGrid {
	return &AbstractionGrid{Rows: BlankRows(3*w, 3*h)}
}

func NewAbstractionGrid(t *TextGrid, cells *CellSet) *AbstractionGrid {
	g := EmptyAbstractionGrid(3*t.Width(), 3*t.Height())
	for c := range cells.Set {
		if t.IsBlank(c) {
			continue
		}
		for _, x := range abstractionChecks {
			if x.check(t, c) {
				g.Set(c, x.result)
				break
			}
		}
	}
	return g
}

func (g *AbstractionGrid) Set(c Cell, brush AbstractCell) {
	x, y := 3*c.X, 3*c.Y
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 3; dx++ {
			if brush.Get(dx, dy) {
				g.Rows[y+dy][x+dx] = '*'
			}
		}
	}
}

func (g *AbstractionGrid) Height() int {
	return len(g.Rows) / 3
}
func (g *AbstractionGrid) Width() int {
	if len(g.Rows) == 0 {
		return 0
	}
	return len(g.Rows[0]) / 3
}

func (g *AbstractionGrid) GetAsTextGrid() *TextGrid {
	t := NewTextGrid(g.Width(), g.Height())
	for y := range g.Rows {
		for x, ch := range g.Rows[y] {
			if ch != ' ' {
				t.Set(Cell{x / 3, y / 3}, '*')
			}
		}
	}
	return t
}

var abstractionChecks = []struct {
	check  func(*TextGrid, Cell) bool
	result AbstractCell
}{
	{(*TextGrid).IsCross, abCross},
	{(*TextGrid).IsT, abT},
	{(*TextGrid).IsK, abK},
	{(*TextGrid).IsInverseT, abInvT},
	{(*TextGrid).IsInverseK, abInvK},
	{(*TextGrid).IsCorner1, abCorner1},
	{(*TextGrid).IsCorner2, abCorner2},
	{(*TextGrid).IsCorner3, abCorner3},
	{(*TextGrid).IsCorner4, abCorner4},
	{(*TextGrid).IsHorizontalLine, abHLine},
	{(*TextGrid).IsVerticalLine, abVLine},
	{(*TextGrid).IsCrossOnLine, abCross},
	{(*TextGrid).IsStarOnLine, abStar},
}
