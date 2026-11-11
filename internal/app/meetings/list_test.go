package meetings

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func ids(l domain.TranscriptList) []string {
	out := []string{}
	for _, it := range l.Items {
		out = append(out, it.ID)
	}
	return out
}

func listFixture(t *testing.T) *fakeStore {
	return newFakeStore(t,
		tr("start", "2030-01-15T14:00:00Z"), tr("mid", "2030-01-15T15:05:00Z"),
		tr("early", "2030-01-15T13:59:59Z"), tr("late", "2030-01-15T16:30:01Z"))
}

func TestListByEventKeepsTheWindow(t *testing.T) {
	st := listFixture(t)
	got, err := List(context.Background(), st, fullSess, Target{EventID: "evt"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 2 || !reflect.DeepEqual(ids(got), []string{"mid", "start"}) {
		t.Fatalf("got %d %q", got.Count, ids(got))
	}
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1")
}

func TestListByJoinURLKeepsAll(t *testing.T) {
	st := listFixture(t)
	got, err := List(context.Background(), st, fullSess, Target{JoinURL: testJoinURL})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 4 || !reflect.DeepEqual(ids(got), []string{"late", "mid", "start", "early"}) {
		t.Fatalf("got %d %q", got.Count, ids(got))
	}
	st.wantCalls(t, "FindMeeting "+testJoinURL, "Transcripts mtg-1")
}

func TestListTiesSortByID(t *testing.T) {
	st := newFakeStore(t, tr("b", "2030-01-15T15:05:00Z"), tr("a", "2030-01-15T15:05:00Z"))
	got, err := List(context.Background(), st, fullSess, Target{JoinURL: testJoinURL})
	if err != nil || !reflect.DeepEqual(ids(got), []string{"a", "b"}) {
		t.Fatalf("got %q err %v", ids(got), err)
	}
}

func TestListEmpty(t *testing.T) {
	got, err := List(context.Background(), newFakeStore(t), fullSess, Target{EventID: "evt"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(got)
	if string(b) != `{"count":0,"items":[]}` {
		t.Fatalf("json %s", b)
	}
}

func TestListChecksBeforeAnyCall(t *testing.T) {
	noMeetings := fullSess
	noMeetings.MeetingsConsented = false
	noCalendar := fullSess
	noCalendar.CalendarConsented = false
	cases := []struct {
		name             string
		sess             domain.Session
		t                Target
		class, msg, hint string
	}{
		{"no meetings consent", noMeetings, Target{EventID: "evt"}, domain.ClassAuth, "missing meetings consent", "run m365 auth login --meetings"},
		{"both selectors", fullSess, Target{EventID: "evt", JoinURL: testJoinURL}, domain.ClassUsage, "use exactly one of --event or --join-url", ""},
		{"no selector", fullSess, Target{}, domain.ClassUsage, "use exactly one of --event or --join-url", ""},
		{"http join url", fullSess, Target{JoinURL: "http://teams.example.test/x"}, domain.ClassUsage, "--join-url must be an https URL", ""},
		{"event without calendar consent", noCalendar, Target{EventID: "evt"}, domain.ClassAuth, "missing calendar consent", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := newFakeStore(t)
			_, err := List(context.Background(), st, c.sess, c.t)
			wantDomainErr(t, err, c.class, c.msg, c.hint)
			st.wantCalls(t)
		})
	}
}
