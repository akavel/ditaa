// WIP

package text

type abstractCell [9]bool

func (c *abstractCell) Get(x, y int) bool {
	return (*c)[y*3+x]
}

func abpix(source, mask int32) bool {
	switch source & mask {
	case 0:
		return false
	case mask:
		return true
	}
	panic("bad abstract pixel")
}

func paintAbCell(hextop, hexmid, hexbot int32) abstractCell {
	return abstractCell{
		abpix(hextop, 0x100), abpix(hextop, 0x010), abpix(hextop, 0x001),
		abpix(hexmid, 0x100), abpix(hexmid, 0x010), abpix(hexmid, 0x001),
		abpix(hexbot, 0x100), abpix(hexbot, 0x010), abpix(hexbot, 0x001),
	}
}

var (
	abHLine   = paintAbCell(0x000, 0x111, 0x000)
	abVLine   = paintAbCell(0x010, 0x010, 0x010)
	abCorner1 = paintAbCell(0x000, 0x011, 0x010)
	abCorner2 = paintAbCell(0x000, 0x110, 0x010)
	abCorner3 = paintAbCell(0x010, 0x110, 0x000)
	abCorner4 = paintAbCell(0x010, 0x011, 0x000)
	abT       = paintAbCell(0x000, 0x111, 0x010)
	abInvT    = paintAbCell(0x010, 0x111, 0x000)
	abK       = paintAbCell(0x010, 0x011, 0x010)
	abInvK    = paintAbCell(0x010, 0x110, 0x010)
	abCross   = paintAbCell(0x010, 0x111, 0x010)
	abStar    = paintAbCell(0x111, 0x111, 0x111)
)
