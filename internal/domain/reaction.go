package domain

import (
	"strings"
	"unicode"
)

// Reaction is one user's emoji on a chat message.
type Reaction struct {
	Emoji   string `json:"emoji"`
	UserID  string `json:"user_id,omitempty"`
	User    string `json:"user,omitempty"`
	Created string `json:"created,omitempty"`
}

// ReactionNames lists the names a reaction can be given by, in help order.
const ReactionNames = "like, heart, laugh, surprised, sad, angry, eyes"

var reactionEmoji = map[string]string{
	"like": "👍", "heart": "❤️", "laugh": "😆", "surprised": "😮", "sad": "😢", "angry": "😡", "eyes": "👀",
}

// maxEmojiCodePoints bounds a direct emoji; ZWJ sequences such as a family use up to 7.
const maxEmojiCodePoints = 10

// ParseReaction returns the emoji for a reaction name or for an emoji given directly.
func ParseReaction(s string) (string, error) {
	s = strings.TrimSpace(s)
	if e, ok := reactionEmoji[strings.ToLower(s)]; ok {
		return e, nil
	}
	if isEmoji(s) {
		return s, nil
	}
	return "", Usagef("reaction %q is not an emoji or one of: %s", s, ReactionNames)
}

// isEmoji accepts symbols with their modifiers and joiners, and refuses text.
func isEmoji(s string) bool {
	n, symbol := 0, false
	for _, r := range s {
		n++
		if unicode.IsLetter(r) || unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
		if unicode.Is(unicode.So, r) || unicode.Is(unicode.Me, r) {
			symbol = true
		}
	}
	return symbol && n <= maxEmojiCodePoints
}

// ReactionEmoji maps Graph's legacy reaction names to their emoji; other values stay as they are.
func ReactionEmoji(graphType string) string {
	if e, ok := reactionEmoji[graphType]; ok {
		return e
	}
	return graphType
}
