package main

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/akavel/ditaa/graphical"
)

type Cell graphical.Cell //struct{ X, Y int }

func (c Cell) North() Cell { return Cell{c.X, c.Y - 1} }
func (c Cell) South() Cell { return Cell{c.X, c.Y + 1} }
func (c Cell) East() Cell  { return Cell{c.X + 1, c.Y} }
func (c Cell) West() Cell  { return Cell{c.X - 1, c.Y} }

func (c Cell) String() string { return fmt.Sprintf("(%d, %d)", c.X, c.Y) }

func isAlphNum(ch rune) bool             { return unicode.IsLetter(ch) || unicode.IsDigit(ch) }
func isOneOf(ch rune, group string) bool { return strings.ContainsRune(group, ch) }

func (t *TextGrid) IsBullet(x, y int) bool {
	ch := t.Get(x, y)
	return (ch == 'o' || ch == '*') &&
		t.IsBlankNon0(x+1, y) &&
		t.IsBlankNon0(x-1, y) &&
		isAlphNum(t.Get(x+2, y))
}

func (t *TextGrid) IsOutOfBounds(c Cell) bool {
	return c.X >= t.Width() || c.Y >= t.Height() || c.X < 0 || c.Y < 0
}

func (t *TextGrid) IsBlankNon0(x, y int) bool { return t.Get(x, y) == ' ' }
func (t *TextGrid) IsBlank(c Cell) bool {
	ch := t.Get(c.X, c.Y)
	if ch == 0 {
		return false // FIXME: should this be false, or true (see 'isBlank(x,y)' in Java)
	}
	return ch == ' '
}
func (t *TextGrid) IsBlankXY(x, y int) bool {
	ch := t.Get(x, y)
	if ch == 0 {
		return true
	}
	return ch == ' '
}

func (t *TextGrid) IsBoundary(c Cell) bool {
	ch := t.Get(c.X, c.Y)
	switch ch {
	case 0:
		return false
	case '+', '\\', '/':
		return t.IsIntersection(c) ||
			t.IsCorner(c) ||
			t.IsStub(c) ||
			t.IsCrossOnLine(c)
	}
	return isOneOf(ch, text_boundaries) && !t.IsLoneDiagonal(c)
}

func (t *TextGrid) IsIntersection(c Cell) bool {
	return intersectionCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsNormalCorner(c Cell) bool {
	return normalCornerCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsRoundCorner(c Cell) bool {
	return roundCornerCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsStub(c Cell) bool {
	return stubCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsCrossOnLine(c Cell) bool {
	return crossOnLineCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsLoneDiagonal(c Cell) bool {
	return loneDiagonalCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsCross(c Cell) bool      { return crossCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsT(c Cell) bool          { return TCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsK(c Cell) bool          { return KCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsInverseT(c Cell) bool   { return inverseTCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsInverseK(c Cell) bool   { return inverseKCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsCorner1(c Cell) bool    { return corner1Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsCorner2(c Cell) bool    { return corner2Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsCorner3(c Cell) bool    { return corner3Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsCorner4(c Cell) bool    { return corner4Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsStarOnLine(c Cell) bool { return starOnLineCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsLinesEnd(c Cell) bool   { return linesEndCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *TextGrid) IsHorizontalCrossOnLine(c Cell) bool {
	return horizontalCrossOnLineCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *TextGrid) IsVerticalCrossOnLine(c Cell) bool {
	return verticalCrossOnLineCriteria.AnyMatch(t.TestingSubGrid(c))
}

// func (t *TextGrid) Is(c Cell) bool { return .AnyMatch(t.TestingSubGrid(c))}

func (t *TextGrid) IsCorner(c Cell) bool { return t.IsNormalCorner(c) || t.IsRoundCorner(c) }
func (t *TextGrid) IsHorizontalLine(c Cell) bool {
	ch := t.Get(c.X, c.Y)
	if ch == 0 {
		return false
	}
	return isOneOf(ch, text_horizontalLines)
}
func (t *TextGrid) IsVerticalLine(c Cell) bool {
	ch := t.Get(c.X, c.Y)
	if ch == 0 {
		return false
	}
	return isOneOf(ch, text_verticalLines)
}
func (t *TextGrid) IsLine(c Cell) bool { return t.IsHorizontalLine(c) || t.IsVerticalLine(c) }

func (t *TextGrid) FollowCell(c Cell, blocked *Cell) *CellSet {
	switch {
	case t.IsIntersection(c):
		return t.followIntersection(c, blocked)
	case t.IsCorner(c):
		return t.followCorner(c, blocked)
	case t.IsLine(c):
		return t.followLine(c, blocked)
	case t.IsStub(c):
		return t.followStub(c, blocked)
	case t.IsCrossOnLine(c):
		return t.followCrossOnLine(c, blocked)
	}
	panic("Cannot follow cell: cannot determine cell type")
}

func (t *TextGrid) followIntersection(c Cell, blocked *Cell) *CellSet {
	result := NewCellSet()
	check := func(c Cell, entry int) {
		if t.hasEntryPoint(c, entry) {
			result.Add(c)
		}
	}
	check(c.North(), 6)
	check(c.South(), 2)
	check(c.East(), 8)
	check(c.West(), 4)
	if blocked != nil {
		result.Remove(*blocked)
	}
	return result
}

func (t *TextGrid) followCorner(c Cell, blocked *Cell) *CellSet {
	switch {
	case t.IsCorner1(c):
		return t.followCornerX(c.South(), c.East(), blocked)
	case t.IsCorner2(c):
		return t.followCornerX(c.South(), c.West(), blocked)
	case t.IsCorner3(c):
		return t.followCornerX(c.North(), c.West(), blocked)
	case t.IsCorner4(c):
		return t.followCornerX(c.North(), c.East(), blocked)
	}
	return nil
}

func (t *TextGrid) followCornerX(c1, c2 Cell, blocked *Cell) *CellSet {
	result := NewCellSet()
	if blocked == nil || *blocked != c1 {
		result.Add(c1)
	}
	if blocked == nil || *blocked != c2 {
		result.Add(c2)
	}
	return result
}

func (t *TextGrid) followLine(c Cell, blocked *Cell) *CellSet {
	switch {
	case t.IsHorizontalLine(c):
		return t.followBoundariesX(blocked, c.East(), c.West())
	case t.IsVerticalLine(c):
		return t.followBoundariesX(blocked, c.North(), c.South())
	}
	return nil
}

func (t *TextGrid) followStub(c Cell, blocked *Cell) *CellSet {
	// [akavel] in original code, the condition quit when first boundary found, but that probably shouldn't matter
	return t.followBoundariesX(blocked, c.East(), c.West(), c.North(), c.South())
}

func (t *TextGrid) followBoundariesX(blocked *Cell, boundaries ...Cell) *CellSet {
	result := NewCellSet()
	for _, c := range boundaries {
		if blocked != nil && *blocked == c {
			continue
		}
		if t.IsBoundary(c) {
			result.Add(c)
		}
	}
	return result
}

func (t *TextGrid) followCrossOnLine(c Cell, blocked *Cell) *CellSet {
	result := NewCellSet()
	switch {
	case t.IsHorizontalCrossOnLine(c):
		result.Add(c.East())
		result.Add(c.West())
	case t.IsVerticalCrossOnLine(c):
		result.Add(c.North())
		result.Add(c.South())
	}
	if blocked != nil {
		result.Remove(*blocked)
	}
	return result
}

func (t *TextGrid) hasEntryPoint(c Cell, entryid int) bool {
	entries := []string{
		text_entryPoints1,
		text_entryPoints2,
		text_entryPoints3,
		text_entryPoints4,
		text_entryPoints5,
		text_entryPoints6,
		text_entryPoints7,
		text_entryPoints8,
	}
	entryid--
	if entryid >= len(entries) {
		return false
	}
	return isOneOf(t.GetCell(c), entries[entryid])
}

func (t *TextGrid) HasBlankCells() bool {
	for it := t.Iter(); it.Next(); {
		if t.IsBlank(it.Cell()) {
			return true
		}
	}
	return false
}

func (t *TextGrid) CellContainsDashedLineChar(c Cell) bool {
	return isOneOf(t.GetCell(c), text_dashedLines)
}

func (t *TextGrid) IsArrowhead(c Cell) bool {
	return t.IsNorthArrowhead(c) || t.IsSouthArrowhead(c) || t.IsWestArrowhead(c) || t.IsEastArrowhead(c)
}

func (t *TextGrid) IsNorthArrowhead(c Cell) bool { return t.GetCell(c) == '^' }
func (t *TextGrid) IsWestArrowhead(c Cell) bool  { return t.GetCell(c) == '<' }
func (t *TextGrid) IsEastArrowhead(c Cell) bool  { return t.GetCell(c) == '>' }
func (t *TextGrid) IsSouthArrowhead(c Cell) bool {
	return isOneOf(t.GetCell(c), "Vv") && t.IsVerticalLine(c.North())
}

func (t *TextGrid) IsPointCell(c Cell) bool {
	return t.IsCorner(c) || t.IsIntersection(c) || t.IsStub(c) || t.IsLinesEnd(c)
}

func (t *TextGrid) IsBlankBetweenCharacters(c Cell) bool {
	return t.IsBlank(c) && !t.IsBlank(c.East()) && !t.IsBlank(c.West())
}

const (
	text_boundaries             = `/\|-*=:`
	text_undisputableBoundaries = `|-*=:`
	text_horizontalLines        = `-=`
	text_verticalLines          = `|:`
	text_arrowHeads             = `<>^vV`
	text_cornerChars            = `\/+`
	text_pointMarkers           = `*`
	text_dashedLines            = `:~=`
	text_entryPoints1           = `\`
	text_entryPoints2           = `|:+\/`
	text_entryPoints3           = `/`
	text_entryPoints4           = `-=+\/`
	text_entryPoints5           = `\`
	text_entryPoints6           = `|:+\/`
	text_entryPoints7           = `/`
	text_entryPoints8           = `-=+\/`
)

func (t *TextGrid) isOnHorizontalLine(c Cell) bool {
	return t.IsHorizontalLine(c.West()) && t.IsHorizontalLine(c.East())
}

func (t *TextGrid) isOnVerticalLine(c Cell) bool {
	return t.IsVerticalLine(c.North()) && t.IsVerticalLine(c.South())
}
