package graph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/adapters/fs"
	"github.com/masonhuemmer/m365/internal/app/meetings"
	"github.com/masonhuemmer/m365/internal/domain"
)

var budgetSession = domain.Session{SignedIn: true, SessionUsable: true, CalendarConsented: true, MeetingsConsented: true}

// budgetFake holds the online event, one meeting, 10 list pages with the one
// in-window transcript on the last page, and that transcript's content.
func budgetFake(t *testing.T, content []byte) (*meetingsFake, *HTTPMeetings) {
	t.Helper()
	f := &meetingsFake{
		events:   map[string]fakeResp{"evt-online": onlineEvent()},
		meetings: map[string]string{fakeJoinURL: "mtg-1"},
		pages:    emptyPages(10, transcriptJSON("t1", "2030-01-15T15:05:00Z")),
		content:  map[string]fakeResp{"t1": {body: content}},
	}
	return f, startMeetingsFake(t, f)
}

func fixtureVTT(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../domain/testdata/meeting.vtt")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTranscriptSaveRequestBudget(t *testing.T) {
	cases := []struct {
		name   string
		target meetings.Target
		want   int
	}{
		{"event", meetings.Target{EventID: "evt-online"}, 13},
		{"join url", meetings.Target{JoinURL: fakeJoinURL}, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, store := budgetFake(t, fixtureVTT(t))
			c.target.Out = filepath.Join(t.TempDir(), "m.vtt")
			if _, err := meetings.Save(context.Background(), store, budgetSession, c.target, fs.WriteFile); err != nil {
				t.Fatal(err)
			}
			if n := len(f.requests()); n != c.want {
				t.Fatalf("%d requests, want %d", n, c.want)
			}
		})
	}
}

func TestTranscriptSaveStopsAtTheContentCap(t *testing.T) {
	big := []byte("WEBVTT\n" + strings.Repeat("x", 10<<20+2-7))
	f, store := budgetFake(t, big)
	out := filepath.Join(t.TempDir(), "m.vtt")
	_, err := meetings.Save(context.Background(), store, budgetSession, meetings.Target{EventID: "evt-online", Out: out}, fs.WriteFile)
	wantMeetingsErr(t, err, domain.ClassService, "transcript is larger than the 10 MiB limit")
	if n := len(f.requests()); n != 13 {
		t.Fatalf("%d requests, want 13 and none after the failure", n)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("file written: %v", err)
	}
}
