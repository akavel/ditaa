package text

import (
	"fmt"
	"strings"
)

func (s *CellSet) PrintAsGrid() {
	bb := s.Bounds()
	g := NewGrid(bb.Max.X+2, bb.Max.Y+2)
	FillCellsWith(g.Rows, s, '*')
	g.printDebug()
}

func (t *Grid) printDebug() {
	fmt.Println("    " + strings.Repeat("0123456789", t.Width()/10+1))
	for i, row := range t.Rows {
		fmt.Printf("%2d (%s)\n", i, string(row))
	}
}
