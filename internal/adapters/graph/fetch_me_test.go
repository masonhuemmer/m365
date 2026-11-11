package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// pointMeAt serves a synthetic /me with status and body, and points fetchMe
// at it. The returned slice records each request's Authorization header.
func pointMeAt(t *testing.T, status int, body string) *[]string {
	t.Helper()
	auth := &[]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*auth = append(*auth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	prev := meURL
	meURL = srv.URL + "/v1.0/me"
	t.Cleanup(func() { meURL = prev })
	return auth
}

func TestFetchMeNamesTheAccount(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"mail first", `{"mail":"avery@example.test","userPrincipalName":"avery.upn@example.test"}`, "avery@example.test"},
		{"no mail, so the UPN", `{"userPrincipalName":"blake@example.test"}`, "blake@example.test"},
	}
	for _, c := range cases {
		auth := pointMeAt(t, http.StatusOK, c.body)
		if got := fetchMe(context.Background(), "synthetic-token"); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
		if len(*auth) != 1 || (*auth)[0] != "Bearer synthetic-token" {
			t.Errorf("%s: Authorization %q", c.name, *auth)
		}
	}
}

func TestFetchMeGivesNothingWhenGraphDoesNot(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"refused", http.StatusUnauthorized, `{"error":{"code":"InvalidAuthenticationToken"}}`},
		{"not JSON", http.StatusOK, `not json`},
	}
	for _, c := range cases {
		pointMeAt(t, c.status, c.body)
		if got := fetchMe(context.Background(), "synthetic-token"); got != "" {
			t.Errorf("%s: %q, want empty", c.name, got)
		}
	}
	prev := meURL
	t.Cleanup(func() { meURL = prev })
	for _, bad := range []string{"http://127.0.0.1:1/v1.0/me", "://no-scheme"} {
		meURL = bad
		if got := fetchMe(context.Background(), "synthetic-token"); got != "" {
			t.Errorf("%s: %q, want empty", bad, got)
		}
	}
}
