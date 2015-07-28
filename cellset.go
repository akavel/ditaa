package main

import (
	"fmt"
	"os"
)

type CellSet struct {
	Set map[Cell]struct{}
	typ CellSetType
}
type CellBounds struct{ Min, Max Cell }

func NewCellSet() *CellSet {
	return &CellSet{Set: make(map[Cell]struct{})}
}

func (s *CellSet) Add(c Cell) { s.Set[c] = struct{}{} }

func (s *CellSet) Remove(c Cell) {
	s.typ = SET_UNINITIALIZED
	delete(s.Set, c)
}

func (s *CellSet) Contains(c Cell) bool {
	_, ok := s.Set[c]
	return ok
}

func (s *CellSet) AddAll(s2 *CellSet) {
	for c := range s2.Set {
		s.Set[c] = struct{}{}
	}
}

func (s *CellSet) HasCommonCells(s2 *CellSet) bool {
	for c := range s2.Set {
		if s.Contains(c) {
			return true
		}
	}
	return false
}

func (s *CellSet) Bounds() CellBounds {
	var bb *CellBounds
	for c := range s.Set {
		if bb == nil {
			bb = &CellBounds{Min: c, Max: c}
			continue
		}
		if c.X < bb.Min.X {
			bb.Min.X = c.X
		}
		if c.X > bb.Max.X {
			bb.Max.X = c.X
		}
		if c.Y < bb.Min.Y {
			bb.Min.Y = c.Y
		}
		if c.Y > bb.Max.Y {
			bb.Max.Y = c.Y
		}
	}
	if bb == nil {
		return CellBounds{}
	}
	return *bb
}

func (s1 *CellSet) Equals(s2 *CellSet) bool {
	if len(s1.Set) != len(s2.Set) {
		return false
	}
	for k := range s1.Set {
		if _, ok := s2.Set[k]; !ok {
			return false
		}
	}
	return true
}

func (s *CellSet) Type(grid *TextGrid) (typ CellSetType) {
	if s.typ != SET_UNINITIALIZED {
		return s.typ
	}
	defer func() {
		s.typ = typ
	}()
	if len(s.Set) <= 1 {
		return SET_OPEN
	}

	traced := s.getTypeAccordingToTraceMethod(grid)
	if traced == SET_OPEN || traced == SET_CLOSED {
		return traced
	}
	if traced != SET_UNDETERMINED {
		return SET_UNDETERMINED // [akavel] can this happen?
	}

	filled := s.getTypeAccordingToFillMethod(grid)
	switch filled {
	case SET_HAS_CLOSED_AREA:
		return SET_MIXED
	case SET_OPEN:
		return SET_OPEN
	}

	//in the case that both return undetermined:
	return SET_UNDETERMINED
}

func (s *CellSet) getTypeAccordingToTraceMethod(grid *TextGrid) CellSetType {
	workGrid := NewTextGrid(grid.Width(), grid.Height())
	CopySelectedCells(workGrid, s, grid)

	//start with a line end if it exists or with a "random" cell if not
	start := s.SomeCell()
	for c := range s.Set {
		if workGrid.IsLinesEnd(c) {
			start = c
			break // [akavel] added this; is it ok?
		}
	}
	prev := start
	nexts := workGrid.FollowCell(prev, nil)
	if len(nexts.Set) == 0 {
		return SET_OPEN
	}
	cell := nexts.SomeCell()
	for cell != start {
		nexts = workGrid.FollowCell(cell, &prev)
		switch len(nexts.Set) {
		case 0:
			// found dead end, shape is open
			return SET_OPEN
		case 1:
			prev = cell
			cell = nexts.SomeCell()
		default:
			return SET_UNDETERMINED
		}
	}
	// arrived back to start, shape is closed
	return SET_CLOSED
}

func (s *CellSet) getTypeAccordingToFillMethod(grid *TextGrid) CellSetType {
	tempSet := NewCellSet()
	tempSet.Set = s.Set
	bb := s.Bounds()
	tempSet.translate(-bb.Min.X+1, -bb.Min.Y+1)
	subGrid := grid.SubGrid(bb.Min.X-1, bb.Min.Y-1, bb.Max.X-bb.Min.X+3, bb.Max.Y-bb.Min.Y+3)
	temp := NewTextGrid(0, 0)
	temp.Rows = NewAbstractionGrid(subGrid, tempSet).Rows

	fillCell := temp.foreach(func(c Cell) interface{} {
		if temp.IsBlank(c) {
			return &c
		}
		return nil
	}).(*Cell)
	if fillCell == nil {
		fmt.Fprintln(os.Stderr, "Unexpected error: fill method cannot fill anywhere")
		return SET_UNDETERMINED
	}
	temp.fillContinuousArea(fillCell.X, fillCell.Y, '*')
	if temp.HasBlankCells() {
		return SET_HAS_CLOSED_AREA
	}
	return SET_OPEN
}

func (s *CellSet) SomeCell() Cell {
	for c := range s.Set {
		return c
	}
	return Cell{} // [akavel] TODO: or panic("should not reach") ?
}

func (s *CellSet) translate(dx, dy int) {
	s.typ = SET_UNINITIALIZED
	result := map[Cell]struct{}{}
	for c := range s.Set {
		c.X += dx
		c.Y += dy
		result[c] = struct{}{}
	}
	s.Set = result
}

func (s *CellSet) SubtractSet(s2 *CellSet) {
	s.typ = SET_UNINITIALIZED
	for c := range s2.Set {
		delete(s.Set, c)
	}
}

type CellSetType int

const (
	SET_UNINITIALIZED CellSetType = iota
	SET_CLOSED
	SET_OPEN
	SET_MIXED
	SET_HAS_CLOSED_AREA
	SET_UNDETERMINED
)

func (s CellSet) GetCellsAsString() string {
	out := ""
	for c := range s.Set {
		out += "/" + c.String()
	}
	if out == "" {
		return out
	}
	return out[1:]
}
