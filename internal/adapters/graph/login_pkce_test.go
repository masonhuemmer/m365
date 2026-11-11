package graph

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

const syntheticRedirect = "http://localhost:1/"

// pkceFixture points a sign-in config at a synthetic token endpoint that
// answers with status, and replaces the live account lookup. The returned
// form holds the last token request.
func pkceFixture(t *testing.T, status int) (*oauth2.Config, *url.Values) {
	t.Helper()
	form := &url.Values{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		*form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"synthetic-access","refresh_token":"synthetic-refresh","token_type":"Bearer","expires_in":3600,"scope":"Mail.Read Chat.Read"}`))
	}))
	t.Cleanup(srv.Close)
	cfg := PKCEConfig("synthetic-client", "synthetic-tenant", "")
	cfg.Endpoint.TokenURL = srv.URL + "/token"
	cfg.Endpoint.AuthStyle = oauth2.AuthStyleInParams
	prev := lookupAccount
	lookupAccount = func(context.Context, string) string { return "avery@example.test" }
	t.Cleanup(func() { lookupAccount = prev })
	return cfg, form
}

// fakeStart is a loopback that returns code and err when waited on.
func fakeStart(code string, err error) auth.LoopbackStart {
	return func(string) (string, func(context.Context) (string, error), error) {
		return syntheticRedirect, func(context.Context) (string, error) { return code, err }, nil
	}
}

func TestLoginPKCERedeemsTheCodeWithItsVerifier(t *testing.T) {
	cfg, form := pkceFixture(t, http.StatusOK)
	var opened string
	open := func(u string) error { opened = u; return nil }
	b, err := LoginPKCE(context.Background(), cfg, open, fakeStart("synthetic-code", nil))
	if err != nil {
		t.Fatal(err)
	}
	if b.AccessToken != "synthetic-access" || b.RefreshToken != "synthetic-refresh" || b.Account != "avery@example.test" {
		t.Fatalf("blob %+v", b)
	}
	if !b.Usable || !b.Mail || !b.Teams || b.Calendar || b.Files || b.Meetings {
		t.Fatalf("flags %+v", b)
	}
	u, err := url.Parse(opened)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("redirect_uri") != syntheticRedirect || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("authorize URL %s", opened)
	}
	if form.Get("code") != "synthetic-code" || form.Get("redirect_uri") != syntheticRedirect {
		t.Fatalf("token request %v", *form)
	}
	if got := oauth2.S256ChallengeFromVerifier(form.Get("code_verifier")); got != q.Get("code_challenge") {
		t.Fatalf("verifier does not match the challenge: %q vs %q", got, q.Get("code_challenge"))
	}
}

func TestLoginPKCEFailuresAreAuthErrors(t *testing.T) {
	ok := func(string) error { return nil }
	bindFails := func(string) (string, func(context.Context) (string, error), error) {
		return "", nil, domain.Auth("synthetic bind failure")
	}
	cases := []struct {
		name   string
		status int
		open   auth.BrowserOpener
		start  auth.LoopbackStart
		want   string
	}{
		{"no browser opener", http.StatusOK, nil, fakeStart("c", nil), "login not available"},
		{"no loopback", http.StatusOK, ok, nil, "login not available"},
		{"loopback bind fails", http.StatusOK, ok, bindFails, "synthetic bind failure"},
		{"browser fails", http.StatusOK, func(string) error { return errors.New("synthetic browser failure") }, fakeStart("c", nil), "synthetic browser failure"},
		{"callback refused", http.StatusOK, ok, fakeStart("", domain.Auth("login refused")), "login refused"},
		{"token endpoint refuses", http.StatusBadRequest, ok, fakeStart("c", nil), "invalid_grant"},
	}
	for _, c := range cases {
		cfg, _ := pkceFixture(t, c.status)
		_, err := LoginPKCE(context.Background(), cfg, c.open, c.start)
		if err == nil || domain.ExitOf(err) != domain.ExitAuth || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want an auth error containing %q", c.name, err, c.want)
		}
	}
}
