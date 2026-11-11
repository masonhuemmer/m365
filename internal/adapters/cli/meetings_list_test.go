package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMeetingsTranscriptsLists(t *testing.T) {
	d, out, errw := testDeps()
	d.Meetings = newCLIMeetings(t, inWindow())
	loginMeetings(t, &d)
	out.Reset()
	if c := Run([]string{"m365", "meetings", "transcripts", "--event", "evt-online"}, d); c != 0 {
		t.Fatalf("exit %d: %s", c, errw.String())
	}
	want := `{"count":1,"items":[{"id":"t1","created":"2030-01-15T15:05:00Z","ended":"2030-01-15T15:29:00Z"}]}` + "\n"
	if out.String() != want {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestMeetingsTranscriptsEmpty(t *testing.T) {
	d, out, errw := testDeps()
	d.Meetings = newCLIMeetings(t)
	loginMeetings(t, &d)
	out.Reset()
	if c := Run([]string{"m365", "meetings", "transcripts", "--join-url", "https://teams.example.test/l/meetup-join/abc"}, d); c != 0 {
		t.Fatalf("exit %d: %s", c, errw.String())
	}
	if out.String() != `{"count":0,"items":[]}`+"\n" {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestMeetingsTranscriptsHuman(t *testing.T) {
	d, out, errw := testDeps()
	d.Meetings = newCLIMeetings(t, inWindow())
	loginMeetings(t, &d)
	out.Reset()
	if c := Run([]string{"m365", "meetings", "transcripts", "--event", "evt-online", "--human"}, d); c != 0 {
		t.Fatalf("exit %d: %s", c, errw.String())
	}
	var got map[string]any
	_ = json.Unmarshal(out.Bytes(), &got)
	want := map[string]any{"count": float64(1), "items": []any{map[string]any{"id": "t1", "created": "2030-01-15T15:05:00Z", "ended": "2030-01-15T15:29:00Z"}}}
	if !strings.HasPrefix(out.String(), "{\n  \"") || !reflect.DeepEqual(got, want) {
		t.Fatalf("human output %q", out.String())
	}
}

func TestMeetingsTranscriptsHelpAndUsage(t *testing.T) {
	d, out, _ := testDeps()
	m := newCLIMeetings(t, inWindow())
	d.Meetings = m
	if c := Run([]string{"m365", "meetings", "transcripts", "--help"}, d); c != 0 || out.String() != meetingsHelpText || m.calls != 0 {
		t.Fatalf("exit %d calls %d out %q", c, m.calls, out.String())
	}
	loginMeetings(t, &d)
	if c := Run([]string{"m365", "meetings", "transcripts", "--event", "e", "--join-url", "https://teams.example.test/x"}, d); c != 3 {
		t.Fatalf("exit %d, want 3", c)
	}
}
