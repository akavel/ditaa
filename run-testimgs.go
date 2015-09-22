// +build none

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	forceAll = flag.Bool("a", false, "run all tests even in case of errors")
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	const (
		sources = "testdata/"
		results = "tmp/testimgs"
	)
	os.MkdirAll(results, 0777)
	err := filepath.Walk(sources, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".txt" {
			return nil
		}
		return RunDitaa(path, filepath.Join(results, info.Name()+".png"))
	})
	if err != nil {
		return err
	}
	return nil
}

func RunDitaa(src, dst string) error {
	os.Remove(dst)
	fmt.Println("\nDITAA", src, "->", dst)
	cmd := exec.Command("ditaa", "--pprof=cpu", src, dst)
	out, err := cmd.CombinedOutput()
	os.Stdout.Write(out)
	if err != nil && !*forceAll {
		return err
	}
	os.Rename("cpu.prof", dst+".prof")
	return nil
}
