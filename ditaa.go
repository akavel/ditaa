package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"runtime/pprof"

	"github.com/akavel/ditaa/graphical"
	"github.com/akavel/ditaa/text"
)

const (
	CELL_WIDTH  = 10
	CELL_HEIGHT = 14
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE: %s INFILE OUTFILE.png\n", os.Args[0])
	}
	profile := flag.String("pprof", "", `["cpu" or empty] write performance profiling info to cpu.prof file`)
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	defer profiler(*profile)()

	err := run(args[0], args[1])
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
	grid := text.NewGrid(0, 0)
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

func profiler(what string) (closer func()) {
	switch what {
	case "cpu":
		f, err := os.Create(what + ".prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		return pprof.StopCPUProfile
	default:
		return func() {}
	}
}
