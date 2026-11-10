package meetings

import (
	"context"
	"net/url"
	"sort"
	"time"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

// windowSlack widens an event occurrence on both sides: a transcript belongs
// to the occurrence when it started from 1 hour before to 1 hour after it.
const windowSlack = time.Hour

type window struct{ from, to time.Time }

type timedTranscript struct {
	domain.Transcript
	created time.Time
}

// Save writes the chosen transcript's attributed WebVTT to t.Out and returns its summary.
func Save(ctx context.Context, st Store, sess domain.Session, t Target, write func(string, []byte, bool) error) (domain.SavedTranscript, error) {
	mid, items, win, err := meetingTranscripts(ctx, st, sess, t, true)
	if err != nil {
		return domain.SavedTranscript{}, err
	}
	chosen, err := choose(items, win, t.ID)
	if err != nil {
		return domain.SavedTranscript{}, err
	}
	content, err := st.TranscriptContent(ctx, mid, chosen.ID)
	if err != nil {
		return domain.SavedTranscript{}, err
	}
	speakers, err := checkContent(content)
	if err != nil {
		return domain.SavedTranscript{}, err
	}
	if err := write(t.Out, content, t.Overwrite); err != nil {
		return domain.SavedTranscript{}, err
	}
	return domain.SavedTranscript{
		Path: t.Out, ID: chosen.ID, Created: chosen.Created, Ended: chosen.Ended,
		Bytes: len(content), Speakers: speakers,
	}, nil
}

// meetingTranscripts checks consent and the target, finds the meeting, and
// returns all of its transcripts newest first, plus the event window (nil
// for a join URL).
func meetingTranscripts(ctx context.Context, st Store, sess domain.Session, t Target, needOut bool) (string, []timedTranscript, *window, error) {
	if err := auth.RequireMeetings(sess); err != nil {
		return "", nil, nil, err
	}
	if err := checkTarget(t, needOut); err != nil {
		return "", nil, nil, err
	}
	mid, win, err := resolve(ctx, st, sess, t)
	if err != nil {
		return "", nil, nil, err
	}
	raw, err := st.Transcripts(ctx, mid)
	if err != nil {
		return "", nil, nil, err
	}
	items, err := parseTimes(raw)
	if err != nil {
		return "", nil, nil, err
	}
	sortNewest(items)
	return mid, items, win, nil
}

func checkTarget(t Target, needOut bool) error {
	if (t.EventID == "") == (t.JoinURL == "") {
		return domain.Usage("use exactly one of --event or --join-url")
	}
	if t.JoinURL != "" && !httpsURL(t.JoinURL) {
		return domain.Usage("--join-url must be an https URL")
	}
	if needOut && t.Out == "" {
		return domain.Usage("--out is required")
	}
	return nil
}

func httpsURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func resolve(ctx context.Context, st Store, sess domain.Session, t Target) (string, *window, error) {
	if t.JoinURL != "" {
		mid, err := st.FindMeeting(ctx, t.JoinURL)
		return mid, nil, err
	}
	if err := auth.RequireCalendar(sess); err != nil {
		return "", nil, err
	}
	occ, err := st.EventMeeting(ctx, t.EventID)
	if err != nil {
		return "", nil, err
	}
	mid, err := st.FindMeeting(ctx, occ.JoinURL)
	return mid, &window{from: occ.Start.Add(-windowSlack), to: occ.End.Add(windowSlack)}, err
}

func parseTimes(raw []domain.Transcript) ([]timedTranscript, error) {
	items := make([]timedTranscript, 0, len(raw))
	for _, it := range raw {
		created, err := time.Parse(time.RFC3339Nano, it.Created)
		if err != nil {
			return nil, domain.Service("invalid transcript")
		}
		items = append(items, timedTranscript{Transcript: it, created: created})
	}
	return items, nil
}

func sortNewest(items []timedTranscript) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].created.Equal(items[j].created) {
			return items[i].created.After(items[j].created)
		}
		return items[i].ID < items[j].ID
	})
}

func inScope(items []timedTranscript, win *window) []timedTranscript {
	if win == nil {
		return items
	}
	scoped := make([]timedTranscript, 0, len(items))
	for _, it := range items {
		if !it.created.Before(win.from) && !it.created.After(win.to) {
			scoped = append(scoped, it)
		}
	}
	return scoped
}

// choose picks the transcript named by id from the whole meeting, or else
// the single one in scope.
func choose(items []timedTranscript, win *window, id string) (timedTranscript, error) {
	if id != "" {
		for _, it := range items {
			if it.ID == id {
				return it, nil
			}
		}
		return timedTranscript{}, domain.NotFound("no transcript with that id")
	}
	scoped := inScope(items, win)
	switch len(scoped) {
	case 0:
		return timedTranscript{}, domain.NotFound("no transcript in scope; transcription may have been off")
	case 1:
		return scoped[0], nil
	}
	return timedTranscript{}, &domain.Error{Class: domain.ClassUsage, Message: "several transcripts in scope", Hint: "run m365 meetings transcripts, then pass --id"}
}

func checkContent(b []byte) ([]string, error) {
	if !domain.IsWebVTT(b) {
		return nil, domain.Service("transcript is not WebVTT")
	}
	names := domain.VoiceNames(b)
	if len(names) == 0 {
		return nil, domain.Service("transcript has no speaker names")
	}
	return names, nil
}

// List returns the meeting's in-scope transcripts, newest first.
func List(ctx context.Context, st Store, sess domain.Session, t Target) (domain.TranscriptList, error) {
	_, items, win, err := meetingTranscripts(ctx, st, sess, t, false)
	if err != nil {
		return domain.TranscriptList{}, err
	}
	scoped := inScope(items, win)
	list := domain.TranscriptList{Count: len(scoped), Items: make([]domain.Transcript, 0, len(scoped))}
	for _, it := range scoped {
		list.Items = append(list.Items, it.Transcript)
	}
	return list, nil
}
