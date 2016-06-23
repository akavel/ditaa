// x+build none

/*
The MIT License (MIT)

Copyright (c) 2015 Mateusz Czapliński <czapkofan@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// embd-go version 1.0 (2015-01-14) https://github.com/akavel/embd-go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
)

var (
	out = flag.String("o", "embd/data.go", "Path to generated file.")
	pkg = flag.String("p", "embd", "Package that the generated file should be in.")
	// TODO(akavel): support gzipping & unzipping when requested via option
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE: %s [FLAGS] PATH...\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	//TODO: for each path, detect if it's a file or directory
	//TODO: bail out if any normalized variable name repeats itself
	//TODO: generate file contents, converted appropriately to escaped multiline string blobs
	//TODO[later]: build only one big string constant in init(), then make the variables contain its subslices
	//TODO[later]: make sure we don't follow symlinks (for simplicity)

	contents := Contents{
		// TODO(akavel): quote them properly for command line, not via {{printf "%q"}}
		Args:  os.Args[1:],
		Pkg:   *pkg,
		Files: map[string]File{},
	}
	for _, path := range flag.Args() {
		// TODO(akavel): support directories
		f, err := NewFile(path)
		if err != nil {
			return err
		}
		if old, exists := contents.Files[f.VarName]; exists {
			return fmt.Errorf(
				"generated variable name conflict: '%s' resolves to"+
					"the same variable name %s as '%s'",
				f.Path, f.VarName, old.Path)
		}
		contents.Files[f.VarName] = f
	}

	err := os.MkdirAll(filepath.Dir(*out), 0777)
	if err != nil {
		return err
	}

	out, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	defer w.Flush()

	err = template.Must(template.New("Contents").Parse(Template)).
		Execute(w, contents)
	if err != nil {
		return err
	}

	return nil
}

var Normalize = regexp.MustCompile(`[^A-Za-z0-9]+`)

func GoEscaped(buf []byte) string {
	s := fmt.Sprintf("%q", string(buf))
	return s[1 : len(s)-1]
}

func NewFile(path string) (File, error) {
	f, err := os.Open(path)
	if err != nil {
		return File{}, err
	}

	ch := make(chan string)
	go func() {
		defer f.Close()

		r := bufio.NewReader(f)

		buf := [20]byte{}
		for {
			n, err := io.ReadFull(r, buf[:])
			switch err {
			case io.ErrUnexpectedEOF:
				ch <- GoEscaped(buf[:n])
				fallthrough
			case io.EOF:
				close(ch)
				return
			case nil:
				ch <- GoEscaped(buf[:])
			default:
				panic(fmt.Errorf("%s: %s", path, err))
			}
		}
	}()
	return File{
		Path:          path,
		VarName:       Normalize.ReplaceAllString("File_"+filepath.Base(path), "_"),
		DataFragments: ch,
	}, nil
}

type Contents struct {
	Args  []string
	Pkg   string
	Files map[string]File
}

type File struct {
	VarName, Path string
	DataFragments <-chan string
}

var Template = `
// DO NOT EDIT BY HAND
//
// Generated with:
//
//	embd-go{{range .Args}}{{. | printf " %q"}}{{end}}

package {{.Pkg}}
{{range .Files}}
// {{.VarName}} contains contents of "{{.Path}}" file.
var {{.VarName}} = []byte("{{range .DataFragments}}{{.}}{{end}}")
{{end}}`[1:]

/*
var dirTemplate = `
var {NAME} = map[string][]byte{
{ENTRIES}
}
`[1:]
*/
