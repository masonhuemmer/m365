package graph

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// callback sends one sign-in callback to the loopback and returns its status.
func callback(t *testing.T, redirect, query string) int {
	t.Helper()
	res, err := http.Get(redirect + "?" + query)
	if err != nil {
		return 0
	}
	_ = res.Body.Close()
	return res.StatusCode
}

func TestLoopbackIgnoresCallbacksWithoutItsState(t *testing.T) {
	redirect, wait, err := StartLoopback("synthetic-state")
	if err != nil {
		t.Fatal(err)
	}
	statuses := make(chan []int, 1)
	go func() {
		statuses <- []int{
			callback(t, redirect, "code=planted&state=other"),
			callback(t, redirect, "code=planted"),
			callback(t, redirect, "code=real&state=synthetic-state"),
		}
	}()
	code, err := wait(context.Background())
	if err != nil || code != "real" {
		t.Fatalf("code %q, err %v; want the callback that carries the state", code, err)
	}
	got := <-statuses
	if got[0] != http.StatusBadRequest || got[1] != http.StatusBadRequest || got[2] != http.StatusOK {
		t.Fatalf("statuses %v, want [400 400 200]", got)
	}
}

func TestLoopbackRefusalWithItsStateEndsTheSignIn(t *testing.T) {
	redirect, wait, err := StartLoopback("synthetic-state")
	if err != nil {
		t.Fatal(err)
	}
	go callback(t, redirect, "error=access_denied&state=synthetic-state")
	if _, err := wait(context.Background()); err == nil || domain.ExitOf(err) != domain.ExitAuth {
		t.Fatalf("err %v, want an auth error", err)
	}
}

func TestEachSignInGetsAFreshState(t *testing.T) {
	a, b := newState(), newState()
	if a == b || a == "state" || len(a) < 43 {
		t.Fatalf("states %q and %q; want two different random values of at least 43 characters", a, b)
	}
}

func TestLoginPKCESendsTheLoopbackTheStateItAsksFor(t *testing.T) {
	cfg, _ := pkceFixture(t, http.StatusOK)
	prev := newState
	newState = func() string { return "synthetic-state" }
	t.Cleanup(func() { newState = prev })
	var opened, waitedFor string
	start := func(state string) (string, func(context.Context) (string, error), error) {
		waitedFor = state
		return fakeStart("synthetic-code", nil)(state)
	}
	if _, err := LoginPKCE(context.Background(), cfg, func(u string) error { opened = u; return nil }, start); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(opened)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query().Get("state"); got != "synthetic-state" || waitedFor != "synthetic-state" {
		t.Fatalf("authorize state %q, loopback waited for %q", got, waitedFor)
	}
}
