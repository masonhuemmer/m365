package main

import "io"

// job is one APS mutation job (mutator-spec.md, "Runner Adapter").
type job struct {
	ID           string `json:"id"`
	FeatureJSON  string `json:"feature_json"`
	GeneratedDir string `json:"generated_dir"`
	WorkDir      string `json:"work_dir"`
	Timeout      string `json:"timeout"`
}

// response is the worker's answer to one job.
type response struct {
	ID       string `json:"id"`
	Outcome  string `json:"outcome"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	Duration int64  `json:"duration"`
}

const (
	testSuccess = "test_success"
	testFailure = "test_failure"
	infraError  = "infrastructure_error"
)

// runFunc runs one job's acceptance test and returns its output and its
// exit error.
type runFunc func(j job) (output string, err error)

// serve reads one job per line from in and writes one response per line
// to out.
func serve(in io.Reader, out io.Writer, run runFunc) error {
	return nil
}

// classify maps a test run to an APS outcome and, for an error, its reason.
func classify(output string, err error) (outcome, reason string) {
	return "", ""
}

// testRunner runs test in the compiled acceptance test binary bin, in dir,
// against the job's IR.
func testRunner(bin, test, dir string) runFunc {
	return func(job) (string, error) { return "", nil }
}
