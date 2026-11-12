package teams

import (
	"context"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Graph cannot read Notes (48:notes) as a chat, and every message there is yours,
// so --last is refused before any Graph call.
func TestReactLastRefusedInNotes(t *testing.T) {
	st, r := reactStore(), &reactorFake{}
	for _, in := range []ReactInput{
		{ChatID: SelfChatID, Last: true, Reaction: "like"},
		{ChatID: SelfChatID, Last: true, Reaction: "like", DryRun: true},
	} {
		_, err := React(context.Background(), st, r, nil, reactSess(), in)
		if domain.ExitOf(err) != domain.ExitUsage || !strings.Contains(err.Error(), "MESSAGE_ID") {
			t.Fatalf("%+v: err %v, want usage naming MESSAGE_ID", in, err)
		}
	}
	if st.getChats != 0 || len(st.queries) != 0 || len(r.calls) != 0 {
		t.Fatalf("Graph calls: chats %d, message pages %d, reactions %v; want none", st.getChats, len(st.queries), r.calls)
	}
}
