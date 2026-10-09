package graph

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/teams"
)

func TestHTTPTeamsSendPostsMentions(t *testing.T) {
	var msg map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &msg)
		_, _ = w.Write([]byte(`{"id":"MSG1"}`))
	}))
	defer srv.Close()
	c := &HTTPTeams{HTTPClient: &HTTPClient{Base: srv.URL, Token: "fake-both", Client: srv.Client()}}
	in := teams.SendInput{
		ChatID:   "19:chat@thread.v2",
		Mentions: []teams.Mention{{ID: 0, Name: "Ajay Mathew", UserID: "u-ajay"}, {ID: 1, Name: "Bo Chen", UserID: "u-bo"}},
	}
	in.Rendered.Content = `<p><at id="0">Ajay Mathew</at> and <at id="1">Bo Chen</at> look</p>`

	if _, err := c.Send(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	ms, _ := msg["mentions"].([]any)
	if len(ms) != 2 {
		t.Fatalf("mentions %v", msg["mentions"])
	}
	first, _ := ms[0].(map[string]any)
	user, _ := first["mentioned"].(map[string]any)["user"].(map[string]any)
	if first["id"] != float64(0) || first["mentionText"] != "Ajay Mathew" ||
		user["id"] != "u-ajay" || user["displayName"] != "Ajay Mathew" || user["userIdentityType"] != "aadUser" {
		t.Fatalf("first mention %v", first)
	}
	body, _ := msg["body"].(map[string]any)
	if body["content"] != in.Rendered.Content {
		t.Fatalf("body %v", body)
	}
}

func TestHTTPTeamsSendWithoutMentionsOmitsTheField(t *testing.T) {
	var msg map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &msg)
		_, _ = w.Write([]byte(`{"id":"MSG1"}`))
	}))
	defer srv.Close()
	c := &HTTPTeams{HTTPClient: &HTTPClient{Base: srv.URL, Token: "fake-both", Client: srv.Client()}}
	in := teams.SendInput{ChatID: "c"}
	in.Rendered.Content = "<p>hi</p>"
	if _, err := c.Send(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if _, ok := msg["mentions"]; ok {
		t.Fatalf("mentions present: %v", msg["mentions"])
	}
}
