// +build none

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"net/http"

	"github.com/akavel/ditaa/graphical/dasher"
	"github.com/golang/freetype/raster"
	"golang.org/x/image/math/fixed"
)

func main() {
	port := flag.String("p", ":8888", "port")
	flag.Parse()
	fmt.Println("listening on", *port)

	http.HandleFunc("/", serveImg)
	log.Fatal(http.ListenAndServe(*port, nil))
}

func serveImg(wr http.ResponseWriter, _ *http.Request) {
	wr.Header().Set("content-type", "image/png")
	w, h := 101, 101
	path := raster.Path{}
	dash := dasher.Dasher{
		Length: fixed.I(5),
		A:      &path,
	}
	p := dasher.DeBezierizer{A: &dash}
	// p := DeBezierizer{A: &path}
	// p := &path // reference implementation
	p.Start(fixed.P(1, 1))
	p.Add1(fixed.P(100, 10))
	p.Add2(fixed.P(100, 50), fixed.P(25, 100))
	// p.Start(fixed.P(25, 100))
	p.Add2(fixed.P(13, 0), fixed.P(1, 100))
	r := raster.NewRasterizer(w, h)
	raster.Stroke(r, path, 2<<6, raster.ButtCapper, nil)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	painter := raster.NewRGBAPainter(img)
	painter.SetColor(color.NRGBA{0, 0, 255, 255})
	r.Rasterize(painter)
	err := png.Encode(wr, img)
	if err != nil {
		panic(err)
	}
}
