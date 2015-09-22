//WIP

package text

import (
	"regexp"
)

/*
This is a Grid (usually 3x3) that contains the equivalent
of a 2D reqular expression (which uses custom syntax to make
things more visual, but standard syntax is also possible).

The custom syntax is:

	. means anything
	b means any boundary (any of - = / \ + | :)
	! means not boundary (none of - = / \ + | :)
	- means - or =
	| means | or :
	[ means not | nor :
	~ means not - nor =
	^ means a boundary but not - nor =
	( means a boundary but not | nor :
	s means a straight boundary (one of - = + | :)
	S means not a straight boundary (none of - = + | :)

	1 means a cell that has entry point 1
	2 means a cell that has entry point 2
	3 means a cell that has entry point 3
	etc. up to number 8

	%1 means a cell that does not have entry point 1 etc.

See below for an explanation of entry points

+, \, / and the space are literal (as is any other character)


Entry points

	1   2   3
	 *--*--*
	 |     |
	8*     *4
	 |     |
	 *--*--*
	7   6   5

We number the entry points for each cell as in the diagram above.
If a cell is occupied by a character, we define as entry points
the points of the above diagram that the character can touch with
the end of its lines. For example:

	- has entry points 8 and 4,
	| and : have entry points 2 and 6,
	/ has 3 and 7,
	\ has 1 and 5,
	+ has 2, 6, 8 and 4
	etc.

*/
var gridPatternChars = map[byte]string{
	'[':  "[^|:]",
	'|':  "[|:]",
	'-':  "[-=]",
	'!':  "[^-=\\/\\\\+|:]",
	'b':  "[-=\\/\\\\+|:]",
	'^':  "[\\/\\\\+|:]",
	'(':  "[-=\\/\\\\+]",
	'~':  ".",
	'+':  "\\+",
	'\\': "\\\\",
	's':  "[-=+|:]",
	'S':  "[\\/\\\\]",
	'*':  "\\*",

	//entry points
	'1': "[\\\\]",
	'2': "[|:+\\/\\\\]",
	'3': "[\\/]",
	'4': "[-=+\\/\\\\]",
	'5': "[\\\\]",
	'6': "[|:+\\/\\\\]",
	'7': "[\\/]",
	'8': "[-=+\\/\\\\]",
}

var gridPatternCharsInv = map[byte]string{
	'1': "[^\\\\]",
	'2': "[^|:+\\/\\\\]",
	'3': "[^\\/]",
	'4': "[^-=+\\/\\\\]",
	'5': "[^\\\\]",
	'6': "[^|:+\\/\\\\]",
	'7': "[^\\/]",
	'8': "[^-=+\\/\\\\]",
}

type GridPattern [3]*regexp.Regexp
type Criteria []GridPattern

func NewCriterion(rowtop, rowmid, rowbot string) Criteria {
	return Criteria{NewGridPattern(rowtop, rowmid, rowbot)}
}
func NewCriteria(criteria ...Criteria) Criteria {
	c := Criteria{}
	for _, newc := range criteria {
		c = append(c, newc...)
	}
	return c
}
func (c Criteria) AnyMatch(g *Grid) bool {
	for _, p := range c {
		if p.Match(g) {
			return true
		}
	}
	return false
}

func NewGridPattern(rowtop, rowmid, rowbot string) GridPattern {
	return GridPattern{
		mustCompileRow(rowtop),
		mustCompileRow(rowmid),
		mustCompileRow(rowbot),
	}
}
func mustCompileRow(pattern string) *regexp.Regexp {
	re := ""
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if elem, ok := gridPatternChars[c]; ok {
			re += elem
			continue
		}
		if c != '%' {
			re += string(c)
			continue
		}
		// c=='%' -- "negation" of following character
		i++
		c = pattern[i]
		if elem, ok := gridPatternCharsInv[c]; ok {
			re += elem
		}
	}
	return regexp.MustCompile(re)
}
func (p *GridPattern) Match(t *Grid) bool {
	for i, re := range *p {
		if !re.MatchString(string(t.Rows[i])) {
			return false
		}
	}
	return true
}

var (
	crossCriteria    = NewCriterion(".6.", "4+8", ".2.")
	KCriteria        = NewCriterion(".6.", "%4+8", ".2.")
	inverseKCriteria = NewCriterion(".6.", "4+%8", ".2.")
	TCriteria        = NewCriterion(".%6.", "4+8", ".2.")
	inverseTCriteria = NewCriterion(".6.", "4+8", ".%2.")

	// ****** normal corners *******
	normalCorner1Criteria = NewCriterion(".[.", "~+(", ".^.")
	normalCorner2Criteria = NewCriterion(".[.", "(+~", ".^.")
	normalCorner3Criteria = NewCriterion(".^.", "(+~", ".[.")
	normalCorner4Criteria = NewCriterion(".^.", "~+(", ".[.")

	// ******* round corners *******
	roundCorner1Criteria = NewCriterion(".[.", "~/4", ".2.")
	roundCorner2Criteria = NewCriterion(".[.", "4\\~", ".2.")
	roundCorner3Criteria = NewCriterion(".6.", "4/~", ".[.")
	roundCorner4Criteria = NewCriterion(".6.", "~\\8", ".[.")

	// stubs
	stubCriteria = NewCriteria(
		NewCriterion("!^!", "!+!", ".!."),
		NewCriterion("!^!", "!+!", ".-."),
		NewCriterion("!!.", "(+!", "!!."),
		NewCriterion("!!.", "(+|", "!!."),
		NewCriterion(".!.", "!+!", "!^!"),
		NewCriterion(".-.", "!+!", "!^!"),
		NewCriterion(".!!", "!+(", ".!!"),
		NewCriterion(".!!", "|+(", ".!!"))

	// ****** ends of lines ******
	verticalLinesEndCriteria = NewCriteria(
		NewCriterion(".^.", ".|.", ".!."),
		NewCriterion(".^.", ".|.", ".-."),
		NewCriterion(".!.", ".|.", ".^."),
		NewCriterion(".-.", ".|.", ".^."))
	horizontalLinesEndCriteria = NewCriteria(
		NewCriterion("...", "(-!", "..."),
		NewCriterion("...", "(-|", "..."),
		NewCriterion("...", "!-(", "..."),
		NewCriterion("...", "|-(", "..."))

	// ****** others *******
	horizontalCrossOnLineCriteria = NewCriterion("...", "(+(", "...")
	verticalCrossOnLineCriteria   = NewCriterion(".^.", ".+.", ".^.")
	horizontalStarOnLineCriteria  = NewCriteria(
		NewCriterion("...", "(*(", "..."),
		NewCriterion("...", "!*(", "..."),
		NewCriterion("...", "(*!", "..."))
	verticalStarOnLineCriteria = NewCriteria(
		NewCriterion(".^.", ".*.", ".^."),
		NewCriterion(".!.", ".*.", ".^."),
		NewCriterion(".^.", ".*.", ".!."))
	loneDiagonalCriteria = NewCriteria(
		NewCriterion(".%6%7", "%4/%8", "%3%2."),
		NewCriterion("%1%6.", "%4\\%8", ".%2%5"))

	// groups
	intersectionCriteria = NewCriteria(crossCriteria, KCriteria, TCriteria, inverseKCriteria, inverseTCriteria)
	normalCornerCriteria = NewCriteria(normalCorner1Criteria, normalCorner2Criteria, normalCorner3Criteria, normalCorner4Criteria)
	roundCornerCriteria  = NewCriteria(roundCorner1Criteria, roundCorner2Criteria, roundCorner3Criteria, roundCorner4Criteria)
	corner1Criteria      = NewCriteria(normalCorner1Criteria, roundCorner1Criteria)
	corner2Criteria      = NewCriteria(normalCorner2Criteria, roundCorner2Criteria)
	corner3Criteria      = NewCriteria(normalCorner3Criteria, roundCorner3Criteria)
	corner4Criteria      = NewCriteria(normalCorner4Criteria, roundCorner4Criteria)
	cornerCriteria       = NewCriteria(normalCornerCriteria, roundCornerCriteria)
	crossOnLineCriteria  = NewCriteria(horizontalCrossOnLineCriteria, verticalCrossOnLineCriteria)
	starOnLineCriteria   = NewCriteria(horizontalStarOnLineCriteria, verticalStarOnLineCriteria)
	linesEndCriteria     = NewCriteria(horizontalLinesEndCriteria, verticalLinesEndCriteria, stubCriteria)
)
