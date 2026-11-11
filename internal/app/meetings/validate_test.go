package meetings

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func TestListRejectsAnInvalidTranscript(t *testing.T) {
	cases := []struct {
		name string
		item domain.Transcript
	}{
		{"empty id", domain.Transcript{ID: "", Created: "2030-01-15T15:05:00Z", Ended: "2030-01-15T15:29:00Z"}},
		{"bad end time", domain.Transcript{ID: "t1", Created: "2030-01-15T15:05:00Z", Ended: "yesterday"}},
	}
	for _, c := range cases {
		_, err := List(context.Background(), newFakeStore(t, c.item), fullSess, Target{JoinURL: testJoinURL})
		wantDomainErr(t, err, domain.ClassService, "invalid transcript", "")
	}
}

func TestListKeepsARunningTranscript(t *testing.T) {
	running := domain.Transcript{ID: "t1", Created: "2030-01-15T15:05:00Z"}
	got, err := List(context.Background(), newFakeStore(t, running), fullSess, Target{JoinURL: testJoinURL})
	if err != nil || got.Count != 1 || got.Items[0].Ended != "" {
		t.Fatalf("got %#v err %v", got, err)
	}
}
