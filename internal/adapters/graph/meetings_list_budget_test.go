package graph

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/meetings"
)

func TestTranscriptListRequestBudget(t *testing.T) {
	cases := []struct {
		name   string
		target meetings.Target
		want   int
	}{
		{"event", meetings.Target{EventID: "evt-online"}, 12},
		{"join url", meetings.Target{JoinURL: fakeJoinURL}, 11},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, store := budgetFake(t, fixtureVTT(t))
			got, err := meetings.List(context.Background(), store, budgetSession, c.target)
			if err != nil {
				t.Fatal(err)
			}
			if got.Count != 1 {
				t.Fatalf("count %d, want 1", got.Count)
			}
			if n := len(f.requests()); n != c.want {
				t.Fatalf("%d requests, want %d", n, c.want)
			}
		})
	}
}
