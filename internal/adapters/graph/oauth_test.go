package graph

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
	"golang.org/x/oauth2"
)

func TestPKCEHelpers(t *testing.T) {
	cfg := PKCEConfig("id", "tenant", "http://127.0.0.1/")
	v := oauth2.GenerateVerifier()
	u := cfg.AuthCodeURL("state", oauth2.S256ChallengeOption(v), oauth2.AccessTypeOffline)
	if u == "" || v == "" {
		t.Fatal("pkce")
	}
	fn := FakeLogin(true, false)
	b, err := fn(context.Background())
	if err != nil || !b.Mail || b.Teams || b.Calendar || b.Files {
		t.Fatalf("%+v %v", b, err)
	}
	all, err := FakeLoginAll(true, true, true, true)(context.Background())
	if err != nil || !all.Calendar || !all.Files {
		t.Fatalf("%+v %v", all, err)
	}
	joined := strings.Join(cfg.Scopes, " ")
	if !strings.Contains(joined, "Calendars.ReadWrite") || !strings.Contains(joined, "Files.ReadWrite") {
		t.Fatal(joined)
	}
	if strings.Contains(joined, "Shared") || strings.Contains(joined, "Files.ReadWrite.All") {
		t.Fatal(joined)
	}
	if domain.ExitOf(Denied()) != domain.ExitAuth {
		t.Fatal("denied")
	}
}

func TestStartLoopbackReceivesCode(t *testing.T) {
	redirect, wait, err := StartLoopback("s")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = http.Get(redirect + "?code=abc&state=s")
	}()
	code, err := wait(context.Background())
	if err != nil || code != "abc" {
		t.Fatalf("%q %v", code, err)
	}
}
