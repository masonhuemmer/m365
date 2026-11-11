package graph

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// meetingsRecorder records each request's origin. The first answer carries
// next as its continuation link; any later answer is an empty page.
type meetingsRecorder struct {
	next    string
	origins []string
}

func (m *meetingsRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	m.origins = append(m.origins, req.URL.Scheme+"://"+req.URL.Host)
	body := `{"value":[]}`
	if len(m.origins) == 1 {
		body = `{"value":[],"@odata.nextLink":"` + m.next + `"}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

// countingBody serves size bytes of WebVTT and counts how many are read.
type countingBody struct {
	size, read int
}

func (b *countingBody) Read(p []byte) (int, error) {
	if b.read >= b.size {
		return 0, io.EOF
	}
	n := min(len(p), b.size-b.read)
	for i := range n {
		p[i] = 'x'
	}
	b.read += n
	return n, nil
}

func (b *countingBody) Close() error { return nil }

type countingTransport struct{ body *countingBody }

func (c countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: c.body, Request: req}, nil
}

func TestTranscriptContentReadsAtMostOneByteOverTheCap(t *testing.T) {
	body := &countingBody{size: 64 << 20}
	c := &HTTPMeetings{HTTPClient: &HTTPClient{Base: "https://graph.microsoft.com/v1.0", Token: "synthetic-token", Client: &http.Client{Transport: countingTransport{body}}}}
	_, err := c.TranscriptContent(context.Background(), "mtg-1", "t1")
	de, ok := err.(*domain.Error)
	if !ok || de.Message != "transcript is larger than the 10 MiB limit" {
		t.Fatalf("err %#v", err)
	}
	if body.read > 10<<20+1 {
		t.Fatalf("read %d bytes, want at most %d", body.read, 10<<20+1)
	}
}

func TestTranscriptLinksStayOnGraph(t *testing.T) {
	links := []string{
		"https://graph.microsoft.com.attacker.example/v1.0/x",
		"https://evilgraph.microsoft.com/v1.0/x",
		"http://graph.microsoft.com/v1.0/x",
	}
	for _, next := range links {
		rec := &meetingsRecorder{next: next}
		c := &HTTPMeetings{HTTPClient: &HTTPClient{Base: "https://graph.microsoft.com/v1.0", Token: "synthetic-token", Client: &http.Client{Transport: rec}}}
		_, err := c.Transcripts(context.Background(), "mtg-1")
		de, ok := err.(*domain.Error)
		if !ok || de.Class != domain.ClassService || de.Message != "refused a transcript list link outside Microsoft Graph" {
			t.Errorf("%s: err %#v", next, err)
		}
		if !reflect.DeepEqual(rec.origins, []string{"https://graph.microsoft.com"}) {
			t.Errorf("%s: requests sent to %v", next, rec.origins)
		}
	}
}
