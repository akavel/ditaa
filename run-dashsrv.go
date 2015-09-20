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
	p := raster.Path{}
	p.Start(fixed.P(1, 1))
	p.Add1(fixed.P(100, 10))
	p.Add2(fixed.P(100, 50), fixed.P(75, 100))
	// p.Add3(fixed.P(50, 33), fixed.P(25, 66), fixed.P(1, 1))
	r := raster.NewRasterizer(w, h)
	raster.Stroke(r, p, 2<<6, nil, nil)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	painter := raster.NewRGBAPainter(img)
	painter.SetColor(color.NRGBA{0, 0, 255, 255})
	r.Rasterize(painter)
	err := png.Encode(wr, img)
	if err != nil {
		panic(err)
	}
}
