package graph

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

const tenantSettingHint = "a Teams administrator controls this setting"

// tenantRefusals are the inner codes of the two tenant settings that block
// transcript reads; Graph's top-level code for both is just Forbidden.
var tenantRefusals = map[string]string{
	"GraphAccessToTranscriptsDisabled": "Graph access to meeting transcripts is turned off for this tenant",
	"SpeakerAttributionNotAllowed":     "speaker attribution is turned off for this tenant",
}

// mapMeetingsError maps a failed meetings request to a CLI error. It names
// Graph's codes, never Graph's free-text message.
func mapMeetingsError(status int, body []byte) error {
	var ge struct {
		Error struct {
			Code  string `json:"code"`
			Inner struct {
				Code string `json:"code"`
			} `json:"innerError"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &ge)
	code, inner := ge.Error.Code, ge.Error.Inner.Code
	if text, ok := tenantRefusals[inner]; ok {
		return &domain.Error{Class: domain.ClassAuth, Message: inner + ": " + text, Hint: tenantSettingHint}
	}
	if status != http.StatusUnauthorized && status != http.StatusForbidden {
		return MapGraphError(status, body)
	}
	return domain.Auth("missing consent or session" + codeSuffix(code, inner))
}

func codeSuffix(code, inner string) string {
	var parts []string
	if code != "" {
		parts = append(parts, "code: "+code)
	}
	if inner != "" {
		parts = append(parts, "inner: "+inner)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
