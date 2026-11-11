package steps

import (
	"strings"

	"github.com/masonhuemmer/m365/acceptance/runtime"
)

func init() {
	runtime.Register("a meetings tenant with", givenMeetingsTenant)
	runtime.Register("the error names", errorNames)
}

func tenantRefusal(code string) *meetingsAnswer {
	return &meetingsAnswer{status: 403, body: []byte(`{"error":{"code":"Forbidden","innerError":{"code":"` + code + `"}}}`)}
}

// givenMeetingsTenant: "a meetings tenant with Graph access to transcripts
// turned off" or "... with speaker attribution turned off".
func givenMeetingsTenant(w *runtime.World, text string) error {
	r := newMeetingsRun(w)
	r.fx.pages = [][]map[string]string{{meetingsTranscript("t1", "2030-01-15T15:05:00Z")}}
	r.fx.content["t1"] = *tenantRefusal("SpeakerAttributionNotAllowed")
	if strings.Contains(text, "Graph access") {
		r.fx.list = tenantRefusal("GraphAccessToTranscriptsDisabled")
	}
	return nil
}

// errorNames: "the error names CODE" checks the stderr JSON names the code
// and the Teams administrator hint.
func errorNames(w *runtime.World, text string) error {
	code := strings.TrimSpace(strings.TrimPrefix(text, "the error names"))
	if !strings.Contains(w.Err, `"message":"`+code+": ") || !strings.Contains(w.Err, "a Teams administrator controls this setting") {
		w.T.Fatalf("stderr %q does not name %s", w.Err, code)
	}
	return nil
}
