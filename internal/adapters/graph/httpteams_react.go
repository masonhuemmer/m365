package graph

import (
	"context"

	"github.com/masonhuemmer/m365/internal/domain"
)

// React sets or unsets the signed-in user's reaction on one chat message.
func (c *HTTPTeams) React(ctx context.Context, chatID, messageID, emoji string, remove bool) error {
	return nil
}

type graphReaction struct {
	ReactionType string `json:"reactionType"`
	Created      string `json:"createdDateTime"`
	User         *struct {
		User *graphIdentity `json:"user"`
	} `json:"user"`
}

func mapReactions(raw []graphReaction) []domain.Reaction {
	return nil
}
