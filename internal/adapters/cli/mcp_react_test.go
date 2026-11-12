package cli

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/adapters/graph"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func reactMCP(t *testing.T, p MCPPolicy) (*mcp.ClientSession, *graph.Memory) {
	t.Helper()
	d, _, _ := testDeps()
	loginAll(t, &d)
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := NewMCPServerWithPolicy(d, p).Connect(ctx, t1, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, d.Teams.(graph.TeamsAPI).Memory
}

func callRun(t *testing.T, cs *mcp.ClientSession, in runIn) bool {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "m365_run", Arguments: in})
	if err != nil {
		t.Fatal(err)
	}
	return !res.IsError
}

func TestMCPReactIsAGuardedWrite(t *testing.T) {
	cs, mem := reactMCP(t, MCPPolicy{})
	byIDs := runIn{Namespace: "teams", Verb: "react", Args: []string{"chat-ajay", "cmsg-ajay-1", "like"}}
	if !callRun(t, cs, byIDs) || len(mem.Sent) != 0 {
		t.Fatalf("without opt-in: want a dry run, sent %d", len(mem.Sent))
	}
	byIDs.WriteOptIn = true
	if !callRun(t, cs, byIDs) || len(mem.Sent) != 1 {
		t.Fatalf("with opt-in: want one reaction, sent %d", len(mem.Sent))
	}
}

func TestMCPReactFollowsAllow(t *testing.T) {
	in := runIn{Namespace: "teams", Verb: "react", WriteOptIn: true, Args: []string{"chat-ajay", "cmsg-ajay-1", "like"}}
	cs, mem := reactMCP(t, MCPPolicy{Allow: map[string]bool{"teams.send": true}})
	if callRun(t, cs, in) || len(mem.Sent) != 0 {
		t.Fatalf("--allow without teams.react: want refusal, sent %d", len(mem.Sent))
	}
	cs, mem = reactMCP(t, MCPPolicy{Allow: map[string]bool{"teams.react": true}})
	if !callRun(t, cs, in) || len(mem.Sent) != 1 {
		t.Fatalf("--allow teams.react: want one reaction, sent %d", len(mem.Sent))
	}
	if _, err := newMCPPolicy(false, "teams.react", false, true); err != nil {
		t.Fatalf("--allow teams.react refused: %v", err)
	}
}

func TestMCPReactExactRecipients(t *testing.T) {
	cs, mem := reactMCP(t, MCPPolicy{ExactRecipients: true})
	byName := runIn{Namespace: "teams", Verb: "react", WriteOptIn: true, Args: []string{"like"}, Flags: map[string]any{"to": "Ajay", "last": true}}
	if callRun(t, cs, byName) || len(mem.Sent) != 0 {
		t.Fatalf("display name with --exact-recipients: want refusal, sent %d", len(mem.Sent))
	}
	byEmail := runIn{Namespace: "teams", Verb: "react", WriteOptIn: true, Args: []string{"like"}, Flags: map[string]any{"to": "ajay@example.com", "last": true}}
	if !callRun(t, cs, byEmail) || len(mem.Sent) != 1 {
		t.Fatalf("exact email with --exact-recipients: want one reaction, sent %d", len(mem.Sent))
	}
}
