package graph

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

type reactCall struct {
	method, path string
	body         map[string]any
}

// reactServer answers message lists with msgs and every other request with status.
func reactServer(t *testing.T, status int, msgs string, calls *[]reactCall) *HTTPTeams {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := reactCall{method: r.Method, path: r.URL.EscapedPath()}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			_ = json.Unmarshal(b, &c.body)
		}
		*calls = append(*calls, c)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/messages") {
			_, _ = w.Write([]byte(msgs))
			return
		}
		w.WriteHeader(status)
		if status >= 400 {
			_, _ = w.Write([]byte(`{"error":{"code":"BadRequest","message":"synthetic"}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return &HTTPTeams{HTTPClient: &HTTPClient{Base: srv.URL, Token: "fake-token"}}
}

func TestHTTPTeamsReactSetsAndUnsets(t *testing.T) {
	for remove, action := range map[bool]string{false: "setReaction", true: "unsetReaction"} {
		var calls []reactCall
		err := reactServer(t, http.StatusNoContent, "", &calls).React(context.Background(), "19:abc@thread.v2", "1700000000000", "👍", remove)
		if err != nil {
			t.Fatal(err)
		}
		want := []reactCall{{method: http.MethodPost, path: "/chats/19:abc@thread.v2/messages/1700000000000/" + action, body: map[string]any{"reactionType": "👍"}}}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("calls %+v, want %+v", calls, want)
		}
	}
}

func TestHTTPTeamsReactEscapesIDs(t *testing.T) {
	var calls []reactCall
	if err := reactServer(t, http.StatusNoContent, "", &calls).React(context.Background(), "a/b", "m?1", "👀", false); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].path != "/chats/a%2Fb/messages/m%3F1/setReaction" {
		t.Fatalf("calls %+v", calls)
	}
}

func TestHTTPTeamsReactErrors(t *testing.T) {
	for status, exit := range map[int]int{http.StatusNotFound: domain.ExitNotFound, http.StatusBadRequest: domain.ExitService, http.StatusForbidden: domain.ExitAuth} {
		var calls []reactCall
		err := reactServer(t, status, "", &calls).React(context.Background(), "19:abc@thread.v2", "1", "👍", false)
		if domain.ExitOf(err) != exit {
			t.Fatalf("status %d: err %v exit %d, want %d", status, err, domain.ExitOf(err), exit)
		}
	}
}

func TestHTTPTeamsMessagesMapsReactions(t *testing.T) {
	msgs := `{"value":[
	  {"id":"m1","createdDateTime":"2026-01-01T00:00:00Z","messageType":"message","body":{"content":"hi"},
	   "from":{"user":{"id":"u-a","displayName":"Ajay"}},
	   "reactions":[
	     {"reactionType":"like","createdDateTime":"2026-01-01T01:00:00Z","user":{"user":{"id":"u-b","displayName":null}}},
	     {"reactionType":"🎉","createdDateTime":"2026-01-01T02:00:00Z","user":{"user":{"id":"u-c","displayName":"Cara"}}},
	     {"reactionType":"custom","displayName":"partyparrot","user":{"user":{"id":"u-d"}}}
	   ]},
	  {"id":"m2","createdDateTime":"2026-01-01T03:00:00Z","messageType":"message","body":{"content":"x"},
	   "from":{"user":{"id":"u-a","displayName":"Ajay"}},"reactions":[]}
	]}`
	var calls []reactCall
	p, err := reactServer(t, http.StatusOK, msgs, &calls).Messages(context.Background(), teams.MessageQuery{ChatID: "c1", Top: 20})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Reaction{
		{Emoji: "👍", UserID: "u-b", Created: "2026-01-01T01:00:00Z"},
		{Emoji: "🎉", UserID: "u-c", User: "Cara", Created: "2026-01-01T02:00:00Z"},
		{Emoji: "custom", UserID: "u-d"},
	}
	if len(p.Items) != 2 || !reflect.DeepEqual(p.Items[0].Reactions, want) {
		t.Fatalf("items %+v, want first with reactions %+v", p.Items, want)
	}
	if len(p.Items[1].Reactions) != 0 {
		t.Fatalf("m2 reactions %+v, want none", p.Items[1].Reactions)
	}
	if len(calls) != 1 {
		t.Fatalf("Graph requests %d, want 1", len(calls))
	}
}
