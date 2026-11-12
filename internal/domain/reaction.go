package domain

// Reaction is one user's emoji on a chat message.
type Reaction struct {
	Emoji   string `json:"emoji"`
	UserID  string `json:"user_id,omitempty"`
	User    string `json:"user,omitempty"`
	Created string `json:"created,omitempty"`
}

// ParseReaction returns the emoji for a reaction name or for an emoji given directly.
func ParseReaction(s string) (string, error) {
	return "", nil
}

// ReactionEmoji maps Graph's legacy reaction names to their emoji; other values stay as they are.
func ReactionEmoji(graphType string) string {
	return ""
}
