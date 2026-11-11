package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// responses decodes every stdout line as one protocol response.
func responses(t *testing.T, out string) []response {
	t.Helper()
	var got []response
	for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		var r response
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("stdout line is not a protocol response: %q", line)
		}
		r.Duration = 0
		got = append(got, r)
	}
	return got
}

func TestServeAnswersEachJobOnItsOwnLine(t *testing.T) {
	in := strings.NewReader(`{"id":"m1","feature_json":"a.json","timeout":"30s"}` + "\n" +
		`{"id":"m2","feature_json":"b.json","timeout":"30s"}` + "\n")
	run := func(j job) (string, error) {
		if j.FeatureJSON == "a.json" {
			return "--- FAIL: TestAcceptance_x (0.00s)\nFAIL\n", errors.New("exit status 1")
		}
		return "PASS\n", nil
	}
	var out bytes.Buffer
	if err := serve(in, &out, run); err != nil {
		t.Fatal(err)
	}
	want := []response{
		{ID: "m1", Outcome: testFailure, Output: "--- FAIL: TestAcceptance_x (0.00s)\nFAIL\n"},
		{ID: "m2", Outcome: testSuccess, Output: "PASS\n"},
	}
	if got := responses(t, out.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("responses\n%+v\nwant\n%+v", got, want)
	}
}

func TestServeAnswersAMalformedJobAsAnError(t *testing.T) {
	var out bytes.Buffer
	run := func(job) (string, error) { t.Fatal("a malformed job must not run"); return "", nil }
	if err := serve(strings.NewReader("not json\n"), &out, run); err != nil {
		t.Fatal(err)
	}
	want := []response{{Outcome: infraError, Error: "malformed job: invalid character 'o' in literal null (expecting 'u')"}}
	if got := responses(t, out.String()); !reflect.DeepEqual(got, want) {
		t.Fatalf("responses %+v, want %+v", got, want)
	}
}

func TestClassifyMapsTestRunsToAPSOutcomes(t *testing.T) {
	cases := []struct {
		name, output string
		err          error
		outcome, why string
	}{
		{"passed", "PASS\n", nil, testSuccess, ""},
		{"nothing ran", "testing: warning: no tests to run\nPASS\n", nil, infraError, "no test ran"},
		{"failed", "--- FAIL: TestAcceptance_x (0.00s)\nFAIL\n", errors.New("exit status 1"), testFailure, ""},
		{"did not start", "", errors.New("fork/exec /x: no such file or directory"), infraError, "fork/exec /x: no such file or directory"},
		{"killed", "partial\n", errors.New("signal: killed"), infraError, "signal: killed"},
	}
	for _, c := range cases {
		if outcome, why := classify(c.output, c.err); outcome != c.outcome || why != c.why {
			t.Errorf("%s: classify = %q, %q; want %q, %q", c.name, outcome, why, c.outcome, c.why)
		}
	}
}

// TestWorkerHelper is the acceptance test that the testRunner tests start
// in a child process. In a normal run M365_ACCEPTANCE_IR is unset and it
// does nothing.
func TestWorkerHelper(t *testing.T) {
	p := os.Getenv("M365_ACCEPTANCE_IR")
	if p == "" {
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	switch strings.TrimSpace(string(b)) {
	case "fail":
		t.Fatal("planted failure")
	case "sleep":
		time.Sleep(10 * time.Second)
	}
}

// irFile writes body to a file and returns its path relative to the test's
// working directory, so testRunner must resolve it before changing dir.
func irFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "feature.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(wd, p)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestTestRunnerRunsItsTestAgainstTheJobIR(t *testing.T) {
	run := testRunner(os.Args[0], "TestWorkerHelper", t.TempDir())
	out, err := run(job{FeatureJSON: irFile(t, "pass"), Timeout: "30s"})
	if outcome, why := classify(out, err); outcome != testSuccess {
		t.Fatalf("passing IR: %s %s\n%s", outcome, why, out)
	}
	out, err = run(job{FeatureJSON: irFile(t, "fail"), Timeout: "30s"})
	if outcome, _ := classify(out, err); outcome != testFailure || !strings.Contains(out, "planted failure") {
		t.Fatalf("failing IR: %s\n%s", outcome, out)
	}
}

func TestTestRunnerReportsATestThatDoesNotExist(t *testing.T) {
	run := testRunner(os.Args[0], "TestNoSuchAcceptanceTest", t.TempDir())
	out, err := run(job{FeatureJSON: irFile(t, "pass"), Timeout: "30s"})
	if outcome, why := classify(out, err); outcome != infraError || why != "no test ran" {
		t.Fatalf("classify = %s %q\n%s", outcome, why, out)
	}
}

func TestTestRunnerStopsAtTheJobTimeout(t *testing.T) {
	run := testRunner(os.Args[0], "TestWorkerHelper", t.TempDir())
	start := time.Now()
	out, err := run(job{FeatureJSON: irFile(t, "sleep"), Timeout: "200ms"})
	if outcome, _ := classify(out, err); outcome != infraError || time.Since(start) > 5*time.Second {
		t.Fatalf("classify = %s after %s\n%s", outcome, time.Since(start), out)
	}
}
