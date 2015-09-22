package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/akavel/ditaa/graphical"
)

const blankBorderSize = 2

var humanColorCodes = map[string]string{
	"GRE": "9D9",
	"BLU": "55B",
	"PNK": "FAA",
	"RED": "E32",
	"YEL": "FF3",
	"BLK": "000",
}

var markupTags = map[string]struct{}{
	"d":  struct{}{},
	"s":  struct{}{},
	"io": struct{}{},
	"c":  struct{}{},
	"mo": struct{}{},
	"tr": struct{}{},
	"o":  struct{}{},
}

var _SPACE = []byte{' '}

type TextGrid struct {
	Rows [][]rune
}

func NewTextGrid(w, h int) *TextGrid {
	if h == 0 {
		return &TextGrid{}
	}
	return &TextGrid{Rows: BlankRows(w, h)}
}

func CopyTextGrid(other *TextGrid) *TextGrid {
	t := TextGrid{}
	t.Rows = make([][]rune, len(other.Rows))
	for y, row := range other.Rows {
		t.Rows[y] = append([]rune(nil), row...)
	}
	return &t
}

func (t TextGrid) Iter() CellIter {
	return CellIter{t.Rows, -1, 0}
}

type CellIter struct {
	rows [][]rune
	x, y int
}

func (it *CellIter) Next() bool {
	if it.y >= len(it.rows) {
		return false
	}
	it.x++
	if it.x >= len(it.rows[it.y]) {
		it.y++
		it.x = 0
	}
	return it.y < len(it.rows)
}
func (it CellIter) Cell() Cell { return Cell{it.x, it.y} }

func (t1 TextGrid) Equals(t2 TextGrid) bool {
	if len(t1.Rows) != len(t2.Rows) {
		return false
	}
	for i := range t1.Rows {
		if len(t1.Rows[i]) != len(t2.Rows[i]) {
			return false
		}
		for j := range t1.Rows[i] {
			if t1.Rows[i][j] != t2.Rows[i][j] {
				return false
			}
		}
	}
	return true
}

func onlyWhitespaces(rs []rune) bool {
	for _, r := range rs {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func (t *TextGrid) Set(c Cell, ch rune) { t.Rows[c.Y][c.X] = ch }
func (t *TextGrid) Get(c Cell) rune {
	if c.X >= t.Width() || c.Y >= t.Height() || c.X < 0 || c.Y < 0 {
		return 0
	}
	return t.Rows[c.Y][c.X]
}

func (t *TextGrid) Height() int { return len(t.Rows) }
func (t *TextGrid) Width() int {
	if len(t.Rows) == 0 {
		return 0
	}
	return len(t.Rows[0])
}

func (t *TextGrid) TestingSubGrid(c Cell) *TextGrid {
	return t.SubGrid(c.X-1, c.Y-1, 3, 3)
}
func (t *TextGrid) SubGrid(x, y, w, h int) *TextGrid {
	g := NewTextGrid(0, 0)
	for i := 0; i < h; i++ {
		g.Rows = append(g.Rows, t.Rows[y+i][x:x+w])
	}
	return g
}

func (t *TextGrid) GetAllNonBlank() *CellSet {
	cells := NewCellSet()
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		if !t.IsBlank(c) {
			cells.Add(c)
		}
	}
	return cells
}

func (t *TextGrid) GetAllBlanksBetweenCharacters() *CellSet {
	cells := NewCellSet()
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		if t.IsBlankBetweenCharacters(c) {
			cells.Add(c)
		}
	}
	return cells
}

func BlankRows(w, h int) [][]rune {
	rows := make([][]rune, h)
	for y := range rows {
		rows[y] = make([]rune, w)
		for x := range rows[y] {
			rows[y][x] = ' '
		}
	}
	return rows
}

func FillCellsWith(rows [][]rune, cells *CellSet, ch rune) {
	for c := range cells.Set {
		switch {
		case c.Y >= len(rows):
			continue
		case c.X >= len(rows[c.Y]):
			continue
		}
		rows[c.Y][c.X] = ch
	}
}

func (t *TextGrid) seedFillOld(seed Cell, newChar rune) *CellSet {
	filled := NewCellSet()
	oldChar := t.Get(seed)
	if oldChar == newChar {
		return filled
	}
	if t.IsOutOfBounds(seed) {
		return filled
	}

	stack := []Cell{seed}

	expand := func(c Cell) {
		if t.Get(c) == oldChar {
			stack = append(stack, c)
		}
	}

	for len(stack) > 0 {
		var c Cell
		c, stack = stack[len(stack)-1], stack[:len(stack)-1]

		t.Set(c, newChar)
		filled.Add(c)

		expand(c.North())
		expand(c.South())
		expand(c.East())
		expand(c.West())
	}
	return filled
}

func (t *TextGrid) fillContinuousArea(c Cell, ch rune) *CellSet {
	return t.seedFillOld(c, ch)
}

// Makes blank all the cells that contain non-text elements.
func (t *TextGrid) RemoveNonText() {
	//the following order is significant
	//since the south-pointing arrowheads
	//are determined based on the surrounding boundaries

	// remove arrowheads
	for it := t.Iter(); it.Next(); {
		if t.IsArrowhead(it.Cell()) {
			t.Set(it.Cell(), ' ')
		}
	}

	// remove color codes
	for _, pair := range t.FindColorCodes() {
		c := pair.Cell
		t.Set(c, ' ')
		c = c.East()
		t.Set(c, ' ')
		c = c.East()
		t.Set(c, ' ')
		c = c.East()
		t.Set(c, ' ')
	}

	// remove boundaries
	rm := []Cell{}
	for it := t.Iter(); it.Next(); {
		if t.IsBoundary(it.Cell()) {
			rm = append(rm, it.Cell())
		}
	}
	for _, c := range rm {
		t.Set(c, ' ')
	}

	// remove markup tags
	for _, pair := range t.findMarkupTags() {
		tag := pair.Tag
		if tag == "" {
			continue
		}
		length := 2 + len(tag)
		t.WriteStringTo(pair.Cell, strings.Repeat(" ", length))
	}
}

func (t *TextGrid) WriteStringTo(c Cell, s string) {
	if t.IsOutOfBounds(c) {
		return
	}
	copy(t.Rows[c.Y][c.X:], []rune(s))
}

func (t *TextGrid) GetStringAt(c Cell, length int) string {
	if t.IsOutOfBounds(c) {
		return ""
	}
	return string(t.Rows[c.Y][c.X : c.X+length])
}

var tagPattern = regexp.MustCompile(`\{(.+?)\}`)

type CellTagPair struct {
	Cell
	Tag string
}

func (t *TextGrid) findMarkupTags() []CellTagPair {
	result := []CellTagPair{}
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		ch := t.Get(c)
		if ch != '{' {
			continue
		}
		rowPart := string(t.Rows[c.Y][c.X:])
		m := tagPattern.FindStringSubmatch(rowPart)
		if len(m) == 0 {
			continue
		}
		tagName := m[1]
		if _, ok := markupTags[tagName]; !ok {
			continue
		}
		result = append(result, CellTagPair{c, tagName})
	}
	return result
}

type Color uint32

type CellColorPair struct {
	Cell
	graphical.Color
}

var (
	colorCodePattern = regexp.MustCompile(`c[A-F0-9]{3}`)
)

func unhex(c byte) uint8 {
	if '0' <= c && c <= '9' {
		return c - '0'
	}
	return 10 + c - 'A'
}

func (t *TextGrid) FindColorCodes() []CellColorPair {
	result := []CellColorPair{}
	w, h := t.Width(), t.Height()
	for yi := 0; yi < h; yi++ {
		for xi := 0; xi < w-3; xi++ {
			c := Cell{xi, yi}
			s := t.GetStringAt(c, 4)
			if colorCodePattern.MatchString(s) {
				cR, cG, cB := s[1], s[2], s[3]
				result = append(result, CellColorPair{
					Cell: c,
					Color: graphical.Color{
						R: unhex(cR) * 17,
						G: unhex(cG) * 17,
						B: unhex(cB) * 17,
						A: 255,
					},
				})
			}
		}
	}
	return result
}

func CopySelectedCells(dst *TextGrid, cells *CellSet, src *TextGrid) {
	for c := range cells.Set {
		dst.Set(c, src.Get(c))
	}
}

func (t *TextGrid) DEBUG() string {
	var buf bytes.Buffer
	buf.WriteString("    " + strings.Repeat("0123456789", t.Width()/10+1) + "\n")
	for i, row := range t.Rows {
		buf.WriteString(fmt.Sprintf("%2d (%s)\n", i, string(row)))
	}
	return buf.String()
}

// ReplaceTypeOnLine replaces letters or numbers that are on horizontal
// or vertical lines, with the appropriate character that will make the
// line continuous (| for vertical and - for horizontal lines)
func (t *TextGrid) ReplaceTypeOnLine() {
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		ch := t.Get(c)
		if !unicode.In(ch, unicode.Digit, unicode.Letter) {
			continue
		}
		onH := t.isOnHorizontalLine(c)
		onV := t.isOnVerticalLine(c)
		switch {
		case onH && onV:
			t.Set(c, '+')
		case onH:
			t.Set(c, '-')
		case onV:
			t.Set(c, '|')
		}
	}
}

func (t *TextGrid) ReplacePointMarkersOnLine() {
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		ch := t.Get(c)
		if !isOneOf(ch, text_pointMarkers) || !t.IsStarOnLine(c) {
			continue
		}
		onH := t.IsHorizontalLine(c.East()) || t.IsHorizontalLine(c.West())
		onV := t.IsVerticalLine(c.North()) || t.IsVerticalLine(c.South())
		switch {
		case onH && onV:
			t.Set(c, '+')
		case onH:
			t.Set(c, '-')
		case onV:
			t.Set(c, '|')
		}
	}
}

func (t *TextGrid) GetPointMarkersOnLine() []Cell {
	result := []Cell{}
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		ch := t.Get(c)
		if isOneOf(ch, text_pointMarkers) && t.IsStarOnLine(c) {
			result = append(result, c)
		}
	}
	return result
}

func (t *TextGrid) FindArrowheads() []Cell {
	result := []Cell{}
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		if t.IsArrowhead(c) {
			result = append(result, c)
		}
	}
	return result
}

type CellStringPair struct {
	C Cell
	S string
}

func (t *TextGrid) FindStrings() []CellStringPair {
	result := []CellStringPair{}
	for y := range t.Rows {
		for x := 0; x < len(t.Rows[y]); x++ {
			start := Cell{x, y}
			if t.IsBlank(start) {
				continue
			}
			s := string(t.Get(start))
			x++
			ch := t.Get(Cell{x, y})
			for {
				s += string(ch)
				x++
				c := Cell{x, y}
				ch = t.Get(c)
				next := t.Get(c.East())
				if (ch == ' ' || ch == 0) && (next == ' ' || next == 0) {
					break
				}
			}
			result = append(result, CellStringPair{start, s})
		}
	}
	return result
}

func (t *TextGrid) OtherStringsStartInTheSameColumn(c Cell) int {
	if !t.IsStringsStart(c) {
		return 0
	}
	result := 0
	for y := range t.Rows {
		cc := Cell{c.X, y}
		if cc != c && t.IsStringsStart(cc) {
			result++
		}
	}
	return result
}

func (t *TextGrid) IsStringsStart(c Cell) bool {
	return !t.IsBlank(c) && t.IsBlank(c.West())
}

func (t *TextGrid) OtherStringsEndInTheSameColumn(c Cell) int {
	if !t.IsStringsEnd(c) {
		return 0
	}
	result := 0
	for y := range t.Rows {
		cc := Cell{c.X, y}
		if cc != c && t.IsStringsEnd(cc) {
			result++
		}
	}
	return result
}

func (t *TextGrid) IsStringsEnd(c Cell) bool {
	return !t.IsBlank(c) && t.IsBlank(c.East())
}
