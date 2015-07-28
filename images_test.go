package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestImageResults(test *testing.T) {
	err := filepath.Walk("orig-java/tests/text", func(path string, info os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case info.IsDir():
			return nil
		case filepath.Ext(info.Name()) != ".txt":
			return nil
		}

		r, err := os.Open(path)
		if err != nil {
			test.Error(err)
			return nil
		}
		defer r.Close()

		w := bytes.NewBuffer(nil)
		err = RenderPNG(r, w)
		if err != nil {
			test.Errorf("%s: %s", path, err)
			return nil
		}

		fname, ext := info.Name(), filepath.Ext(info.Name())
		fname = fname[:len(fname)-len(ext)]
		err = diffimg(filepath.Join("testdata", fname+".png"), w)
		if err != nil {
			test.Errorf("%s: %s", path, err)
			return nil
		}
		return nil
	})
	if err != nil {
		test.Error(err)
	}
}

func diffimg(path string, buf *bytes.Buffer) error {
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer r.Close()
	oldImg, err := png.Decode(r)
	if err != nil {
		return fmt.Errorf("decoding %s: %s", path, err)
	}
	newImg, err := png.Decode(buf)
	if err != nil {
		return fmt.Errorf("decoding rendered PNG: %s", err)
	}

	// Compare the images.
	bounds := oldImg.Bounds()
	if newImg.Bounds() != bounds {
		return fmt.Errorf("bounds differ, expected %v, got %v", bounds, newImg.Bounds())
	}
	var diff *image.RGBA
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			oldpx := oldImg.At(x, y)
			newpx := newImg.At(x, y)
			if !colorsEqual(oldpx, newpx) {
				// Difference - draw it in "pink" color.
				if diff == nil {
					diff = image.NewRGBA(bounds)
				}
				diff.Set(x, y, color.RGBA{255, 0, 255, 255})
			}
		}
	}
	// If images are different, write "difference mask" to diff-*.png, and
	// "bad image" to test-*.png
	if diff != nil {
		fname, ext := filepath.Base(path), filepath.Ext(path)
		fname = fname[:len(fname)-len(ext)]
		diffname, outname := "diff-"+fname+".png", "test-"+fname+".png"
		diffbuf := bytes.NewBuffer(nil)
		err := png.Encode(diffbuf, diff)
		if err != nil {
			return fmt.Errorf("cannot encode %s: %s", diffname, err)
		}
		err = ioutil.WriteFile(diffname, diffbuf.Bytes(), 0644)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(outname, buf.Bytes(), 0644)
		if err != nil {
			return err
		}
		return fmt.Errorf("rendered image differs from %s, mask written to %s, image to %s",
			path, diffname, outname)
	}
	return nil
}

func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}
