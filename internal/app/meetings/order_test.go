package meetings

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// TestSaveReportsTheFirstFailingCheck breaks two checks at once and expects
// the earlier one, in command-catalog order (rows 1 to 6).
func TestSaveReportsTheFirstFailingCheck(t *testing.T) {
	noMeetings := fullSess
	noMeetings.MeetingsConsented = false
	noCalendar := fullSess
	noCalendar.CalendarConsented = false
	cases := []struct {
		name       string
		sess       domain.Session
		t          Target
		class, msg string
	}{
		{"signed out before consent", domain.Session{}, Target{EventID: "evt", JoinURL: testJoinURL}, domain.ClassAuth, "not signed in"},
		{"consent before selectors", noMeetings, Target{EventID: "evt", JoinURL: testJoinURL}, domain.ClassAuth, "missing meetings consent"},
		{"selectors before out", fullSess, Target{EventID: "evt", JoinURL: testJoinURL}, domain.ClassUsage, "use exactly one of --event or --join-url"},
		{"https before out", fullSess, Target{JoinURL: "http://teams.example.test/x"}, domain.ClassUsage, "--join-url must be an https URL"},
		{"out before calendar consent", noCalendar, Target{EventID: "evt"}, domain.ClassUsage, "--out is required"},
	}
	for _, c := range cases {
		st := newFakeStore(t)
		_, err := Save(context.Background(), st, c.sess, c.t, (&writeRec{}).fn)
		de, ok := err.(*domain.Error)
		if !ok || de.Class != c.class || de.Message != c.msg {
			t.Errorf("%s: err %#v, want %s %q", c.name, err, c.class, c.msg)
		}
		st.wantCalls(t)
	}
}
