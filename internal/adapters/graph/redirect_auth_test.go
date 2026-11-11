package graph

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// redirectOnce answers the first request with a 302 to `to` and records the
// Authorization header each request carried.
type redirectOnce struct {
	to   string
	auth []string
}

func (r *redirectOnce) RoundTrip(req *http.Request) (*http.Response, error) {
	r.auth = append(r.auth, req.Header.Get("Authorization"))
	if len(r.auth) == 1 {
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {r.to}}, Body: http.NoBody, Request: req}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

// authAfterRedirect returns the Authorization header sent to `to`.
func authAfterRedirect(t *testing.T, to string) string {
	t.Helper()
	rt := &redirectOnce{to: to}
	c := &HTTPClient{Base: liveGraphBase, Token: "fake-token", Client: &http.Client{Transport: rt}}
	res, err := c.request(context.Background(), http.MethodGet, liveGraphBase+"/me/drive/items/x/content", nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if len(rt.auth) != 2 {
		t.Fatalf("%d requests, want the original and the redirect", len(rt.auth))
	}
	return rt.auth[1]
}

func TestRedirectOffTheGraphHostDropsTheToken(t *testing.T) {
	for _, to := range []string{
		"https://evil.graph.microsoft.com/v1.0/x",
		"http://graph.microsoft.com/v1.0/x",
		"https://graph.microsoft.com:8443/v1.0/x",
	} {
		if got := authAfterRedirect(t, to); got != "" {
			t.Errorf("redirect to %s sent Authorization %q", to, got)
		}
	}
}

func TestRedirectOnTheGraphHostKeepsTheToken(t *testing.T) {
	if got := authAfterRedirect(t, "https://graph.microsoft.com/v1.0/other"); got != "Bearer fake-token" {
		t.Fatalf("Authorization %q after a same-host redirect", got)
	}
}

// redirectLoop answers every request with a 302 back to itself.
type redirectLoop struct{ calls int }

func (r *redirectLoop) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls++
	return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {req.URL.String()}}, Body: http.NoBody, Request: req}, nil
}

func TestRedirectLoopStopsAfterTen(t *testing.T) {
	rt := &redirectLoop{}
	c := &HTTPClient{Base: liveGraphBase, Token: "fake-token", Client: &http.Client{Transport: rt}}
	res, err := c.request(context.Background(), http.MethodGet, liveGraphBase+"/me", nil, "", "")
	if err == nil {
		_ = res.Body.Close()
		t.Fatal("a redirect loop returned no error")
	}
	if !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("error %q", err)
	}
}
