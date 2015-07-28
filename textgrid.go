package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

func (t TextGrid) foreach(f func(Cell) interface{}) interface{} {
	for c, iter := t.Iter(); iter.Ok(); c, iter = iter.Iter() {
		result := f(c)
		if result != nil {
			return result
		}
	}
	return nil
}

func (t TextGrid) Iter() (Cell, CellIter) {
	return Cell{0, 0}, CellIter{t.Rows, 0, 0}
}

type CellIter struct {
	rows [][]rune
	x, y int
}

func (iter CellIter) Ok() bool { return iter.y < len(iter.rows) && iter.x < len(iter.rows[0]) }
func (iter CellIter) Iter() (Cell, CellIter) {
	iter.x++
	if iter.x > len(iter.rows[iter.y]) {
		iter.y++
		iter.x = 0
	}
	return Cell{iter.x, iter.y}, iter
}

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

func (t *TextGrid) LoadFrom(r io.Reader) error {
	lines := [][]rune{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := []rune(scanner.Text())
		lines = append(lines, line)
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}

	// strip trailing blank lines
	for i := len(lines) - 1; i >= 0; i-- {
		if !onlyWhitespaces(lines[i]) {
			lines = lines[:i+1]
			break
		}
	}

	fixTabs(lines, DEFAULT_TAB_SIZE)
	t.Rows = lines

	// make all lines of equal length
	// add blank outline around the buffer to prevent fill glitch
	// convert tabs to spaces (or remove them if setting is 0)

	maxLen := 0
	for _, row := range t.Rows {
		if len(row) > maxLen {
			maxLen = len(row)
		}
	}

	newrows := make([][]rune, 0, len(t.Rows)+2*blankBorderSize)
	for i := 0; i < blankBorderSize; i++ {
		newrows = append(newrows, appendSpaces(nil, maxLen+2*blankBorderSize))
	}
	for _, row := range t.Rows {
		newrow := make([]rune, 0, maxLen+2*blankBorderSize)
		newrow = appendSpaces(newrow, blankBorderSize)
		newrow = append(newrow, row...)
		newrow = appendSpaces(newrow, cap(newrow)-len(newrow))
		newrows = append(newrows, newrow)
	}
	for i := 0; i < blankBorderSize; i++ {
		newrows = append(newrows, appendSpaces(nil, maxLen+2*blankBorderSize))
	}
	t.Rows = newrows

	t.replaceBullets()
	t.replaceHumanColorCodes()

	return nil
}

func onlyWhitespaces(rs []rune) bool {
	for _, r := range rs {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func fixTabs(rows [][]rune, tabSize int) {
	for y, row := range rows {
		newrow := make([]rune, 0, len(row))
		for _, c := range row {
			if c == '\t' {
				newrow = appendSpaces(newrow, tabSize-len(newrow)%tabSize)
			} else {
				newrow = append(newrow, c)
			}
		}
		rows[y] = newrow
	}
}

func appendSpaces(row []rune, n int) []rune {
	for i := 0; i < n; i++ {
		row = append(row, ' ')
	}
	return row
}

func (t *TextGrid) replaceBullets() {
	for y, row := range t.Rows {
		for x, _ := range row {
			if t.IsBullet(x, y) {
				t.Set(x, y, ' ')
				t.Set(x+1, y, '\u2022')
			}
		}
	}
}

func (t *TextGrid) replaceHumanColorCodes() {
	for y, row := range t.Rows {
		s := string(row)
		for k, v := range humanColorCodes {
			k, v = "c"+k, "c"+v
			s = strings.Replace(s, k, v, -1)
		}
		t.Rows[y] = []rune(s)
	}
}

func (t *TextGrid) Set(x, y int, ch rune) { t.Rows[y][x] = ch }
func (t *TextGrid) Get(x, y int) rune {
	if x >= t.Width() || y >= t.Height() || x < 0 || y < 0 {
		return 0
	}
	return t.Rows[y][x]
}
func (t *TextGrid) SetCell(c Cell, ch rune) { t.Set(c.X, c.Y, ch) }
func (t *TextGrid) GetCell(c Cell) rune     { return t.Get(c.X, c.Y) }

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
	for y := range t.Rows {
		for x := range t.Rows[y] {
			c := Cell{x, y}
			if !t.IsBlank(c) {
				cells.Add(c)
			}
		}
	}
	return cells
}

func (t *TextGrid) GetAllBlanksBetweenCharacters() *CellSet {
	cells := NewCellSet()
	for y := range t.Rows {
		for x := range t.Rows[y] {
			c := Cell{x, y}
			if t.IsBlankBetweenCharacters(c) {
				cells.Add(c)
			}
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
	oldChar := t.GetCell(seed)
	if oldChar == newChar {
		return filled
	}
	if t.IsOutOfBounds(seed) {
		return filled
	}

	stack := []Cell{seed}

	expand := func(c Cell) {
		if t.GetCell(c) == oldChar {
			stack = append(stack, c)
		}
	}

	for len(stack) > 0 {
		var c Cell
		c, stack = stack[len(stack)-1], stack[:len(stack)-1]

		t.SetCell(c, newChar)
		filled.Add(c)

		expand(c.North())
		expand(c.South())
		expand(c.East())
		expand(c.West())
	}
	return filled
}

func (t *TextGrid) seedFill2(seed Cell, newChar rune) *CellSet {
	filled := NewCellSet()
	oldChar := t.GetCell(seed)
	if t.IsOutOfBounds(seed) {
		return filled
	}

	var expandDFS func(c Cell)
	expandDFS = func(c Cell) {
		if t.GetCell(c) != oldChar {
			return
		}
		t.SetCell(c, newChar)
		filled.Add(c)

		expandDFS(c.North())
		expandDFS(c.South())
		expandDFS(c.East())
		expandDFS(c.West())
	}
	expandDFS(seed)
	return filled
}

func (t *TextGrid) fillContinuousArea(x, y int, ch rune) *CellSet {
	return t.seedFillOld(Cell{x, y}, ch)
}

// Makes blank all the cells that contain non-text elements.
func (t *TextGrid) RemoveNonText() {
	//the following order is significant
	//since the south-pointing arrowheads
	//are determined based on the surrounding boundaries

	// remove arrowheads
	t.foreach(func(c Cell) interface{} {
		if t.IsArrowhead(c) {
			t.SetCell(c, ' ')
		}
		return nil
	})

	// remove color codes
	for _, pair := range t.FindColorCodes() {
		c := pair.Cell
		t.SetCell(c, ' ')
		c = c.East()
		t.SetCell(c, ' ')
		c = c.East()
		t.SetCell(c, ' ')
		c = c.East()
		t.SetCell(c, ' ')
	}

	// remove boundaries
	rm := []Cell{}
	t.foreach(func(c Cell) interface{} {
		if t.IsBoundary(c) {
			rm = append(rm, c)
		}
		return nil
	})
	for _, c := range rm {
		t.SetCell(c, ' ')
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
	t.foreach(func(c Cell) interface{} {
		ch := t.GetCell(c)
		if ch != '{' {
			return nil
		}
		rowPart := string(t.Rows[c.Y][c.X:])
		m := tagPattern.FindStringSubmatch(rowPart)
		if len(m) == 0 {
			return nil
		}
		tagName := m[1]
		if _, ok := markupTags[tagName]; !ok {
			return nil
		}
		result = append(result, CellTagPair{c, tagName})
		return nil
	})
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
		dst.SetCell(c, src.GetCell(c))
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
	for c, iter := t.Iter(); iter.Ok(); c, iter = iter.Iter() {
		ch := t.GetCell(c)
		if !unicode.In(ch, unicode.Digit, unicode.Letter) {
			continue
		}
		onH := t.isOnHorizontalLine(c)
		onV := t.isOnVerticalLine(c)
		switch {
		case onH && onV:
			t.SetCell(c, '+')
		case onH:
			t.SetCell(c, '-')
		case onV:
			t.SetCell(c, '|')
		}
	}
}

func (t *TextGrid) ReplacePointMarkersOnLine() {
	for yi := 0; yi < t.Height(); yi++ {
		for xi := 0; xi < t.Width(); xi++ {
			cell := Cell{xi, yi}
			c := t.GetCell(cell)
			if !isOneOf(c, text_pointMarkers) || !t.IsStarOnLine(cell) {
				continue
			}
			isOnHorizontalLine :=
				isOneOf(t.GetCell(cell.East()), text_horizontalLines) ||
					isOneOf(t.GetCell(cell.West()), text_horizontalLines)
			isOnVerticalLine :=
				isOneOf(t.GetCell(cell.North()), text_verticalLines) ||
					isOneOf(t.GetCell(cell.South()), text_verticalLines)
			switch {
			case isOnHorizontalLine && isOnVerticalLine:
				t.Set(xi, yi, '+')
				// if DEBUG {
				// 	fmt.Printf("replaced marker on line '%v' with +\n", c)
				// }
			case isOnHorizontalLine:
				t.Set(xi, yi, '-')
				// if DEBUG {
				// 	fmt.Printf("replaced marker on line '%v' with -\n", c)
				// }
			case isOnVerticalLine:
				t.Set(xi, yi, '|')
				// if DEBUG {
				// 	fmt.Printf("replaced marker on line '%v' with |\n", c)
				// }
			}
		}
	}
}

func (t *TextGrid) GetPointMarkersOnLine() []Cell {
	result := []Cell{}
	w, h := t.Width(), t.Height()
	for yi := 0; yi < h; yi++ {
		for xi := 0; xi < w; xi++ {
			c := Cell{xi, yi}
			ch := t.GetCell(c)
			if isOneOf(ch, text_pointMarkers) && t.IsStarOnLine(c) {
				result = append(result, c)
			}
		}
	}
	return result
}

func (t *TextGrid) FindArrowheads() []Cell {
	result := []Cell{}
	w, h := t.Width(), t.Height()
	for yi := 0; yi < h; yi++ {
		for xi := 0; xi < w; xi++ {
			c := Cell{xi, yi}
			if t.IsArrowhead(c) {
				result = append(result, c)
			}
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
			s := string(t.Get(x, y))
			x++
			c := t.Get(x, y)
			for {
				s += string(c)
				x++
				c = t.Get(x, y)
				next := t.Get(x+1, y)
				if (c == ' ' || c == 0) && (next == ' ' || next == 0) {
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
