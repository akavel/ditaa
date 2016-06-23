// +build none

// cross_compile builds ditaa for all known platforms. Call with: go run tools/cross_compile.go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

var targets = []struct{ os, arch string }{
	{"windows", "386"},
	{"windows", "amd64"},
	{"darwin", "386"},
	{"darwin", "amd64"},
	{"linux", "386"},
	{"linux", "amd64"},
	{"linux", "arm"},
	{"freebsd", "386"},
	{"freebsd", "amd64"},
	{"freebsd", "arm"},
	{"netbsd", "386"},
	{"netbsd", "amd64"},
	{"netbsd", "arm"},
	{"openbsd", "386"},
	{"openbsd", "amd64"},
	{"plan9", "386"},
	{"plan9", "amd64"},
	{"solaris", "amd64"},
}

func main() {
	if _, err := os.Stat("ditaa.go"); err != nil {
		log.Fatal("Please call from main dir as 'go run tools/cross_compile.go'")
	}

	if err := os.Mkdir("bin", 0755); err != nil {
		if !os.IsExist(err) {
			// it's okay for the path to exist already
			log.Fatal(err)
		}
	}

	for _, target := range targets {
		// build environment
		os.Setenv("GOOS", target.os)
		os.Setenv("GOARCH", target.arch)

		// output file name
		ext := ""
		if target.os == "windows" {
			ext = ".exe"
		}
		out := path.Join("bin", fmt.Sprintf("ditaa-%s-%s%s", target.os, target.arch, ext))

		// build
		fmt.Printf("Building %s...", out)
		output, err := exec.Command("go", "build", "-o", out).CombinedOutput()
		if err != nil {
			fmt.Printf("\n%s\n", output)
		} else {
			fmt.Printf(" done.\n")
		}
	}
}
