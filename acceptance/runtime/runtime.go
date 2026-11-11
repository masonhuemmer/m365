package runtime

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

type World struct {
	T    *testing.T
	Code int
	Out  string
	Err  string
	// Sent counts messages the last command delivered to the fake store.
	Sent int
}

type Step struct {
	Kind, Text string
}

type Scenario struct {
	Name  string
	Steps []Step
}

type Feature struct {
	Name      string
	Scenarios []Scenario
}

type Handler func(w *World, text string) error

var handlers []struct {
	prefix string
	fn     Handler
}

func Register(prefix string, fn Handler) {
	handlers = append(handlers, struct {
		prefix string
		fn     Handler
	}{prefix, fn})
}

func Parse(path string) (Feature, error) {
	f, err := os.Open(path) // #nosec G304 -- feature path from generated tests under features/
	if err != nil {
		return Feature{}, err
	}
	defer f.Close()
	var feat Feature
	var cur *Scenario
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		switch {
		case unsupported(line):
			// The runner cannot run these; failing beats skipping them.
			return Feature{}, fmt.Errorf("%s:%d: unsupported Gherkin %q", path, n, line)
		case strings.HasPrefix(line, "Feature:"):
			feat.Name = strings.TrimSpace(strings.TrimPrefix(line, "Feature:"))
		case strings.HasPrefix(line, "Scenario:"):
			feat.Scenarios = append(feat.Scenarios, Scenario{Name: strings.TrimSpace(strings.TrimPrefix(line, "Scenario:"))})
			cur = &feat.Scenarios[len(feat.Scenarios)-1]
		case isStep(line):
			if cur == nil {
				return Feature{}, fmt.Errorf("%s:%d: step before any Scenario: %q", path, n, line)
			}
			kind, text, _ := strings.Cut(line, " ")
			cur.Steps = append(cur.Steps, Step{Kind: kind, Text: text})
		}
	}
	return feat, sc.Err()
}

// RunFeature runs the parser IR of the feature at path (see irPathFor):
// each scenario once, or once per example row as example_<n>.
func RunFeature(t *testing.T, path string) {
	t.Helper()
	// A step with no handler used to pass silently (SDO-566). Check every
	// expanded step first and report every problem before running any.
	runs, err := planFeature(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range runs {
		t.Run(sc.Name, func(t *testing.T) {
			for _, e := range sc.Runs {
				if e.Name == "" {
					runSteps(t, e.Steps)
					continue
				}
				t.Run(e.Name, func(t *testing.T) { runSteps(t, e.Steps) })
			}
		})
	}
}

// runSteps runs one execution in a fresh world.
func runSteps(t *testing.T, steps []Step) {
	t.Helper()
	w := &World{T: t}
	for _, st := range steps {
		if err := dispatch(w, st.Text); err != nil {
			t.Fatalf("%s %s: %v", st.Kind, st.Text, err)
		}
	}
}

func dispatch(w *World, text string) error {
	if h, ok := handlerFor(text); ok {
		return h(w, text)
	}
	return fmt.Errorf("no step handler for %q", text)
}

// handlerFor finds the first handler whose prefix starts the step text.
// The prefix must be the whole step or be followed by a space, so "I run"
// does not catch "I run-something".
func handlerFor(text string) (Handler, bool) {
	for _, h := range handlers {
		if text == h.prefix || strings.HasPrefix(text, h.prefix+" ") {
			return h.fn, true
		}
	}
	return nil, false
}

var stepKinds = []string{"Given ", "When ", "Then ", "And ", "But ", "* "}

func isStep(line string) bool {
	for _, k := range stepKinds {
		if strings.HasPrefix(line, k) {
			return true
		}
	}
	return false
}

// Keywords and step arguments (doc strings, data tables) the runner cannot
// run. Tags (@) and comments (#) carry no instructions and stay ignored.
var unsupportedKeywords = []string{"Background:", "Scenario Outline:", "Scenario Template:", "Examples:", "Scenarios:", "Rule:", `"""`, "```", "|"}

func unsupported(line string) bool {
	for _, k := range unsupportedKeywords {
		if strings.HasPrefix(line, k) {
			return true
		}
	}
	return false
}

// unmatchedSteps lists each step in feat that has no registered handler,
// as "scenario: Kind text".
func unmatchedSteps(feat Feature) []string {
	var out []string
	for _, scn := range feat.Scenarios {
		for _, st := range scn.Steps {
			if _, ok := handlerFor(st.Text); !ok {
				out = append(out, scn.Name+": "+st.Kind+" "+st.Text)
			}
		}
	}
	return out
}

// checkFeature parses a feature and reports unsupported syntax or steps
// with no handler, before any scenario runs.
func checkFeature(path string) (Feature, error) {
	feat, err := Parse(path)
	if err != nil {
		return Feature{}, err
	}
	if missing := unmatchedSteps(feat); len(missing) > 0 {
		return Feature{}, fmt.Errorf("%s: no step handler for:\n  %s", path, strings.Join(missing, "\n  "))
	}
	return feat, nil
}
