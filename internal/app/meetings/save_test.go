package meetings

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func TestSaveChecksBeforeAnyCall(t *testing.T) {
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
		{"no meetings consent", noMeetings, Target{EventID: "evt", Out: "o"}, domain.ClassAuth, "missing meetings consent", "run m365 auth login --meetings"},
		{"both selectors", fullSess, Target{EventID: "evt", JoinURL: testJoinURL, Out: "o"}, domain.ClassUsage, "use exactly one of --event or --join-url", ""},
		{"no selector", fullSess, Target{Out: "o"}, domain.ClassUsage, "use exactly one of --event or --join-url", ""},
		{"http join url", fullSess, Target{JoinURL: "http://teams.example.test/x", Out: "o"}, domain.ClassUsage, "--join-url must be an https URL", ""},
		{"no scheme", fullSess, Target{JoinURL: "teams.example.test/x", Out: "o"}, domain.ClassUsage, "--join-url must be an https URL", ""},
		{"no host", fullSess, Target{JoinURL: "https:///x", Out: "o"}, domain.ClassUsage, "--join-url must be an https URL", ""},
		{"no out", fullSess, Target{EventID: "evt"}, domain.ClassUsage, "--out is required", ""},
		{"event without calendar consent", noCalendar, Target{EventID: "evt", Out: "o"}, domain.ClassAuth, "missing calendar consent", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, w := newFakeStore(t), &writeRec{}
			_, err := Save(context.Background(), st, c.sess, c.t, w.fn)
			wantDomainErr(t, err, c.class, c.msg, c.hint)
			st.wantCalls(t)
			if w.n != 0 {
				t.Fatal("wrote a file")
			}
		})
	}
}

func TestSaveByEvent(t *testing.T) {
	st, w := newFakeStore(t, tr("t1", "2030-01-15T15:05:00Z")), &writeRec{}
	got, err := Save(context.Background(), st, fullSess, Target{EventID: "evt", Out: "dir/m.vtt", Overwrite: true}, w.fn)
	if err != nil {
		t.Fatal(err)
	}
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1", "TranscriptContent mtg-1 t1")
	if w.n != 1 || w.path != "dir/m.vtt" || !bytes.Equal(w.data, st.content) || !w.overwrite {
		t.Fatalf("write %d %q overwrite=%v same=%v", w.n, w.path, w.overwrite, bytes.Equal(w.data, st.content))
	}
	want := domain.SavedTranscript{
		Path: "dir/m.vtt", ID: "t1", Created: "2030-01-15T15:05:00Z", Ended: "2030-01-15T15:29:00Z",
		Bytes: len(st.content), Speakers: []string{"Avery Example", "Blake Sample"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSaveByJoinURL(t *testing.T) {
	st, w := newFakeStore(t, tr("t1", "2030-01-15T15:05:00Z")), &writeRec{}
	got, err := Save(context.Background(), st, fullSess, Target{JoinURL: testJoinURL, Out: "m.vtt"}, w.fn)
	if err != nil {
		t.Fatal(err)
	}
	st.wantCalls(t, "FindMeeting "+testJoinURL, "Transcripts mtg-1", "TranscriptContent mtg-1 t1")
	if got.ID != "t1" || w.n != 1 || w.overwrite {
		t.Fatalf("got %#v write %d overwrite=%v", got, w.n, w.overwrite)
	}
}
