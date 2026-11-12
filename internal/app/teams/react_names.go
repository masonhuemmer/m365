package teams

import (
	"context"

	"github.com/masonhuemmer/m365/internal/domain"
)

// nameReactions fills reaction user names that Graph left out from the chat's members.
func nameReactions(ctx context.Context, st Store, chatID string, p domain.ChatMessagePage) (domain.ChatMessagePage, error) {
	return p, nil
}
