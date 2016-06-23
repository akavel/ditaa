package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"../graphical"
	"image"
	"image/png"
	"os"
)

const (
	sources = "../orig-java/tests/xmls"
	results = "imgs"
)

func LoadDiagram(path string) (*graphical.Diagram, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("loading diagram from '%s': %s", path, err)
	}
	defer r.Close()

	diagram := graphical.Diagram{}
	err = xml.NewDecoder(bufio.NewReader(r)).Decode(&diagram)
	if err != nil {
		return nil, fmt.Errorf("decoding diagram from '%s': %s", path, err)
	}
	//panic(fmt.Sprintf("%s: %#v", path, diagram))
	//if len(diagram.Labels)>0 {
	//	panic(fmt.Sprintf("%s: %#v", path, diagram.Labels))
	//}
	return &diagram, nil
}

func RunRender(src, dst string) error {
	diagram, err := LoadDiagram(src)
	if err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, diagram.Grid.W, diagram.Grid.H))
	err = graphical.RenderDiagram(img, diagram, graphical.Options{DropShadows: true})
	if err != nil {
		return err
	}
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	wbuf := bufio.NewWriter(w)
	err = png.Encode(wbuf, img)
	if err != nil {
		return err
	}
	err = wbuf.Flush()
	return err
}
