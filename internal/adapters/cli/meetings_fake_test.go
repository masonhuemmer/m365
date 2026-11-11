package cli

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/app/meetings"
	"github.com/masonhuemmer/m365/internal/domain"
)

const meetingsHelpText = `m365 meetings — Teams meeting transcripts you were invited to

Verbs: transcripts (list), transcript (save one)
Name the meeting: --event EVENT_ID (from calendar list) or --join-url URL (https). Exactly one.
--event lists transcripts that started from 1 hour before the event to 1 hour after it ended.
--join-url lists all of the meeting's transcripts.
Save: --out FILE (required) --id TRANSCRIPT_ID (when several) --overwrite
The file is the attributed WebVTT, byte for byte, owner-only. No transcript text on stdout.
Limits: 10 pages of transcripts; 10 MiB of content.
Needs: m365 auth login --meetings (admin-consented scopes); --event also needs calendar consent.
Tenant codes (exit 4): GraphAccessToTranscriptsDisabled, SpeakerAttributionNotAllowed.
Output: JSON (default) or --human.
`

// cliMeetings is a fake meetings store: one online occurrence, one meeting
// and the given transcripts, all with the synthetic fixture as content.
type cliMeetings struct {
	items   []domain.Transcript
	content []byte
	calls   int
}

func newCLIMeetings(t *testing.T, items ...domain.Transcript) *cliMeetings {
	t.Helper()
	b, err := os.ReadFile("../../domain/testdata/meeting.vtt")
	if err != nil {
		t.Fatal(err)
	}
	return &cliMeetings{items: items, content: b}
}

func (m *cliMeetings) EventMeeting(context.Context, string) (meetings.Occurrence, error) {
	m.calls++
	return meetings.Occurrence{
		JoinURL: "https://teams.example.test/l/meetup-join/abc",
		Start:   time.Date(2030, 1, 15, 15, 0, 0, 0, time.UTC),
		End:     time.Date(2030, 1, 15, 15, 30, 0, 0, time.UTC),
	}, nil
}

func (m *cliMeetings) FindMeeting(context.Context, string) (string, error) {
	m.calls++
	return "mtg-1", nil
}

func (m *cliMeetings) Transcripts(context.Context, string) ([]domain.Transcript, error) {
	m.calls++
	return m.items, nil
}

func (m *cliMeetings) TranscriptContent(context.Context, string, string) ([]byte, error) {
	m.calls++
	return m.content, nil
}

func inWindow() domain.Transcript {
	return domain.Transcript{ID: "t1", Created: "2030-01-15T15:05:00Z", Ended: "2030-01-15T15:29:00Z"}
}

// loginMeetings signs in with all five consents.
func loginMeetings(t *testing.T, d *Deps) {
	t.Helper()
	d.Login = func(context.Context) (auth.Blob, error) {
		return auth.Blob{Account: "user@example.com", Mail: true, Teams: true, Calendar: true, Files: true, Meetings: true, Usable: true}, nil
	}
	if c := Run([]string{"m365", "auth", "login"}, *d); c != 0 {
		t.Fatal(c)
	}
}
