package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

// maxOutput caps the test output carried in one response.
const maxOutput = 16 << 10

// runFunc runs one job's acceptance test and returns its output and its
// exit error.
type runFunc func(j job) (output string, err error)

// serve reads one job per line from in and writes one response per line
// to out. Nothing else is written to out.
func serve(in io.Reader, out io.Writer, run runFunc) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	enc := json.NewEncoder(out)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if err := enc.Encode(answer(line, run)); err != nil {
			return err
		}
	}
	return sc.Err()
}

// answer runs one job line and builds its response.
func answer(line string, run runFunc) response {
	var j job
	if err := json.Unmarshal([]byte(line), &j); err != nil {
		return response{Outcome: infraError, Error: "malformed job: " + err.Error()}
	}
	start := time.Now()
	output, err := run(j)
	outcome, why := classify(output, err)
	if len(output) > maxOutput {
		output = output[len(output)-maxOutput:]
	}
	return response{ID: j.ID, Outcome: outcome, Output: output, Error: why, Duration: time.Since(start).Nanoseconds()}
}

// classify maps a test run to an APS outcome and, for an error, its reason.
// A run that selected no test is an error, never a pass.
func classify(output string, err error) (outcome, reason string) {
	switch {
	case err == nil && strings.Contains(output, "no tests to run"):
		return infraError, "no test ran"
	case err == nil:
		return testSuccess, ""
	case strings.Contains(output, "--- FAIL"):
		return testFailure, ""
	default:
		return infraError, err.Error()
	}
}

// testRunner runs test in the compiled acceptance test binary bin, in dir,
// against the job's IR, within the job's timeout.
func testRunner(bin, test, dir string) runFunc {
	return func(j job) (string, error) {
		ir, err := filepath.Abs(j.FeatureJSON)
		if err != nil {
			return "", err
		}
		ctx, cancel := jobContext(j.Timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, "-test.run", "^"+test+"$") // #nosec G204 -- binary and test come from the gate script
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "M365_ACCEPTANCE_IR="+ir)
		cmd.WaitDelay = time.Second
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
}

// jobContext applies a job's timeout; an empty or bad one means none.
func jobContext(timeout string) (context.Context, context.CancelFunc) {
	if d, err := time.ParseDuration(timeout); err == nil && d > 0 {
		return context.WithTimeout(context.Background(), d)
	}
	return context.WithCancel(context.Background())
}
