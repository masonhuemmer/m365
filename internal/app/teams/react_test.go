package teams

import (
	"context"
	"reflect"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// reactMem is a chat store with messages; it returns every message, system events included,
// in the order given, as Graph does when it lists by last change.
type reactMem struct {
	chatMem
	msgs     map[string][]domain.ChatMessage
	queries  []MessageQuery
	getChats int
}

func (m *reactMem) GetChat(ctx context.Context, id string) (domain.Chat, error) {
	m.getChats++
	return m.chatMem.GetChat(ctx, id)
}

func (m *reactMem) Messages(_ context.Context, q MessageQuery) (domain.ChatMessagePage, error) {
	m.queries = append(m.queries, q)
	items := m.msgs[q.ChatID]
	return domain.ChatMessagePage{Limit: q.Top, Count: len(items), Items: items}, nil
}

type reactorFake struct {
	calls []string
	err   error
}

func (r *reactorFake) React(_ context.Context, chatID, messageID, emoji string, remove bool) error {
	action := "add"
	if remove {
		action = "remove"
	}
	r.calls = append(r.calls, chatID+"|"+messageID+"|"+emoji+"|"+action)
	return r.err
}

func reactSess() domain.Session {
	return domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "me@example.com"}
}

func reactChats() []domain.Chat {
	return []domain.Chat{
		{ID: "chat-1", Type: "oneOnOne", Members: []domain.Person{{ID: "u-alice", Name: "Alice", Address: "alice@example.com"}}},
		{ID: "chat-ajay", Type: "oneOnOne", Members: []domain.Person{
			{ID: "u-ajay", Name: "Ajay Kumar", Address: "ajay@example.com"},
			{ID: "u-me", Name: "Me", Address: "Me@Example.com"},
		}},
	}
}

func reactStore() *reactMem {
	return &reactMem{chatMem: chatMem{chats: reactChats()}, msgs: map[string][]domain.ChatMessage{
		"chat-ajay": {
			{ID: "m-old", ChatID: "chat-ajay", FromUserID: "u-ajay", From: "Ajay Kumar", Created: "2026-01-01T09:00:00Z", Text: "<p>old</p>"},
			{ID: "m-mine", ChatID: "chat-ajay", FromUserID: "u-me", From: "Me", Created: "2026-01-01T11:00:00Z", Text: "<p>mine</p>"},
			{ID: "m-sys", ChatID: "chat-ajay", System: true, Created: "2026-01-01T12:00:00Z"},
			{ID: "m-new", ChatID: "chat-ajay", FromUserID: "u-ajay", From: "Ajay Kumar", Created: "2026-01-01T10:00:00.5Z", Text: "<p>done</p>"},
		},
	}}
}

func TestReactByIDs(t *testing.T) {
	r := &reactorFake{}
	out, err := React(context.Background(), reactStore(), r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", MessageID: "m-old", Reaction: "like"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"chat_id": "chat-ajay", "message_id": "m-old", "reaction": "👍", "reacted": true}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("out %v, want %v", out, want)
	}
	if !reflect.DeepEqual(r.calls, []string{"chat-ajay|m-old|👍|add"}) {
		t.Fatalf("calls %v", r.calls)
	}
}

func TestReactRemove(t *testing.T) {
	r := &reactorFake{}
	out, err := React(context.Background(), reactStore(), r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", MessageID: "m-old", Reaction: "🎉", Remove: true})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"chat_id": "chat-ajay", "message_id": "m-old", "reaction": "🎉", "removed": true}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("out %v, want %v", out, want)
	}
	if !reflect.DeepEqual(r.calls, []string{"chat-ajay|m-old|🎉|remove"}) {
		t.Fatalf("calls %v", r.calls)
	}
}

func TestReactDryRunChangesNothing(t *testing.T) {
	for _, remove := range []bool{false, true} {
		r := &reactorFake{}
		out, err := React(context.Background(), reactStore(), r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", MessageID: "m-old", Reaction: "eyes", Remove: remove, DryRun: true})
		if err != nil {
			t.Fatal(err)
		}
		action := map[bool]string{false: "react", true: "remove"}[remove]
		want := map[string]any{"dry_run": true, "action": action, "chat_id": "chat-ajay", "message_id": "m-old", "reaction": "👀"}
		if !reflect.DeepEqual(out, want) || len(r.calls) != 0 {
			t.Fatalf("out %v calls %v, want %v and no calls", out, r.calls, want)
		}
	}
}

func TestReactLastPicksNewestFromSomeoneElse(t *testing.T) {
	st, r := reactStore(), &reactorFake{}
	out, err := React(context.Background(), st, r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", Last: true, Reaction: "eyes"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"chat_id": "chat-ajay", "message_id": "m-new", "reaction": "👀", "reacted": true,
		"from": "Ajay Kumar", "created": "2026-01-01T10:00:00.5Z", "text": "<p>done</p>",
	}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("out %v, want %v", out, want)
	}
	if !reflect.DeepEqual(r.calls, []string{"chat-ajay|m-new|👀|add"}) {
		t.Fatalf("calls %v", r.calls)
	}
	if len(st.queries) != 1 || st.queries[0].Top != LastMessageScan || st.queries[0].ChatID != "chat-ajay" {
		t.Fatalf("message queries %+v, want one page of %d", st.queries, LastMessageScan)
	}
}

func TestReactLastCountsAppMessages(t *testing.T) {
	st, r := reactStore(), &reactorFake{}
	st.msgs["chat-ajay"] = append(st.msgs["chat-ajay"], domain.ChatMessage{
		ID: "m-bot", ChatID: "chat-ajay", FromApplicationID: "app-1", From: "Build bot", Created: "2026-01-01T10:30:00Z",
	})
	out, err := React(context.Background(), st, r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", Last: true, Reaction: "like", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := out.(map[string]any)
	if m["message_id"] != "m-bot" || m["from"] != "Build bot" || m["dry_run"] != true || len(r.calls) != 0 {
		t.Fatalf("out %v calls %v", m, r.calls)
	}
}

func TestReactLastNoneFromOthers(t *testing.T) {
	st, r := reactStore(), &reactorFake{}
	st.msgs["chat-ajay"] = []domain.ChatMessage{{ID: "m-mine", FromUserID: "u-me", Created: "2026-01-01T11:00:00Z"}}
	_, err := React(context.Background(), st, r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", Last: true, Reaction: "like"})
	if domain.ExitOf(err) != domain.ExitNotFound || len(r.calls) != 0 {
		t.Fatalf("err %v calls %v, want not found and no calls", err, r.calls)
	}
}

func TestReactLastNeedsYouAmongMembers(t *testing.T) {
	st, r := reactStore(), &reactorFake{}
	sess := reactSess()
	sess.Account = "someone.else@example.com"
	_, err := React(context.Background(), st, r, nil, sess, ReactInput{ChatID: "chat-ajay", Last: true, Reaction: "like"})
	if domain.ExitOf(err) != domain.ExitUsage || len(r.calls) != 0 {
		t.Fatalf("err %v calls %v, want usage and no calls", err, r.calls)
	}
}

func TestReactLastByTo(t *testing.T) {
	for _, in := range []ReactInput{
		{To: "Ajay", Last: true, Reaction: "like"},
		{To: "ajay@example.com", ExactRecipient: true, Last: true, Reaction: "like"},
	} {
		r := &reactorFake{}
		out, err := React(context.Background(), reactStore(), r, nil, reactSess(), in)
		if err != nil {
			t.Fatalf("%+v: %v", in, err)
		}
		if m, _ := out.(map[string]any); m["chat_id"] != "chat-ajay" || m["message_id"] != "m-new" {
			t.Fatalf("%+v: out %v", in, m)
		}
		if !reflect.DeepEqual(r.calls, []string{"chat-ajay|m-new|👍|add"}) {
			t.Fatalf("%+v: calls %v", in, r.calls)
		}
	}
}

func TestReactRefusesBadForms(t *testing.T) {
	for _, in := range []ReactInput{
		{To: "Ajay", Reaction: "like"},
		{To: "Ajay", MessageID: "m-old", Reaction: "like"},
		{ChatID: "chat-ajay", MessageID: "m-old", Last: true, Reaction: "like"},
		{To: "Ajay", ChatID: "chat-ajay", Last: true, Reaction: "like"},
		{MessageID: "m-old", Reaction: "like"},
		{ChatID: "chat-ajay", Reaction: "like"},
		{Last: true, Reaction: "like"},
		{ChatID: "chat-ajay", MessageID: "m-old", Reaction: "hello"},
		{ChatID: "chat-ajay", MessageID: "m-old"},
	} {
		r := &reactorFake{}
		_, err := React(context.Background(), reactStore(), r, nil, reactSess(), in)
		if domain.ExitOf(err) != domain.ExitUsage || len(r.calls) != 0 {
			t.Fatalf("%+v: err %v calls %v, want usage and no calls", in, err, r.calls)
		}
	}
}

func TestReactNeedsTeams(t *testing.T) {
	r := &reactorFake{}
	sess := reactSess()
	sess.TeamsConsented = false
	_, err := React(context.Background(), reactStore(), r, nil, sess, ReactInput{ChatID: "chat-ajay", MessageID: "m-old", Reaction: "like"})
	if domain.ExitOf(err) != domain.ExitAuth || len(r.calls) != 0 {
		t.Fatalf("err %v calls %v, want auth and no calls", err, r.calls)
	}
}

func TestReactReturnsReactorErrors(t *testing.T) {
	for _, want := range []error{domain.Service("graph refused"), domain.NotFound("message not found")} {
		r := &reactorFake{err: want}
		_, err := React(context.Background(), reactStore(), r, nil, reactSess(), ReactInput{ChatID: "chat-ajay", MessageID: "m-old", Reaction: "like"})
		if err != want {
			t.Fatalf("err %v, want %v", err, want)
		}
	}
}
