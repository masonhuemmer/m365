// Command runner is the persistent runner worker that the APS
// gherkin-mutator drives (scripts/acceptance-mutation.sh). It reads one job
// per stdin line and runs one acceptance test against the job's mutated IR.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	bin := flag.String("bin", "", "compiled acceptance test binary (go test -c ./acceptance/generated)")
	test := flag.String("test", "", "acceptance test to run, for example TestAcceptance_cli_exit_codes")
	dir := flag.String("dir", "acceptance/generated", "working directory for the test binary")
	flag.Parse()
	if *bin == "" || *test == "" {
		fmt.Fprintln(os.Stderr, "usage: runner -bin BINARY -test NAME [-dir DIR]")
		os.Exit(2)
	}
	abs, err := filepath.Abs(*bin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := serve(os.Stdin, os.Stdout, testRunner(abs, *test, *dir)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
