package teams

import (
	"context"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Reactor adds or removes the signed-in user's reaction on one chat message.
type Reactor interface {
	React(ctx context.Context, chatID, messageID, emoji string, remove bool) error
}

// ReactInput addresses one message by chat and message ID, or by --last in a chat or a --to match.
type ReactInput struct {
	ChatID         string
	MessageID      string
	To             string
	Reaction       string
	Last           bool
	Remove         bool
	DryRun         bool
	ExactRecipient bool
}

// LastMessageScan is how many messages --last looks at: the first page Graph returns.
const LastMessageScan = 50

// React adds or removes a reaction on one chat message.
func React(ctx context.Context, st Store, r Reactor, m ChatMap, sess domain.Session, in ReactInput) (any, error) {
	return nil, nil
}
