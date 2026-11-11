package graph

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/mail"
	"github.com/masonhuemmer/m365/internal/domain"
)

const liveGraphBase = "https://graph.microsoft.com/v1.0"

var untrustedNextLinks = []string{
	"https://graph.microsoft.com.attacker.example/v1.0/me/messages?$skiptoken=x",
	"https://evilgraph.microsoft.com/v1.0/me/messages?$skiptoken=x",
	"http://graph.microsoft.com/v1.0/me/messages?$skiptoken=x",
}

type sentRequests struct{ origins []string }

func (s *sentRequests) RoundTrip(req *http.Request) (*http.Response, error) {
	s.origins = append(s.origins, req.URL.Scheme+"://"+req.URL.Host)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"value":[]}`)),
		Request:    req,
	}, nil
}

func clientRecording() (*HTTPClient, *sentRequests) {
	sent := &sentRequests{}
	return &HTTPClient{Base: liveGraphBase, Token: "fake-both", Client: &http.Client{Transport: sent}}, sent
}

func pageTokenFor(next string) string {
	return "p." + base64.RawURLEncoding.EncodeToString([]byte(next))
}

func TestPageTokenNeverSendsTheTokenToAnUntrustedHost(t *testing.T) {
	for _, next := range untrustedNextLinks {
		c, sent := clientRecording()
		_, err := c.List(context.Background(), mail.ListQuery{Top: 10, PageToken: pageTokenFor(next)})
		if domain.ClassOf(err) != domain.ClassUsage {
			t.Errorf("%s: want a usage error, got %v", next, err)
		}
		if len(sent.origins) != 0 {
			t.Errorf("%s: request sent to %v", next, sent.origins)
		}
	}
}

func TestMailDeltaCursorNeverSendsTheTokenToAnUntrustedHost(t *testing.T) {
	for _, next := range untrustedNextLinks {
		c, sent := clientRecording()
		_, err := (&HTTPMailDelta{HTTPClient: c}).Delta(context.Background(), mail.DeltaQuery{Folder: "inbox", Token: next})
		if domain.ClassOf(err) != domain.ClassUsage {
			t.Errorf("%s: want a usage error, got %v", next, err)
		}
		if len(sent.origins) != 0 {
			t.Errorf("%s: request sent to %v", next, sent.origins)
		}
	}
}

func TestPageTokenStillFollowsGraph(t *testing.T) {
	c, sent := clientRecording()
	_, err := c.List(context.Background(), mail.ListQuery{Top: 10, PageToken: pageTokenFor(liveGraphBase + "/me/messages?$skiptoken=x")})
	if err != nil {
		t.Fatal(err)
	}
	if len(sent.origins) != 1 || sent.origins[0] != "https://graph.microsoft.com" {
		t.Fatalf("want one request to Graph, got %v", sent.origins)
	}
}
