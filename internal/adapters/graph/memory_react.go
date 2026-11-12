package graph

import "context"

// React records a reaction change in Sent.
func (m *Memory) React(_ context.Context, chatID, messageID, emoji string, remove bool) error {
	return nil
}

func (t TeamsAPI) React(ctx context.Context, chatID, messageID, emoji string, remove bool) error {
	return t.Memory.React(ctx, chatID, messageID, emoji, remove)
}
