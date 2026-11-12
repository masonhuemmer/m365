package cli

import (
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

func TestTeamsReactThroughCLI(t *testing.T) {
	_, mem, run := adminDeps(t)
	code, out := run("react", "chat-ajay", "cmsg-ajay-1", "like", "--dry-run")
	if code != 0 || out["dry_run"] != true || out["action"] != "react" || out["reaction"] != "👍" || len(mem.Sent) != 0 {
		t.Fatalf("dry run: %d %v sent=%d", code, out, len(mem.Sent))
	}
	code, out = run("react", "chat-ajay", "cmsg-ajay-1", "👀")
	if code != 0 || out["reacted"] != true || out["chat_id"] != "chat-ajay" || out["message_id"] != "cmsg-ajay-1" || out["reaction"] != "👀" || len(mem.Sent) != 1 {
		t.Fatalf("react: %d %v sent=%d", code, out, len(mem.Sent))
	}
	code, out = run("react", "chat-ajay", "cmsg-ajay-1", "eyes", "--remove")
	if code != 0 || out["removed"] != true || len(mem.Sent) != 2 {
		t.Fatalf("remove: %d %v sent=%d", code, out, len(mem.Sent))
	}
	code, out = run("react", "--to", "Ajay", "--last", "like")
	if code != 0 || out["message_id"] != "cmsg-ajay-1" || out["from"] != "Ajay Kumar" || out["reacted"] != true {
		t.Fatalf("--to --last: %d %v", code, out)
	}
	code, out = run("react", "chat-ajay", "--last", "like", "--dry-run")
	if code != 0 || out["message_id"] != "cmsg-ajay-1" || out["dry_run"] != true {
		t.Fatalf("CHAT_ID --last: %d %v", code, out)
	}
}

func TestTeamsReactExitCodes(t *testing.T) {
	_, mem, run := adminDeps(t)
	for _, c := range []struct {
		args []string
		code int
	}{
		{[]string{"react", "chat-ajay", "cmsg-ajay-1", "hello"}, domain.ExitUsage},
		{[]string{"react", "--to", "Ajay", "like"}, domain.ExitUsage},
		{[]string{"react", "chat-ajay", "cmsg-ajay-1", "--last", "like"}, domain.ExitUsage},
		{[]string{"react", "a", "b", "c", "like"}, domain.ExitUsage},
		{[]string{"react"}, domain.ExitUsage},
		{[]string{"react", "chat-ajay", "cmsg-missing", "like"}, domain.ExitNotFound},
		{[]string{"react", "chat-missing", "cmsg-ajay-1", "like"}, domain.ExitNotFound},
	} {
		if code, out := run(c.args...); code != c.code {
			t.Fatalf("%v: exit %d (%v), want %d", c.args, code, out, c.code)
		}
	}
	if len(mem.Sent) != 0 {
		t.Fatalf("sent %d, want 0", len(mem.Sent))
	}
	mem.Throttle = true
	if code, _ := run("react", "chat-ajay", "cmsg-ajay-1", "like"); code != domain.ExitService {
		t.Fatalf("Graph failure: exit %d, want %d", code, domain.ExitService)
	}
}

func TestTeamsReactSignedOut(t *testing.T) {
	d, _, errw := testDeps()
	if code := Run([]string{"m365", "teams", "react", "chat-ajay", "cmsg-ajay-1", "like"}, d); code != domain.ExitAuth {
		t.Fatalf("exit %d, want %d (%s)", code, domain.ExitAuth, errw.String())
	}
}

// storeOnly hides React, as a Teams dependency that cannot react would.
type storeOnly struct{ teams.Store }

func TestTeamsReactWithoutReactor(t *testing.T) {
	d, _, _ := adminDeps(t)
	d.Teams = storeOnly{d.Teams}
	if code := Run([]string{"m365", "teams", "react", "chat-ajay", "cmsg-ajay-1", "like"}, d); code != domain.ExitService {
		t.Fatalf("exit %d, want %d", code, domain.ExitService)
	}
}

func TestTeamsReactHumanOutput(t *testing.T) {
	d, out, _ := testDeps()
	loginAll(t, &d)
	out.Reset()
	if code := Run([]string{"m365", "--human", "teams", "react", "chat-ajay", "cmsg-ajay-1", "like"}, d); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if s := out.String(); !strings.Contains(s, "👍") {
		t.Fatalf("human output: %q", s)
	}
}

func TestTeamsReactHelp(t *testing.T) {
	d, out, _ := testDeps()
	if code := Run([]string{"m365", "teams", "react", "--help"}, d); code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"m365 teams react", "CHAT_ID MESSAGE_ID", "--to", "--last", "50", "like, heart, laugh, surprised, sad, angry, eyes", "--remove", "--dry-run", "--human"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("react help lacks %q:\n%s", want, out.String())
		}
	}
	out.Reset()
	Run([]string{"m365", "teams", "--help"}, d)
	if !strings.Contains(out.String(), "m365 teams react") {
		t.Fatalf("teams help lacks react:\n%s", out.String())
	}
	out.Reset()
	Run([]string{"m365", "teams", "messages", "--help"}, d)
	if !strings.Contains(out.String(), "reactions") {
		t.Fatalf("messages help lacks reactions:\n%s", out.String())
	}
}

func TestTeamsMessagesShowReactions(t *testing.T) {
	_, _, run := adminDeps(t)
	code, out := run("messages", "chat-ajay")
	items, _ := out["items"].([]any)
	if code != 0 || len(items) != 2 {
		t.Fatalf("%d %v", code, out)
	}
	second, _ := items[1].(map[string]any)
	reactions, _ := second["reactions"].([]any)
	var first map[string]any
	if len(reactions) > 0 {
		first, _ = reactions[0].(map[string]any)
	}
	if first["emoji"] != "👍" || first["user_id"] != "u-ajay" || first["user"] != "Ajay Kumar" {
		t.Fatalf("reactions %v", reactions)
	}
	if _, has := items[0].(map[string]any)["reactions"]; has {
		t.Fatalf("message without reactions has the field: %v", items[0])
	}
}
