package graph

import (
	"context"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func forbidden(inner string) fakeResp {
	return fakeResp{status: 403, body: []byte(`{"error":{"code":"Forbidden","message":"tenant detail","innerError":{"code":"` + inner + `"}}}`)}
}

func wantAuthNaming(t *testing.T, err error, code string) {
	t.Helper()
	de, ok := err.(*domain.Error)
	if !ok || de.Class != domain.ClassAuth || !strings.HasPrefix(de.Message, code+": ") || de.Hint != "a Teams administrator controls this setting" {
		t.Fatalf("err %#v, want auth naming %s", err, code)
	}
}

func TestTranscriptListRefusedByTheTenant(t *testing.T) {
	refusal := forbidden("GraphAccessToTranscriptsDisabled")
	f := &meetingsFake{list: &refusal}
	c := startMeetingsFake(t, f)
	_, err := c.Transcripts(context.Background(), "mtg-1")
	wantAuthNaming(t, err, "GraphAccessToTranscriptsDisabled")
	if n := len(f.requests()); n != 1 {
		t.Fatalf("%d requests, want 1 and none after the refusal", n)
	}
}

func TestTranscriptContentRefusedWithoutSpeakerNames(t *testing.T) {
	f := &meetingsFake{content: map[string]fakeResp{"t1": forbidden("SpeakerAttributionNotAllowed")}}
	c := startMeetingsFake(t, f)
	_, err := c.TranscriptContent(context.Background(), "mtg-1", "t1")
	wantAuthNaming(t, err, "SpeakerAttributionNotAllowed")
	reqs := f.requests()
	if len(reqs) != 1 || reqs[0].query != "$format=text/vtt" {
		t.Fatalf("requests %#v, want one text/vtt request and no other format", reqs)
	}
}

func TestEventReadRefusalCarriesTheCodes(t *testing.T) {
	f := &meetingsFake{events: map[string]fakeResp{"evt-online": forbidden("Other")}}
	c := startMeetingsFake(t, f)
	_, err := c.EventMeeting(context.Background(), "evt-online")
	wantMeetingsErr(t, err, domain.ClassAuth, "missing consent or session (code: Forbidden, inner: Other)")
}
