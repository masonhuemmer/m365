// Package meetings reads Teams meeting transcripts the signed-in user was invited to.
package meetings

import (
	"context"
	"time"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Store is the meetings port, in workload terms. The adapter enforces the
// page and content caps.
type Store interface {
	EventMeeting(ctx context.Context, eventID string) (Occurrence, error)
	FindMeeting(ctx context.Context, joinURL string) (string, error)
	Transcripts(ctx context.Context, meetingID string) ([]domain.Transcript, error)
	TranscriptContent(ctx context.Context, meetingID, transcriptID string) ([]byte, error)
}

// Occurrence is one calendar occurrence of an online meeting, in UTC.
type Occurrence struct {
	JoinURL string
	Start   time.Time
	End     time.Time
}

// Target names the meeting (exactly one of EventID or JoinURL) and, for a
// save, the transcript and the destination.
type Target struct {
	EventID   string
	JoinURL   string
	ID        string
	Out       string
	Overwrite bool
}
