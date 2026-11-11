package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// featureWithoutIR writes a feature under features/ with no
// build/acceptance/ir beside it, as in a fresh checkout before
// scripts/acceptance.sh runs.
func featureWithoutIR(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "features", "x.feature")
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPlanFeatureReadsTheFeatureTextWhenItHasNoIR(t *testing.T) {
	t.Setenv(irEnv, "")
	Register("irtest known", func(*World, string) error { return nil })
	p := featureWithoutIR(t, "Feature: f\n  Scenario: s\n    Given irtest known\n")
	got, err := planFeature(p)
	want := []scenarioRun{{Name: "s", Runs: []execution{{Steps: []Step{{Kind: "Given", Text: "irtest known"}}}}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("planFeature = %+v, %v; want %+v", got, err, want)
	}
}

func TestPlanFeatureWithoutIRStillRefusesAnOutline(t *testing.T) {
	t.Setenv(irEnv, "")
	p := featureWithoutIR(t, "Feature: f\n  Scenario Outline: o\n    Given irtest known <x>\n\n    Examples:\n      | x |\n      | 1 |\n")
	_, err := planFeature(p)
	if err == nil || !strings.Contains(err.Error(), `unsupported Gherkin "Scenario Outline: o"`) {
		t.Fatalf("planFeature error = %v; want the text parser's refusal", err)
	}
}
