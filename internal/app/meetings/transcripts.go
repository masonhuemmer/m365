package meetings

import (
	"context"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Save writes the chosen transcript's attributed WebVTT to t.Out and returns its summary.
func Save(ctx context.Context, st Store, sess domain.Session, t Target, write func(string, []byte, bool) error) (domain.SavedTranscript, error) {
	return domain.SavedTranscript{}, nil
}
