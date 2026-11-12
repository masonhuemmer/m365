package graph

import (
	"context"

	"github.com/masonhuemmer/m365/internal/domain"
)

// React records a reaction change in Sent.
func (m *Memory) React(_ context.Context, chatID, messageID, emoji string, remove bool) error {
	return nil
}

func (t TeamsAPI) React(ctx context.Context, chatID, messageID, emoji string, remove bool) error {
	return t.Memory.React(ctx, chatID, messageID, emoji, remove)
}

// seedAjayMessages is the 1:1 chat with Ajay: his message, then yours with his 👍,
// which Graph reports without a user name.
func seedAjayMessages() []domain.ChatMessage {
	return []domain.ChatMessage{
		{ID: "cmsg-ajay-1", ChatID: "chat-ajay", From: "Ajay Kumar", FromUserID: "u-ajay", SenderType: "user", Created: "2026-01-02T09:00:00Z", Text: "<p>Build is green.</p>"},
		{ID: "cmsg-ajay-2", ChatID: "chat-ajay", From: "Test User", FromUserID: "u-test", SenderType: "user", Created: "2026-01-02T09:05:00Z", Text: "<p>Thanks, deploying.</p>",
			Reactions: []domain.Reaction{{Emoji: "👍", UserID: "u-ajay", Created: "2026-01-02T09:06:00Z"}}},
	}
}
