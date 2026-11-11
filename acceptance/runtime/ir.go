package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// irEnv names a parser IR file that replaces a feature's own IR. Only the
// acceptance-mutation worker sets it, for the one test it runs.
const irEnv = "M365_ACCEPTANCE_IR"

// irStep, irScenario and irFeature are the parser's JSON IR (APS
// parser-spec.md).
type irStep struct {
	Keyword string `json:"keyword"`
	Text    string `json:"text"`
}

type irScenario struct {
	Name     string              `json:"name"`
	Steps    []irStep            `json:"steps"`
	Examples []map[string]string `json:"examples"`
}

type irFeature struct {
	Name       string       `json:"name"`
	Background []irStep     `json:"background"`
	Scenarios  []irScenario `json:"scenarios"`
}

// execution is one run of a scenario: its example name (empty when the
// scenario has no examples) and its steps, Background first, with
// placeholders filled.
type execution struct {
	Name  string
	Steps []Step
}

// scenarioRun is a scenario and its executions.
type scenarioRun struct {
	Name string
	Runs []execution
}

// nonIdent is the generator's test-name rule: cli/help -> cli_help.
var nonIdent = regexp.MustCompile(`[^A-Za-z0-9_]`)

// placeholder is an APS parameter in step text.
var placeholder = regexp.MustCompile(`<([A-Za-z0-9_]+)>`)

// irPathFor returns the IR file for a feature path, or the file named by
// M365_ACCEPTANCE_IR. scripts/acceptance.sh writes the IR to
// build/acceptance/ir/<test name>.json beside features/.
func irPathFor(featurePath string) string {
	if p := os.Getenv(irEnv); p != "" {
		return p
	}
	p := filepath.ToSlash(featurePath)
	root, rel := "", p
	if i := strings.Index(p, "features/"); i >= 0 {
		root, rel = p[:i], p[i+len("features/"):]
	}
	name := nonIdent.ReplaceAllString(strings.TrimSuffix(rel, ".feature"), "_")
	return root + "build/acceptance/ir/" + name + ".json"
}

// loadIR reads a parser IR file.
func loadIR(path string) (irFeature, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- IR path from generated tests or the mutation worker
	if err != nil {
		return irFeature{}, err
	}
	var f irFeature
	if err := json.Unmarshal(b, &f); err != nil {
		return irFeature{}, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// expand turns a feature's IR into scenario runs: one execution for a
// scenario without examples, one per example row otherwise.
func expand(f irFeature) ([]scenarioRun, error) {
	var out []scenarioRun
	for _, sc := range f.Scenarios {
		run := scenarioRun{Name: sc.Name}
		if len(sc.Examples) == 0 {
			run.Runs = []execution{{Steps: withBackground(f.Background, sc.Steps)}}
		}
		for i, row := range sc.Examples {
			name := fmt.Sprintf("example_%d", i+1)
			filled, err := fill(sc.Steps, row)
			if err != nil {
				return nil, fmt.Errorf("scenario %q %s: %w", sc.Name, name, err)
			}
			run.Runs = append(run.Runs, execution{Name: name, Steps: withBackground(f.Background, filled)})
		}
		out = append(out, run)
	}
	return out, nil
}

// withBackground returns the Background steps followed by the scenario's.
func withBackground(background, steps []irStep) []Step {
	out := make([]Step, 0, len(background)+len(steps))
	for _, s := range append(append([]irStep{}, background...), steps...) {
		out = append(out, Step{Kind: s.Keyword, Text: s.Text})
	}
	return out
}

// fill substitutes one example row into each step.
func fill(steps []irStep, row map[string]string) ([]irStep, error) {
	out := make([]irStep, len(steps))
	for i, s := range steps {
		text, err := substitute(s.Text, row)
		if err != nil {
			return nil, err
		}
		out[i] = irStep{Keyword: s.Keyword, Text: text}
	}
	return out, nil
}

// substitute fills each <name> in text from row.
func substitute(text string, row map[string]string) (string, error) {
	missing := ""
	out := placeholder.ReplaceAllStringFunc(text, func(m string) string {
		name := m[1 : len(m)-1]
		v, ok := row[name]
		if !ok && missing == "" {
			missing = name
		}
		return v
	})
	if missing != "" {
		return "", fmt.Errorf("no example value for <%s>", missing)
	}
	return out, nil
}

// planFeature loads and expands a feature's IR and checks that every
// expanded step has a handler, reporting all that do not. A feature with no
// IR file and no M365_ACCEPTANCE_IR (a test calling RunFeature before
// scripts/acceptance.sh has run) is read as text instead.
func planFeature(featurePath string) ([]scenarioRun, error) {
	f, err := loadIR(irPathFor(featurePath))
	if errors.Is(err, fs.ErrNotExist) && os.Getenv(irEnv) == "" {
		return planFromText(featurePath)
	}
	if err != nil {
		return nil, err
	}
	runs, err := expand(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", featurePath, err)
	}
	if missing := unmatchedSteps(asFeature(runs)); len(missing) > 0 {
		return nil, fmt.Errorf("%s: no step handler for:\n  %s", featurePath, strings.Join(missing, "\n  "))
	}
	return runs, nil
}

// planFromText runs a feature through the text parser, which refuses
// outlines, examples and background, so such a feature still needs its IR.
func planFromText(featurePath string) ([]scenarioRun, error) {
	feat, err := checkFeature(featurePath)
	if err != nil {
		return nil, err
	}
	runs := make([]scenarioRun, 0, len(feat.Scenarios))
	for _, sc := range feat.Scenarios {
		runs = append(runs, scenarioRun{Name: sc.Name, Runs: []execution{{Steps: sc.Steps}}})
	}
	return runs, nil
}

// asFeature names each execution scenario/example_n for unmatchedSteps.
func asFeature(runs []scenarioRun) Feature {
	var f Feature
	for _, r := range runs {
		for _, e := range r.Runs {
			name := r.Name
			if e.Name != "" {
				name += "/" + e.Name
			}
			f.Scenarios = append(f.Scenarios, Scenario{Name: name, Steps: e.Steps})
		}
	}
	return f
}
