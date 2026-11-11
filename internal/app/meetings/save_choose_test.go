package meetings

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func eventTarget() Target { return Target{EventID: "evt", Out: "m.vtt"} }

func TestSaveWindowEdgesOutside(t *testing.T) {
	st, w := newFakeStore(t, tr("early", "2030-01-15T13:59:59Z"), tr("late", "2030-01-15T16:30:01Z")), &writeRec{}
	_, err := Save(context.Background(), st, fullSess, eventTarget(), w.fn)
	wantDomainErr(t, err, domain.ClassNotFound, "no transcript in scope; transcription may have been off", "")
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1")
	if w.n != 0 {
		t.Fatal("wrote a file")
	}
}

func TestSaveWindowEdgesInsideAreSeveral(t *testing.T) {
	st, w := newFakeStore(t, tr("start", "2030-01-15T14:00:00Z"), tr("end", "2030-01-15T16:30:00Z")), &writeRec{}
	_, err := Save(context.Background(), st, fullSess, eventTarget(), w.fn)
	wantDomainErr(t, err, domain.ClassUsage, "several transcripts in scope", "run m365 meetings transcripts, then pass --id")
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1")
}

func TestSaveIDOverridesTheWindow(t *testing.T) {
	st, w := newFakeStore(t, tr("early", "2030-01-15T13:59:59Z"), tr("t1", "2030-01-15T15:05:00Z")), &writeRec{}
	tg := eventTarget()
	tg.ID = "early"
	got, err := Save(context.Background(), st, fullSess, tg, w.fn)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "early" || got.Created != "2030-01-15T13:59:59Z" {
		t.Fatalf("got %#v", got)
	}
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1", "TranscriptContent mtg-1 early")
}

func TestSaveUnknownID(t *testing.T) {
	st, w := newFakeStore(t, tr("t1", "2030-01-15T15:05:00Z")), &writeRec{}
	tg := eventTarget()
	tg.ID = "t9"
	_, err := Save(context.Background(), st, fullSess, tg, w.fn)
	wantDomainErr(t, err, domain.ClassNotFound, "no transcript with that id", "")
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1")
}

func TestSaveRefusesContent(t *testing.T) {
	cases := []struct{ name, content, msg string }{
		{"not webvtt", `{"value":[]}`, "transcript is not WebVTT"},
		{"no speaker names", "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nno speaker\n", "transcript has no speaker names"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, w := newFakeStore(t, tr("t1", "2030-01-15T15:05:00Z")), &writeRec{}
			st.content = []byte(c.content)
			_, err := Save(context.Background(), st, fullSess, eventTarget(), w.fn)
			wantDomainErr(t, err, domain.ClassService, c.msg, "")
			if w.n != 0 {
				t.Fatal("wrote a file")
			}
		})
	}
}

func TestSaveReturnsTheWriteError(t *testing.T) {
	st := newFakeStore(t, tr("t1", "2030-01-15T15:05:00Z"))
	w := &writeRec{err: domain.Usage("destination exists; pass --overwrite")}
	_, err := Save(context.Background(), st, fullSess, eventTarget(), w.fn)
	wantDomainErr(t, err, domain.ClassUsage, "destination exists; pass --overwrite", "")
}

func TestSaveRejectsAnUnparseableTime(t *testing.T) {
	st, w := newFakeStore(t, tr("t1", "yesterday")), &writeRec{}
	_, err := Save(context.Background(), st, fullSess, eventTarget(), w.fn)
	wantDomainErr(t, err, domain.ClassService, "invalid transcript", "")
}

func TestSaveStopsAtTheFirstStoreError(t *testing.T) {
	st, w := newFakeStore(t), &writeRec{}
	st.errs["Transcripts"] = domain.Service("x")
	_, err := Save(context.Background(), st, fullSess, eventTarget(), w.fn)
	wantDomainErr(t, err, domain.ClassService, "x", "")
	st.wantCalls(t, "EventMeeting evt", "FindMeeting "+testJoinURL, "Transcripts mtg-1")
}
