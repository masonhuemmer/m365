package teams

import (
	"context"
	"reflect"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func TestMessagesNamesReactionsFromMembers(t *testing.T) {
	st := reactStore()
	st.msgs["chat-ajay"] = []domain.ChatMessage{
		{ID: "m1", Reactions: []domain.Reaction{{Emoji: "👍", UserID: "u-ajay", Created: "2026-01-01T10:00:00Z"}}},
		{ID: "m2", Reactions: []domain.Reaction{{Emoji: "👀", UserID: "u-gone"}, {Emoji: "😆", UserID: "u-me", User: "Named by Graph"}}},
	}
	p, err := Messages(context.Background(), st, reactSess(), MessageQuery{ChatID: "chat-ajay"})
	if err != nil {
		t.Fatal(err)
	}
	want := [][]domain.Reaction{
		{{Emoji: "👍", UserID: "u-ajay", User: "Ajay Kumar", Created: "2026-01-01T10:00:00Z"}},
		{{Emoji: "👀", UserID: "u-gone"}, {Emoji: "😆", UserID: "u-me", User: "Named by Graph"}},
	}
	for i, m := range p.Items {
		if !reflect.DeepEqual(m.Reactions, want[i]) {
			t.Fatalf("message %s reactions %+v, want %+v", m.ID, m.Reactions, want[i])
		}
	}
	if st.getChats != 1 {
		t.Fatalf("chat lookups %d, want 1", st.getChats)
	}
}

func TestMessagesSkipsLookupWhenNamed(t *testing.T) {
	st := reactStore()
	st.msgs["chat-ajay"] = []domain.ChatMessage{
		{ID: "m1"},
		{ID: "m2", Reactions: []domain.Reaction{{Emoji: "😆", UserID: "u-me", User: "Named by Graph"}}},
	}
	if _, err := Messages(context.Background(), st, reactSess(), MessageQuery{ChatID: "chat-ajay"}); err != nil {
		t.Fatal(err)
	}
	if st.getChats != 0 {
		t.Fatalf("chat lookups %d, want 0", st.getChats)
	}
}
