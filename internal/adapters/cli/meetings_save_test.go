package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func savedSummary(t *testing.T, path string, size int) map[string]any {
	t.Helper()
	return map[string]any{
		"path": path, "id": "t1", "created": "2030-01-15T15:05:00Z", "ended": "2030-01-15T15:29:00Z",
		"bytes": float64(size), "speakers": []any{"Avery Example", "Blake Sample"},
	}
}

func TestMeetingsTranscriptSaves(t *testing.T) {
	d, out, errw := testDeps()
	m := newCLIMeetings(t, inWindow())
	d.Meetings = m
	loginMeetings(t, &d)
	out.Reset()
	p := filepath.Join(t.TempDir(), "m.vtt")
	if c := Run([]string{"m365", "meetings", "transcript", "--event", "evt-online", "--out", p}, d); c != 0 {
		t.Fatalf("exit %d: %s", c, errw.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err, out.String())
	}
	if want := savedSummary(t, p, len(m.content)); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != string(m.content) {
		t.Fatalf("file differs: %v", err)
	}
	if strings.Contains(out.String()+errw.String(), "Opening the review.") {
		t.Fatal("transcript text printed")
	}
}

func TestMeetingsTranscriptHuman(t *testing.T) {
	d, out, errw := testDeps()
	m := newCLIMeetings(t, inWindow())
	d.Meetings = m
	loginMeetings(t, &d)
	out.Reset()
	p := filepath.Join(t.TempDir(), "m.vtt")
	if c := Run([]string{"m365", "meetings", "transcript", "--event", "evt-online", "--out", p, "--human"}, d); c != 0 {
		t.Fatalf("exit %d: %s", c, errw.String())
	}
	want, _ := json.MarshalIndent(savedSummary(t, p, len(m.content)), "", "  ")
	var gotObj, wantObj map[string]any
	_ = json.Unmarshal(out.Bytes(), &gotObj)
	_ = json.Unmarshal(want, &wantObj)
	if !strings.HasPrefix(out.String(), "{\n  \"") || !reflect.DeepEqual(gotObj, wantObj) {
		t.Fatalf("human output %q", out.String())
	}
}

func TestMeetingsTranscriptExitCodes(t *testing.T) {
	d, out, errw := testDeps()
	d.Meetings = newCLIMeetings(t, inWindow())
	loginMeetings(t, &d)
	dir := t.TempDir()
	cases := []struct {
		args []string
		code int
	}{
		{[]string{"--event", "evt-online", "--join-url", "https://teams.example.test/x", "--out", filepath.Join(dir, "a.vtt")}, 3},
		{[]string{"--event", "evt-online", "--id", "t9", "--out", filepath.Join(dir, "b.vtt")}, 6},
	}
	for _, c := range cases {
		if got := Run(append([]string{"m365", "meetings", "transcript"}, c.args...), d); got != c.code {
			t.Errorf("%v: exit %d, want %d (%s)", c.args, got, c.code, errw.String())
		}
	}
	p := filepath.Join(dir, "kept.vtt")
	if err := os.WriteFile(p, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	errw.Reset()
	out.Reset()
	if got := Run([]string{"m365", "meetings", "transcript", "--event", "evt-online", "--out", p}, d); got != 3 {
		t.Fatalf("exit %d", got)
	}
	if errw.String() != `{"class":"usage","message":"destination exists; pass --overwrite"}`+"\n" {
		t.Fatalf("stderr %q", errw.String())
	}
}

func TestMeetingsHelp(t *testing.T) {
	for _, args := range [][]string{{"meetings"}, {"meetings", "--help"}, {"meetings", "transcript", "--help"}} {
		d, out, _ := testDeps()
		m := newCLIMeetings(t, inWindow())
		d.Meetings = m
		if c := Run(append([]string{"m365"}, args...), d); c != 0 || out.String() != meetingsHelpText || m.calls != 0 {
			t.Errorf("%v: exit %d calls %d out %q", args, c, m.calls, out.String())
		}
	}
	d, out, _ := testDeps()
	Run([]string{"m365", "--help"}, d)
	if !strings.Contains(out.String(), "\n  meetings  Teams meeting transcripts\n") {
		t.Fatalf("root help %q", out.String())
	}
}

func TestMeetingsNotAvailable(t *testing.T) {
	d, _, errw := testDeps()
	loginMeetings(t, &d)
	if c := Run([]string{"m365", "meetings", "transcript", "--event", "e", "--out", "o"}, d); c != 3 {
		t.Fatalf("exit %d", c)
	}
	if errw.String() != `{"class":"usage","message":"meetings not available"}`+"\n" {
		t.Fatalf("stderr %q", errw.String())
	}
}

func TestMCPRunMeetingsTranscriptIsARead(t *testing.T) {
	d, _, errw := testDeps()
	d.Meetings = newCLIMeetings(t, inWindow())
	loginMeetings(t, &d)
	cs := connectMCP(t, d)
	dest := filepath.Join(t.TempDir(), "m.vtt")
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "m365_run", Arguments: runIn{Namespace: "meetings", Verb: "transcript", Flags: map[string]any{"event": "evt-online", "out": dest}},
	})
	if err != nil || res.IsError {
		t.Fatal(err, toolText(t, res), errw.String())
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(toolText(t, res)), &got); err != nil {
		t.Fatal(err)
	}
	if _, dry := got["dry_run"]; dry || got["id"] != "t1" {
		t.Fatalf("result %v", got)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
}
