package teams

import (
	"context"

	"github.com/masonhuemmer/m365/internal/domain"
)

// nameReactions fills reaction user names that Graph left out from the chat's members.
// It reads the chat only when a name is missing. A failed lookup leaves the names out
// (user_id stays), so reading messages never fails on it.
func nameReactions(ctx context.Context, st Store, chatID string, p domain.ChatMessagePage) domain.ChatMessagePage {
	if !missingReactionName(p.Items) {
		return p
	}
	chat, err := st.GetChat(ctx, chatID)
	if err != nil {
		return p
	}
	names := map[string]string{}
	for _, member := range chat.Members {
		if member.ID != "" {
			names[member.ID] = member.Name
		}
	}
	items := make([]domain.ChatMessage, len(p.Items))
	for i, msg := range p.Items {
		if len(msg.Reactions) > 0 {
			msg.Reactions = append([]domain.Reaction(nil), msg.Reactions...)
			for j := range msg.Reactions {
				if msg.Reactions[j].User == "" {
					msg.Reactions[j].User = names[msg.Reactions[j].UserID]
				}
			}
		}
		items[i] = msg
	}
	p.Items = items
	return p
}

func missingReactionName(msgs []domain.ChatMessage) bool {
	for _, msg := range msgs {
		for _, r := range msg.Reactions {
			if r.User == "" {
				return true
			}
		}
	}
	return false
}
