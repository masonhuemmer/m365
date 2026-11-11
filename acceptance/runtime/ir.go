package runtime

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

// irPathFor returns the IR file for a feature path, or the file named by
// M365_ACCEPTANCE_IR.
func irPathFor(featurePath string) string {
	return ""
}

// loadIR reads a parser IR file.
func loadIR(path string) (irFeature, error) {
	return irFeature{}, nil
}

// expand turns a feature's IR into scenario runs.
func expand(f irFeature) ([]scenarioRun, error) {
	return nil, nil
}

// substitute fills each <name> in text from row.
func substitute(text string, row map[string]string) (string, error) {
	return "", nil
}

// planFeature loads and expands a feature's IR and checks that every
// expanded step has a handler, reporting all that do not.
func planFeature(featurePath string) ([]scenarioRun, error) {
	return nil, nil
}
