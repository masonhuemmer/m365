package graph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type adminCall struct {
	method, path string
	body         map[string]any
}

func adminServer(t *testing.T, calls *[]adminCall) *HTTPTeams {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := adminCall{method: r.Method, path: r.URL.Path}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			_ = json.Unmarshal(b, &c.body)
		}
		*calls = append(*calls, c)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/me":
			_, _ = w.Write([]byte(`{"id":"me-id"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/chats":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"19:new@thread.v2","topic":"NOC","chatType":"group"}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)
	return &HTTPTeams{HTTPClient: &HTTPClient{Base: srv.URL, Token: fakeJWT(`{"tid":"tenant-1"}`)}}
}

func fakeJWT(claims string) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(claims)) + ".sig"
}

func TestHTTPTeamsCreateChatPostsOwnersWithSelfFirst(t *testing.T) {
	var calls []adminCall
	chat, err := adminServer(t, &calls).CreateChat(context.Background(), "NOC", []string{"a@example.com", "o'neil@example.com"})
	if err != nil || chat.ID != "19:new@thread.v2" || chat.Type != "group" {
		t.Fatal(chat, err)
	}
	post := calls[len(calls)-1]
	if post.body["chatType"] != "group" || post.body["topic"] != "NOC" {
		t.Fatalf("%v", post.body)
	}
	members := post.body["members"].([]any)
	if len(members) != 3 {
		t.Fatalf("members: %v", members)
	}
	want := []string{"me-id", "a@example.com", "o''neil@example.com"}
	for i, m := range members {
		mm := m.(map[string]any)
		got := mm["user@odata.bind"].(string)
		if got != "https://graph.microsoft.com/v1.0/users('"+want[i]+"')" || mm["roles"].([]any)[0] != "owner" {
			t.Fatalf("member %d: %v", i, mm)
		}
	}
}

func TestHTTPTeamsRenameChatPatchesTopic(t *testing.T) {
	var calls []adminCall
	if err := adminServer(t, &calls).RenameChat(context.Background(), "19:abc@thread.v2", "New"); err != nil {
		t.Fatal(err)
	}
	c := calls[0]
	if c.method != http.MethodPatch || c.path != "/chats/19:abc@thread.v2" || c.body["topic"] != "New" {
		t.Fatalf("%+v", c)
	}
}

func TestHTTPTeamsHideAndUnhidePostForSelfAndTenant(t *testing.T) {
	var calls []adminCall
	tc := adminServer(t, &calls)
	for _, hidden := range []bool{true, false} {
		if err := tc.HideChat(context.Background(), "19:abc@thread.v2", hidden); err != nil {
			t.Fatal(err)
		}
	}
	var posts []adminCall
	for _, c := range calls {
		if c.method == http.MethodPost {
			posts = append(posts, c)
		}
	}
	if len(posts) != 2 || posts[0].path != "/chats/19:abc@thread.v2/hideForUser" || posts[1].path != "/chats/19:abc@thread.v2/unhideForUser" {
		t.Fatalf("%+v", posts)
	}
	user := posts[0].body["user"].(map[string]any)
	if user["id"] != "me-id" || user["tenantId"] != "tenant-1" {
		t.Fatalf("user: %v", user)
	}
}

func TestHTTPTeamsHideNeedsTenantClaim(t *testing.T) {
	var calls []adminCall
	tc := adminServer(t, &calls)
	tc.HTTPClient.Token = "not-a-jwt"
	if err := tc.HideChat(context.Background(), "19:abc@thread.v2", true); err == nil {
		t.Fatal("hid a chat without a tenant id")
	}
	for _, c := range calls {
		if c.method == http.MethodPost {
			t.Fatalf("posted without a tenant id: %+v", c)
		}
	}
}
