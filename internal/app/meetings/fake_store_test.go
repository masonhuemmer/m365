package meetings

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/masonhuemmer/m365/internal/domain"
)

const testJoinURL = "https://teams.example.test/l/meetup-join/abc"

var (
	occStart = time.Date(2030, 1, 15, 15, 0, 0, 0, time.UTC)
	occEnd   = time.Date(2030, 1, 15, 15, 30, 0, 0, time.UTC)
	fullSess = domain.Session{SignedIn: true, SessionUsable: true, CalendarConsented: true, MeetingsConsented: true}
)

// fakeStore returns fixtures and records every call in order.
type fakeStore struct {
	items   []domain.Transcript
	content []byte
	errs    map[string]error
	calls   []string
}

func newFakeStore(t *testing.T, items ...domain.Transcript) *fakeStore {
	t.Helper()
	b, err := os.ReadFile("../../domain/testdata/meeting.vtt")
	if err != nil {
		t.Fatal(err)
	}
	return &fakeStore{items: items, content: b, errs: map[string]error{}}
}

func (f *fakeStore) EventMeeting(_ context.Context, id string) (Occurrence, error) {
	f.calls = append(f.calls, "EventMeeting "+id)
	return Occurrence{JoinURL: testJoinURL, Start: occStart, End: occEnd}, f.errs["EventMeeting"]
}

func (f *fakeStore) FindMeeting(_ context.Context, join string) (string, error) {
	f.calls = append(f.calls, "FindMeeting "+join)
	return "mtg-1", f.errs["FindMeeting"]
}

func (f *fakeStore) Transcripts(_ context.Context, mid string) ([]domain.Transcript, error) {
	f.calls = append(f.calls, "Transcripts "+mid)
	return f.items, f.errs["Transcripts"]
}

func (f *fakeStore) TranscriptContent(_ context.Context, mid, tid string) ([]byte, error) {
	f.calls = append(f.calls, "TranscriptContent "+mid+" "+tid)
	return f.content, f.errs["TranscriptContent"]
}

func (f *fakeStore) wantCalls(t *testing.T, want ...string) {
	t.Helper()
	if len(want) == 0 && len(f.calls) == 0 {
		return
	}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("calls %q, want %q", f.calls, want)
	}
}

// writeRec records the one write a save may make.
type writeRec struct {
	n         int
	path      string
	data      []byte
	overwrite bool
	err       error
}

func (w *writeRec) fn(p string, b []byte, o bool) error {
	w.n++
	w.path, w.data, w.overwrite = p, b, o
	return w.err
}

func wantDomainErr(t *testing.T, err error, class, msg, hint string) {
	t.Helper()
	de, ok := err.(*domain.Error)
	if !ok || de.Class != class || de.Message != msg || de.Hint != hint {
		t.Fatalf("err %#v, want %s %q hint %q", err, class, msg, hint)
	}
}

func tr(id, created string) domain.Transcript {
	return domain.Transcript{ID: id, Created: created, Ended: "2030-01-15T15:29:00Z"}
}
