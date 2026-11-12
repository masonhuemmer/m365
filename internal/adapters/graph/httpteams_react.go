package graph

import (
	"context"
	"net/http"
	"net/url"

	"github.com/masonhuemmer/m365/internal/domain"
)

// React sets or unsets the signed-in user's reaction on one chat message.
func (c *HTTPTeams) React(ctx context.Context, chatID, messageID, emoji string, remove bool) error {
	action := "setReaction"
	if remove {
		action = "unsetReaction"
	}
	path := "/chats/" + url.PathEscape(chatID) + "/messages/" + url.PathEscape(messageID) + "/" + action
	return c.doJSON(ctx, http.MethodPost, path, map[string]string{"reactionType": emoji})
}

type graphReaction struct {
	ReactionType string `json:"reactionType"`
	Created      string `json:"createdDateTime"`
	User         *struct {
		User *graphIdentity `json:"user"`
	} `json:"user"`
}

func mapReactions(raw []graphReaction) []domain.Reaction {
	var out []domain.Reaction
	for _, g := range raw {
		r := domain.Reaction{Emoji: domain.ReactionEmoji(g.ReactionType), Created: g.Created}
		if g.User != nil && g.User.User != nil {
			r.UserID, r.User = g.User.User.ID, g.User.User.DisplayName
		}
		out = append(out, r)
	}
	return out
}
