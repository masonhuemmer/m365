package graph

import (
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func TestMapMeetingsError(t *testing.T) {
	const hint = "a Teams administrator controls this setting"
	graphAccess := "GraphAccessToTranscriptsDisabled: Graph access to meeting transcripts is turned off for this tenant"
	speakers := "SpeakerAttributionNotAllowed: speaker attribution is turned off for this tenant"
	cases := []struct {
		name             string
		status           int
		body             string
		class, msg, hint string
	}{
		{"graph access off", 403, `{"error":{"code":"Forbidden","message":"tenant detail","innerError":{"code":"GraphAccessToTranscriptsDisabled"}}}`, domain.ClassAuth, graphAccess, hint},
		{"lower-case innererror", 403, `{"error":{"code":"Forbidden","message":"tenant detail","innererror":{"code":"GraphAccessToTranscriptsDisabled"}}}`, domain.ClassAuth, graphAccess, hint},
		{"speaker attribution off", 403, `{"error":{"code":"Forbidden","message":"tenant detail","innerError":{"code":"SpeakerAttributionNotAllowed"}}}`, domain.ClassAuth, speakers, hint},
		{"other inner code", 403, `{"error":{"code":"Forbidden","message":"tenant detail","innerError":{"code":"Other"}}}`, domain.ClassAuth, "missing consent or session (code: Forbidden, inner: Other)", ""},
		{"code only", 401, `{"error":{"code":"InvalidAuthenticationToken","message":"tenant detail"}}`, domain.ClassAuth, "missing consent or session (code: InvalidAuthenticationToken)", ""},
		{"no body", 403, ``, domain.ClassAuth, "missing consent or session", ""},
		{"not found", 404, `{"error":{"code":"NotFound","message":"tenant detail"}}`, domain.ClassNotFound, "not found", ""},
		{"server error", 500, `{"error":{"code":"InternalServerError","message":"tenant detail"}}`, domain.ClassService, "microsoft 365 service error", ""},
	}
	for _, c := range cases {
		err := mapMeetingsError(c.status, []byte(c.body))
		de, ok := err.(*domain.Error)
		if !ok || de.Class != c.class || de.Message != c.msg || de.Hint != c.hint {
			t.Errorf("%s: err %#v, want %s %q hint %q", c.name, err, c.class, c.msg, c.hint)
			continue
		}
		if strings.Contains(de.Message+de.Hint, "tenant detail") {
			t.Errorf("%s: Graph's message leaked: %q", c.name, de.Message)
		}
	}
}
