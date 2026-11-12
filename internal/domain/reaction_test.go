package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseReactionNames(t *testing.T) {
	cases := map[string]string{
		"like": "👍", "heart": "❤️", "laugh": "😆", "surprised": "😮",
		"sad": "😢", "angry": "😡", "eyes": "👀",
		"Like": "👍", "EYES": "👀", " heart ": "❤️",
	}
	for in, want := range cases {
		got, err := ParseReaction(in)
		if err != nil || got != want {
			t.Fatalf("ParseReaction(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestParseReactionEmoji(t *testing.T) {
	for _, in := range []string{"👍", "🎉", "👍🏽", "👩🏽‍💻", "1️⃣", "❤️", "👀"} {
		got, err := ParseReaction(in)
		if err != nil || got != in {
			t.Fatalf("ParseReaction(%q) = %q, %v; want it unchanged", in, got, err)
		}
	}
}

func TestParseReactionRefusesText(t *testing.T) {
	for _, in := range []string{"", "  ", "hello", "+1", "1", "!", "a👍", "👍 👍", "\x07", "👍\x07", strings.Repeat("👍", 11)} {
		got, err := ParseReaction(in)
		if err == nil {
			t.Fatalf("ParseReaction(%q) = %q, want a usage error", in, got)
		}
		if ClassOf(err) != ClassUsage {
			t.Fatalf("ParseReaction(%q) class %q, want usage", in, ClassOf(err))
		}
		if !strings.Contains(err.Error(), "like") || !strings.Contains(err.Error(), "eyes") {
			t.Fatalf("ParseReaction(%q) error %q should name the accepted names", in, err)
		}
	}
}

func TestReactionEmojiMapsLegacyNames(t *testing.T) {
	cases := map[string]string{
		"like": "👍", "heart": "❤️", "laugh": "😆", "surprised": "😮", "sad": "😢", "angry": "😡",
		"🎉": "🎉", "custom": "custom",
	}
	for in, want := range cases {
		if got := ReactionEmoji(in); got != want {
			t.Fatalf("ReactionEmoji(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChatMessageOmitsEmptyReactions(t *testing.T) {
	raw, err := json.Marshal(ChatMessage{ID: "m1", ChatID: "c1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "reactions") {
		t.Fatalf("empty reactions should be omitted: %s", raw)
	}
	raw, _ = json.Marshal(ChatMessage{ID: "m1", Reactions: []Reaction{{Emoji: "👍", UserID: "u1", User: "Ajay", Created: "2026-01-01T00:00:00Z"}}})
	if !strings.Contains(string(raw), `"reactions":[{"emoji":"👍","user_id":"u1","user":"Ajay","created":"2026-01-01T00:00:00Z"}]`) {
		t.Fatalf("reactions JSON: %s", raw)
	}
}
