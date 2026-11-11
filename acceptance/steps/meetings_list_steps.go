package steps

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/masonhuemmer/m365/acceptance/runtime"
)

func init() {
	runtime.Register("a meetings list fake", givenMeetingsListFake)
	runtime.Register("I list transcripts by", listTranscripts)
	runtime.Register("the listed", listedStep)
}

// givenMeetingsListFake: "... with transcripts across two occurrences" (one
// a week before the event, two during it) or "... with no transcripts".
func givenMeetingsListFake(w *runtime.World, text string) error {
	r := newMeetingsRun(w)
	r.fx.pages = [][]map[string]string{{}}
	if strings.HasSuffix(text, "across two occurrences") {
		r.fx.pages = [][]map[string]string{
			{meetingsTranscript("t-prev", "2030-01-08T15:05:00Z"), meetingsTranscript("t-in-a", "2030-01-15T15:05:00Z")},
			{meetingsTranscript("t-in-b", "2030-01-15T15:20:00Z")},
		}
	}
	return nil
}

// listTranscripts: "I list transcripts by event" or "... by join URL".
func listTranscripts(w *runtime.World, text string) error {
	args := []string{"meetings", "transcripts", "--event", "evt-online"}
	if strings.HasSuffix(text, "by join URL") {
		args = []string{"meetings", "transcripts", "--join-url", meetingsJoinURL}
	}
	meetingsRunOf(w).run(w, args...)
	return nil
}

// listedStep: "the listed transcripts are ID ..." (in order) or "the listed count is N".
func listedStep(w *runtime.World, text string) error {
	var got struct {
		Count int `json:"count"`
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(w.Out), &got); err != nil {
		w.T.Fatalf("stdout is not a transcript list: %q", w.Out)
	}
	if i := strings.Index(text, "count is "); i >= 0 {
		if want, _ := strconv.Atoi(text[i+len("count is "):]); got.Count != want {
			w.T.Fatalf("count %d, want %d", got.Count, want)
		}
		return nil
	}
	ids := []string{}
	for _, it := range got.Items {
		ids = append(ids, it.ID)
	}
	if want := strings.Fields(strings.TrimPrefix(text, "the listed transcripts are ")); !reflect.DeepEqual(ids, want) {
		w.T.Fatalf("listed %q, want %q", ids, want)
	}
	return nil
}
