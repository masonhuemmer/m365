package runtime

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIRPathForFollowsTheGeneratorNameRule(t *testing.T) {
	t.Setenv(irEnv, "")
	cases := map[string]string{
		"../../features/cli/help.feature":        "../../build/acceptance/ir/cli_help.json",
		"../../features/mail/send-reply.feature": "../../build/acceptance/ir/mail_send_reply.json",
		"features/top.feature":                   "build/acceptance/ir/top.json",
	}
	for feature, want := range cases {
		if got := irPathFor(feature); got != want {
			t.Errorf("irPathFor(%q) = %q, want %q", feature, got, want)
		}
	}
}

func TestIRPathForPrefersTheMutationIR(t *testing.T) {
	t.Setenv(irEnv, "/work/mutations/m1/feature.json")
	if got := irPathFor("../../features/cli/help.feature"); got != "/work/mutations/m1/feature.json" {
		t.Fatalf("irPathFor = %q, want the %s file", got, irEnv)
	}
}

func writeIR(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "feature.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadIRReadsTheParserIR(t *testing.T) {
	p := writeIR(t, `{"name":"f","background":[{"keyword":"Given","text":"b"}],"scenarios":[{"name":"s","steps":[{"keyword":"When","text":"I run \"<args>\""}],"examples":[{"args":"--help"}]}]}`)
	got, err := loadIR(p)
	want := irFeature{
		Name:       "f",
		Background: []irStep{{Keyword: "Given", Text: "b"}},
		Scenarios: []irScenario{{
			Name:     "s",
			Steps:    []irStep{{Keyword: "When", Text: `I run "<args>"`}},
			Examples: []map[string]string{{"args": "--help"}},
		}},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("loadIR = %+v, %v; want %+v", got, err, want)
	}
	if _, err := loadIR(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("loadIR of a missing file: want an error")
	}
}

func TestExpandRunsAScenarioWithoutExamplesOnceAfterTheBackground(t *testing.T) {
	f := irFeature{
		Background: []irStep{{Keyword: "Given", Text: "b"}},
		Scenarios: []irScenario{{Name: "s", Steps: []irStep{
			{Keyword: "When", Text: "a"},
			{Keyword: "Then", Text: `stdout contains "<p>"`},
		}}},
	}
	got, err := expand(f)
	want := []scenarioRun{{Name: "s", Runs: []execution{{Steps: []Step{
		{Kind: "Given", Text: "b"},
		{Kind: "When", Text: "a"},
		{Kind: "Then", Text: `stdout contains "<p>"`},
	}}}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("expand = %+v, %v; want %+v", got, err, want)
	}
}

func TestExpandRunsEachExampleRowWithItsValues(t *testing.T) {
	f := irFeature{Scenarios: []irScenario{{
		Name: "o",
		Steps: []irStep{
			{Keyword: "When", Text: `I run "<args>"`},
			{Keyword: "Then", Text: "exit code <code>"},
		},
		Examples: []map[string]string{
			{"args": "--help", "code": "0"},
			{"args": "mail", "code": "3"},
		},
	}}}
	got, err := expand(f)
	want := []scenarioRun{{Name: "o", Runs: []execution{
		{Name: "example_1", Steps: []Step{{Kind: "When", Text: `I run "--help"`}, {Kind: "Then", Text: "exit code 0"}}},
		{Name: "example_2", Steps: []Step{{Kind: "When", Text: `I run "mail"`}, {Kind: "Then", Text: "exit code 3"}}},
	}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("expand = %+v, %v; want %+v", got, err, want)
	}
}

func TestExpandFailsAPlaceholderWithNoValue(t *testing.T) {
	f := irFeature{Scenarios: []irScenario{{
		Name:     "o",
		Steps:    []irStep{{Keyword: "Then", Text: "exit code <code>"}},
		Examples: []map[string]string{{"args": "--help"}},
	}}}
	_, err := expand(f)
	if err == nil || err.Error() != `scenario "o" example_1: no example value for <code>` {
		t.Fatalf("expand error = %v", err)
	}
}

func TestSubstituteFillsEveryPlaceholder(t *testing.T) {
	row := map[string]string{"x": "1", "y_2": "two"}
	cases := map[string]string{
		"a <x> b <y_2> <x>": "a 1 b two 1",
		"no placeholders":   "no placeholders",
		"<not-a-name> <x>":  "<not-a-name> 1",
	}
	for text, want := range cases {
		got, err := substitute(text, row)
		if err != nil || got != want {
			t.Errorf("substitute(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
}

func TestPlanFeatureListsEveryUnmatchedExpandedStep(t *testing.T) {
	Register("irtest known", func(*World, string) error { return nil })
	t.Setenv(irEnv, writeIR(t, `{"name":"f","scenarios":[
		{"name":"plain","steps":[{"keyword":"Given","text":"irtest known"},{"keyword":"Then","text":"irtest missing"}],"examples":[]},
		{"name":"o","steps":[{"keyword":"Given","text":"irtest known"},{"keyword":"Then","text":"irtest gone <n>"}],"examples":[{"n":"1"},{"n":"2"}]}]}`))
	_, err := planFeature("../../features/x.feature")
	want := "../../features/x.feature: no step handler for:\n" +
		"  plain: Then irtest missing\n" +
		"  o/example_1: Then irtest gone 1\n" +
		"  o/example_2: Then irtest gone 2"
	if err == nil || err.Error() != want {
		t.Fatalf("planFeature error =\n%v\nwant\n%s", err, want)
	}
}

func TestPlanFeatureReturnsTheRunsWhenEveryStepMatches(t *testing.T) {
	Register("irtest known", func(*World, string) error { return nil })
	t.Setenv(irEnv, writeIR(t, `{"name":"f","scenarios":[{"name":"s","steps":[{"keyword":"Given","text":"irtest known"}],"examples":[]}]}`))
	got, err := planFeature("../../features/x.feature")
	want := []scenarioRun{{Name: "s", Runs: []execution{{Steps: []Step{{Kind: "Given", Text: "irtest known"}}}}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("planFeature = %+v, %v; want %+v", got, err, want)
	}
}

func TestPlanFeatureFailsWithoutItsIR(t *testing.T) {
	t.Setenv(irEnv, filepath.Join(t.TempDir(), "missing.json"))
	if _, err := planFeature("../../features/x.feature"); err == nil {
		t.Fatal("planFeature without an IR file: want an error")
	}
}
