package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// wantOneJSONError checks stderr holds exactly one JSON error line of class usage.
func wantOneJSONError(t *testing.T, stderr string) {
	t.Helper()
	if strings.Count(stderr, "\n") != 1 || !strings.HasSuffix(stderr, "\n") {
		t.Fatalf("stderr is not one line: %q", stderr)
	}
	var e errObj
	if err := json.Unmarshal([]byte(stderr), &e); err != nil || e.Class != "usage" {
		t.Fatalf("stderr %q", stderr)
	}
}

func TestMeetingsFlagErrorsAreOneRedactedJSONLine(t *testing.T) {
	cases := [][]string{
		{"meetings", "transcript", "--overwrite=access_token=synthetic-secret", "--event", "e", "--out", "o"},
		{"meetings", "transcripts", "--bogus", "--event", "e"},
	}
	for _, args := range cases {
		d, _, errw := testDeps()
		d.Meetings = newCLIMeetings(t, inWindow())
		loginMeetings(t, &d)
		errw.Reset()
		if code := Run(append([]string{"m365"}, args...), d); code != 3 {
			t.Fatalf("%v: exit %d", args, code)
		}
		wantOneJSONError(t, errw.String())
		if strings.Contains(errw.String(), "synthetic-secret") {
			t.Fatalf("%v: secret in stderr %q", args, errw.String())
		}
	}
}

func TestMeetingsVerbHelpNeedsNoAdapter(t *testing.T) {
	d, out, _ := testDeps()
	if code := Run([]string{"m365", "meetings", "transcript", "--help"}, d); code != 0 || out.String() != meetingsHelpText {
		t.Fatalf("exit %d out %q", code, out.String())
	}
	res, err := connectMCP(t, d).CallTool(context.Background(), &mcp.CallToolParams{
		Name: "m365_help", Arguments: helpIn{Namespace: "meetings", Verb: "transcript"},
	})
	if err != nil || res.IsError || !strings.Contains(toolText(t, res), "Verbs: transcripts (list), transcript (save one)") {
		t.Fatalf("m365_help: %v %v", err, res)
	}
}
