package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/adapters/graph"
)

// refusingGraph is a TLS fake Graph whose list or content answers 403 with
// the given tenant code; everything before that step succeeds.
func refusingGraph(t *testing.T, listCode, contentCode string) *graph.HTTPMeetings {
	t.Helper()
	forbidden := func(w http.ResponseWriter, code string) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"Forbidden","innerError":{"code":"` + code + `"}}}`))
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch p := r.URL.Path; {
		case strings.HasPrefix(p, "/me/events/"):
			_, _ = w.Write([]byte(`{"isOnlineMeeting":true,"onlineMeeting":{"joinUrl":"https://teams.example.test/l/meetup-join/abc"},` +
				`"start":{"dateTime":"2030-01-15T15:00:00"},"end":{"dateTime":"2030-01-15T15:30:00"}}`))
		case p == "/me/onlineMeetings":
			_, _ = w.Write([]byte(`{"value":[{"id":"mtg-1"}]}`))
		case strings.HasSuffix(p, "/transcripts") && listCode != "":
			forbidden(w, listCode)
		case strings.HasSuffix(p, "/transcripts"):
			_, _ = w.Write([]byte(`{"value":[{"id":"t1","createdDateTime":"2030-01-15T15:05:00Z","endDateTime":"2030-01-15T15:29:00Z"}]}`))
		default:
			forbidden(w, contentCode)
		}
	}))
	t.Cleanup(srv.Close)
	return &graph.HTTPMeetings{HTTPClient: &graph.HTTPClient{Base: srv.URL, Token: "synthetic-token", Client: srv.Client()}}
}

func TestMeetingsTranscriptRefusedWithoutSpeakerNames(t *testing.T) {
	d, _, errw := testDeps()
	d.Meetings = refusingGraph(t, "", "SpeakerAttributionNotAllowed")
	loginMeetings(t, &d)
	p := filepath.Join(t.TempDir(), "m.vtt")
	if code := Run([]string{"m365", "meetings", "transcript", "--event", "evt-online", "--out", p}, d); code != 4 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	want := `{"class":"auth","message":"SpeakerAttributionNotAllowed: speaker attribution is turned off for this tenant","hint":"a Teams administrator controls this setting"}` + "\n"
	if errw.String() != want {
		t.Fatalf("stderr %q", errw.String())
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("file written: %v", err)
	}
}

func TestMeetingsTranscriptsRefusedByTheTenant(t *testing.T) {
	d, _, errw := testDeps()
	d.Meetings = refusingGraph(t, "GraphAccessToTranscriptsDisabled", "")
	loginMeetings(t, &d)
	if code := Run([]string{"m365", "meetings", "transcripts", "--event", "evt-online"}, d); code != 4 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	want := `{"class":"auth","message":"GraphAccessToTranscriptsDisabled: Graph access to meeting transcripts is turned off for this tenant","hint":"a Teams administrator controls this setting"}` + "\n"
	if errw.String() != want {
		t.Fatalf("stderr %q", errw.String())
	}
}
