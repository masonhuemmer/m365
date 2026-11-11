package graph

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// hugeBody stands in for a 401 body far larger than any Graph error. It
// counts the bytes the client reads.
type hugeBody struct{ read int }

func (b *hugeBody) Read(p []byte) (int, error) {
	if b.read >= 64<<20 {
		return 0, io.EOF
	}
	n := copy(p, strings.Repeat("x", len(p)))
	b.read += n
	return n, nil
}

func (b *hugeBody) Close() error { return nil }

// refusedOnce answers 401 with the huge body, then 200.
type refusedOnce struct {
	body  *hugeBody
	calls int
}

func (r *refusedOnce) RoundTrip(req *http.Request) (*http.Response, error) {
	r.calls++
	if r.calls == 1 {
		return &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{}, Body: r.body, Request: req}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

func TestRetryAfter401ReadsAtMost64KiBOfTheRefusal(t *testing.T) {
	body := &hugeBody{}
	c := &HTTPClient{
		Base:    liveGraphBase,
		Client:  &http.Client{Transport: &refusedOnce{body: body}},
		Refresh: func(bool) (string, error) { return "fresh", nil },
	}
	res, err := c.request(context.Background(), http.MethodGet, liveGraphBase+"/me", nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want the retry's 200", res.StatusCode)
	}
	if body.read > 64<<10 {
		t.Fatalf("read %d bytes of the 401 body, want at most %d", body.read, 64<<10)
	}
}
