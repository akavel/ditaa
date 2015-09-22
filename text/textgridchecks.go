package text

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

func (t *Grid) IsBullet(c Cell) bool {
	ch := t.Get(c)
	return (ch == 'o' || ch == '*') &&
		t.IsBlankNon0(c.East()) &&
		t.IsBlankNon0(c.West()) &&
		isAlphNum(t.Get(c.East().East()))
}

func (t *Grid) IsOutOfBounds(c Cell) bool {
	return c.X >= t.Width() || c.Y >= t.Height() || c.X < 0 || c.Y < 0
}

func (t *Grid) IsBlankNon0(c Cell) bool { return t.Get(c) == ' ' }
func (t *Grid) IsBlank(c Cell) bool {
	ch := t.Get(c)
	if ch == 0 {
		return false // FIXME: should this be false, or true (see 'isBlank(x,y)' in Java)
	}
	return ch == ' '
}
func (t *Grid) IsBlankXY(c Cell) bool {
	ch := t.Get(c)
	if ch == 0 {
		return true
	}
	return ch == ' '
}

func (t *Grid) IsBoundary(c Cell) bool {
	ch := t.Get(c)
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

func (t *Grid) IsIntersection(c Cell) bool {
	return intersectionCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsNormalCorner(c Cell) bool {
	return normalCornerCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsRoundCorner(c Cell) bool {
	return roundCornerCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsStub(c Cell) bool {
	return stubCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsCrossOnLine(c Cell) bool {
	return crossOnLineCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsLoneDiagonal(c Cell) bool {
	return loneDiagonalCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsCross(c Cell) bool      { return crossCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsT(c Cell) bool          { return _TCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsK(c Cell) bool          { return _KCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsInverseT(c Cell) bool   { return inverseTCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsInverseK(c Cell) bool   { return inverseKCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsCorner1(c Cell) bool    { return corner1Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsCorner2(c Cell) bool    { return corner2Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsCorner3(c Cell) bool    { return corner3Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsCorner4(c Cell) bool    { return corner4Criteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsStarOnLine(c Cell) bool { return starOnLineCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsLinesEnd(c Cell) bool   { return linesEndCriteria.AnyMatch(t.TestingSubGrid(c)) }
func (t *Grid) IsHorizontalCrossOnLine(c Cell) bool {
	return horizontalCrossOnLineCriteria.AnyMatch(t.TestingSubGrid(c))
}
func (t *Grid) IsVerticalCrossOnLine(c Cell) bool {
	return verticalCrossOnLineCriteria.AnyMatch(t.TestingSubGrid(c))
}

// func (t *Grid) Is(c Cell) bool { return .AnyMatch(t.TestingSubGrid(c))}

func (t *Grid) IsCorner(c Cell) bool { return t.IsNormalCorner(c) || t.IsRoundCorner(c) }
func (t *Grid) IsHorizontalLine(c Cell) bool {
	ch := t.Get(c)
	if ch == 0 {
		return false
	}
	return isOneOf(ch, text_horizontalLines)
}
func (t *Grid) IsVerticalLine(c Cell) bool {
	ch := t.Get(c)
	if ch == 0 {
		return false
	}
	return isOneOf(ch, text_verticalLines)
}
func (t *Grid) IsLine(c Cell) bool { return t.IsHorizontalLine(c) || t.IsVerticalLine(c) }

func (t *Grid) FollowCell(c Cell, blocked *Cell) *CellSet {
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

func (t *Grid) followIntersection(c Cell, blocked *Cell) *CellSet {
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

func (t *Grid) followCorner(c Cell, blocked *Cell) *CellSet {
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

func (t *Grid) followCornerX(c1, c2 Cell, blocked *Cell) *CellSet {
	result := NewCellSet()
	if blocked == nil || *blocked != c1 {
		result.Add(c1)
	}
	if blocked == nil || *blocked != c2 {
		result.Add(c2)
	}
	return result
}

func (t *Grid) followLine(c Cell, blocked *Cell) *CellSet {
	switch {
	case t.IsHorizontalLine(c):
		return t.followBoundariesX(blocked, c.East(), c.West())
	case t.IsVerticalLine(c):
		return t.followBoundariesX(blocked, c.North(), c.South())
	}
	return nil
}

func (t *Grid) followStub(c Cell, blocked *Cell) *CellSet {
	// [akavel] in original code, the condition quit when first boundary found, but that probably shouldn't matter
	return t.followBoundariesX(blocked, c.East(), c.West(), c.North(), c.South())
}

func (t *Grid) followBoundariesX(blocked *Cell, boundaries ...Cell) *CellSet {
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

func (t *Grid) followCrossOnLine(c Cell, blocked *Cell) *CellSet {
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

func (t *Grid) hasEntryPoint(c Cell, entryid int) bool {
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
	return isOneOf(t.Get(c), entries[entryid])
}

func (t *Grid) HasBlankCells() bool {
	for it := t.Iter(); it.Next(); {
		if t.IsBlank(it.Cell()) {
			return true
		}
	}
	return false
}

func (t *Grid) CellContainsDashedLineChar(c Cell) bool {
	return isOneOf(t.Get(c), text_dashedLines)
}

func (t *Grid) IsArrowhead(c Cell) bool {
	return t.IsNorthArrowhead(c) || t.IsSouthArrowhead(c) || t.IsWestArrowhead(c) || t.IsEastArrowhead(c)
}

func (t *Grid) IsNorthArrowhead(c Cell) bool { return t.Get(c) == '^' }
func (t *Grid) IsWestArrowhead(c Cell) bool  { return t.Get(c) == '<' }
func (t *Grid) IsEastArrowhead(c Cell) bool  { return t.Get(c) == '>' }
func (t *Grid) IsSouthArrowhead(c Cell) bool {
	return isOneOf(t.Get(c), "Vv") && t.IsVerticalLine(c.North())
}

func (t *Grid) IsPointCell(c Cell) bool {
	return t.IsCorner(c) || t.IsIntersection(c) || t.IsStub(c) || t.IsLinesEnd(c)
}

func (t *Grid) IsBlankBetweenCharacters(c Cell) bool {
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

func (t *Grid) isOnHorizontalLine(c Cell) bool {
	return t.IsHorizontalLine(c.West()) && t.IsHorizontalLine(c.East())
}

func (t *Grid) isOnVerticalLine(c Cell) bool {
	return t.IsVerticalLine(c.North()) && t.IsVerticalLine(c.South())
}
