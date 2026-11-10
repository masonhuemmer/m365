package graph

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// fixedAnswer answers every request with 200 and the same body.
type fixedAnswer string

func (f fixedAnswer) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(f))), Request: req}, nil
}

func answering(body string) *HTTPMeetings {
	return &HTTPMeetings{HTTPClient: &HTTPClient{Base: "https://graph.microsoft.com/v1.0", Token: "synthetic-token", Client: &http.Client{Transport: fixedAnswer(body)}}}
}

// TestUnparseableGraphAnswers covers command-catalog row 15 at the adapter.
func TestUnparseableGraphAnswers(t *testing.T) {
	ctx := context.Background()
	_, err := answering("not json").EventMeeting(ctx, "evt")
	wantMeetingsErr(t, err, domain.ClassService, "invalid event")
	_, err = answering(`{"isOnlineMeeting":true,"onlineMeeting":{"joinUrl":"https://teams.example.test/x"},"start":{"dateTime":"soon"},"end":{"dateTime":"later"}}`).EventMeeting(ctx, "evt")
	wantMeetingsErr(t, err, domain.ClassService, "invalid event")
	_, err = answering("not json").FindMeeting(ctx, "https://teams.example.test/x")
	wantMeetingsErr(t, err, domain.ClassService, "invalid meeting list")
	_, err = answering("not json").Transcripts(ctx, "mtg-1")
	wantMeetingsErr(t, err, domain.ClassService, "invalid transcript")
}
