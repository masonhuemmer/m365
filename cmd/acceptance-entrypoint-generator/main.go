package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	list := flag.Bool("list", false, "print each feature's test name and path, tab-separated, and exit")
	flag.Parse()
	if err := run(*list); err != nil {
		fmt.Fprintln(os.Stderr, "acceptance-entrypoint-generator:", err)
		os.Exit(1)
	}
}

func run(list bool) error {
	if !list {
		if err := generate("features", "acceptance/generated"); err != nil {
			return err
		}
		features, err := listFeatures("features")
		if err != nil {
			return err
		}
		return writeMetadata(features, "acceptance/generated", "build/acceptance/ir", "build/acceptance/metadata")
	}
	features, err := listFeatures("features")
	if err != nil {
		return err
	}
	for _, f := range features {
		fmt.Printf("%s\t%s\n", f.Name, f.Path)
	}
	return nil
}
