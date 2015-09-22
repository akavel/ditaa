package main

import (
	"bufio"
	"io"
	"strings"
)

func (t *TextGrid) LoadFrom(r io.Reader) error {
	lines, err := preSplit(r)
	if err != nil {
		return err
	}
	lines = preTrimTrailing(lines)
	// convert tabs to spaces (or remove them if setting is 0)
	preFixTabs(lines, DEFAULT_TAB_SIZE)
	// make all lines of equal length
	// add blank outline around the buffer to prevent fill glitch
	lines = preAddOutline(lines)
	t.Rows = lines
	t.replaceBullets()
	t.replaceHumanColorCodes()

	return nil
}

func preSplit(r io.Reader) (lines [][]rune, err error) {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		lines = append(lines,
			[]rune(scan.Text()))
	}
	if scan.Err() != nil {
		return nil, scan.Err()
	}
	return
}
func preTrimTrailing(lines [][]rune) [][]rune {
	// strip trailing blank lines
	for n := len(lines); n > 0; n-- {
		if !onlyWhitespaces(lines[n-1]) {
			return lines[:n]
		}
	}
	return nil
}
func preFixTabs(rows [][]rune, tabSize int) {
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
func preAddOutline(rows [][]rune) [][]rune {
	maxLen := 0
	for _, row := range rows {
		if len(row) > maxLen {
			maxLen = len(row)
		}
	}

	newrows := make([][]rune, 0, len(rows)+2*blankBorderSize)
	for i := 0; i < blankBorderSize; i++ {
		newrows = append(newrows, appendSpaces(nil, maxLen+2*blankBorderSize))
	}
	for _, row := range rows {
		newrow := make([]rune, 0, maxLen+2*blankBorderSize)
		newrow = appendSpaces(newrow, blankBorderSize)
		newrow = append(newrow, row...)
		newrow = appendSpaces(newrow, cap(newrow)-len(newrow))
		newrows = append(newrows, newrow)
	}
	for i := 0; i < blankBorderSize; i++ {
		newrows = append(newrows, appendSpaces(nil, maxLen+2*blankBorderSize))
	}
	return newrows
}
func (t *TextGrid) replaceBullets() {
	for it := t.Iter(); it.Next(); {
		c := it.Cell()
		if t.IsBullet(c) {
			t.Set(c, ' ')
			t.Set(c.East(), '\u2022')
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

func appendSpaces(row []rune, n int) []rune {
	for i := 0; i < n; i++ {
		row = append(row, ' ')
	}
	return row
}
