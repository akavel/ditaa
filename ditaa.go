package main

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"

	"github.com/akavel/ditaa/graphical"
)

const (
	DEFAULT_TAB_SIZE = 8
	CELL_WIDTH       = 10
	CELL_HEIGHT      = 14
)

func main() {
	if len(os.Args[1:]) != 2 {
		fmt.Fprintf(os.Stderr, "USAGE: %s INFILE OUTFILE.png\n", os.Args[0])
		os.Exit(1)
	}

	err := run(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(2)
	}
}

func run(infile, outfile string) error {
	r, err := os.Open(infile)
	if err != nil {
		return err
	}
	w, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer w.Close()
	wbuf := bufio.NewWriter(w)
	err = RenderPNG(r, wbuf)
	if err != nil {
		return err
	}
	err = wbuf.Flush()
	if err != nil {
		return err
	}
	return nil
}

func RenderPNG(r io.Reader, w io.Writer) error {
	grid := NewTextGrid(0, 0)
	err := grid.LoadFrom(r)
	if err != nil {
		return err
	}
	if DEBUG {
		fmt.Println("Using grid:")
		fmt.Print(grid.DEBUG())
		//fmt.Print(grid.DEBUG()) // why this gets printed twice in Java code?
	}
	diagram := NewDiagram(grid)

	img := image.NewRGBA(image.Rect(0, 0, diagram.G.Grid.W, diagram.G.Grid.H))
	err = graphical.RenderDiagram(img, &diagram.G, graphical.Options{DropShadows: true}, baseFont)
	if err != nil {
		return err
	}

	err = png.Encode(w, img)
	if err != nil {
		return err
	}
	return nil
}
