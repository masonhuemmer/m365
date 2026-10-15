package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/adapters/graph"
	"github.com/masonhuemmer/m365/internal/domain"
)

func adminDeps(t *testing.T) (Deps, *graph.Memory, func(args ...string) (int, map[string]any)) {
	d, out, errw := testDeps()
	mem := graph.Seed()
	d.Teams = graph.TeamsAPI{Memory: mem}
	d.Login = graph.FakeLoginAll(true, true, true, true)
	if code := Run([]string{"m365", "auth", "login"}, d); code != 0 {
		t.Fatal(errw.String())
	}
	run := func(args ...string) (int, map[string]any) {
		out.Reset()
		code := Run(append([]string{"m365", "teams"}, args...), d)
		var m map[string]any
		_ = json.Unmarshal(out.Bytes(), &m)
		return code, m
	}
	return d, mem, run
}

func TestTeamsCreateAndRenameThroughCLI(t *testing.T) {
	_, mem, run := adminDeps(t)
	before := len(mem.Chats)
	flags := []string{"create", "--name", "NOC Weekly", "--member", "a@example.com", "--member", "b@example.com"}
	if code, out := run(append(flags, "--dry-run")...); code != 0 || out["dry_run"] != true || len(mem.Chats) != before {
		t.Fatal(code, out)
	}
	code, out := run(flags...)
	if code != 0 || out["created"] != true || len(mem.Chats) != before+1 {
		t.Fatal(code, out)
	}
	id := out["id"].(string)
	if code, out := run("rename", id, "--name", "NOC Daily"); code != 0 || out["renamed"] != true || mem.Chats[len(mem.Chats)-1].Topic != "NOC Daily" {
		t.Fatal(code, out)
	}
	if code, _ := run("rename", "chat-1", "--name", "x"); code == 0 {
		t.Fatal("1:1 chat renamed")
	}
	if code, _ := run("create", "--name", "x", "--member", "a@example.com"); code == 0 {
		t.Fatal("one-member group created")
	}
	if code, _ := run("create", "stray", "--name", "x", "--member", "a@example.com", "--member", "b@example.com"); code == 0 {
		t.Fatal("positional argument accepted")
	}
}

func TestMCPTreatsChatCreateAndRenameAsWrites(t *testing.T) {
	for _, verb := range []string{"create", "rename"} {
		if !isWrite("teams", verb) {
			t.Fatalf("teams %s is not a guarded write", verb)
		}
	}
	p, err := newMCPPolicy(false, "teams.create,teams.rename", false, true)
	if err != nil {
		t.Fatal(err)
	}
	in := runIn{Namespace: "teams", Verb: "create", WriteOptIn: true}
	if err := p.check(&in); err != nil {
		t.Fatal(err)
	}
	in.Verb = "send"
	if err := p.check(&in); err == nil || !strings.Contains(err.Error(), "not permitted") || domain.ClassOf(err) != domain.ClassUsage {
		t.Fatalf("send allowed by create-only list: %v", err)
	}
}
