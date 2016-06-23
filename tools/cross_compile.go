// +build none

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

type Target struct {
	os   string
	arch string
}

var targets = []Target{
	{os: "windows", arch: "386"},
	{os: "windows", arch: "amd64"},
	{os: "darwin", arch: "386"},
	{os: "darwin", arch: "amd64"},
	{os: "linux", arch: "386"},
	{os: "linux", arch: "amd64"},
	{os: "linux", arch: "arm"},
	{os: "freebsd", arch: "386"},
	{os: "freebsd", arch: "amd64"},
	{os: "freebsd", arch: "arm"},
	{os: "netbsd", arch: "386"},
	{os: "netbsd", arch: "amd64"},
	{os: "netbsd", arch: "arm"},
	{os: "openbsd", arch: "386"},
	{os: "openbsd", arch: "amd64"},
	{os: "plan9", arch: "386"},
	{os: "plan9", arch: "amd64"},
	{os: "solaris", arch: "amd64"},
}

func main() {
	if _, err := os.Stat("ditaa.go"); err != nil {
		log.Fatal("Please call from main dir as 'go run tools/cross_compile.go'")
	}

	os.Mkdir("bin", 0755)

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
		cmd := exec.Command("go", "build", "-o", out)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("\n%s\n", output)
		} else {
			fmt.Printf(" done.\n")
		}

	}
}
