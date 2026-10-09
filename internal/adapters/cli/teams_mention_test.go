package cli

import (
	"encoding/json"
	"testing"
)

// The fake environment (M365_FAKE) must be able to exercise a real mention, so
// its seeded members carry user ids.
func TestTeamsSendMentionsSeededMemberInFakeEnvironment(t *testing.T) {
	d, out, errw := testDeps()
	loginAll(t, &d)
	out.Reset()
	if c := Run([]string{"m365", "teams", "send", "chat-1", "--text", "@Alice are you free?", "--dry-run"}, d); c != 0 {
		t.Fatalf("exit %d: %s %s", c, out.String(), errw.String())
	}
	var got struct {
		Mentions []string `json:"mentions"`
		Rendered struct {
			Content string `json:"content"`
		} `json:"rendered"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Mentions) != 1 || got.Mentions[0] != "Alice" || got.Rendered.Content != `<p><at id="0">Alice</at> are you free?</p>` {
		t.Fatalf("%s", out.String())
	}
}
