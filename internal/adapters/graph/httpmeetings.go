package graph

import (
	"context"

	"github.com/masonhuemmer/m365/internal/app/meetings"
	"github.com/masonhuemmer/m365/internal/domain"
)

// HTTPMeetings reads online meetings and their transcripts from Microsoft Graph.
type HTTPMeetings struct{ *HTTPClient }

func (c *HTTPMeetings) EventMeeting(ctx context.Context, eventID string) (meetings.Occurrence, error) {
	return meetings.Occurrence{}, nil
}

func (c *HTTPMeetings) FindMeeting(ctx context.Context, joinURL string) (string, error) {
	return "", nil
}

func (c *HTTPMeetings) Transcripts(ctx context.Context, meetingID string) ([]domain.Transcript, error) {
	return nil, nil
}

func (c *HTTPMeetings) TranscriptContent(ctx context.Context, meetingID, transcriptID string) ([]byte, error) {
	return nil, nil
}
