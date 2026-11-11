package teams

import (
	"context"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

type memMap struct {
	ids map[string]string
}

func (m *memMap) Lookup(_, key string) (string, bool, error) {
	id, ok := m.ids[chatMapKey(key)]
	return id, ok && id != "", nil
}

func (m *memMap) Remember(_, key, chatID string) error {
	if m.ids == nil {
		m.ids = map[string]string{}
	}
	k := chatMapKey(key)
	if k == "" {
		return nil
	}
	m.ids[k] = chatID
	return nil
}

func TestChatMapKeysFromOneOnOne(t *testing.T) {
	c := domain.Chat{
		ID:   "chat-ajay",
		Type: "oneOnOne",
		Members: []domain.Person{
			{Name: "Ajay Kumar", Address: "ajay@example.com"},
			{Name: "Me", Address: "user@example.com"},
		},
	}
	keys := chatMapKeys(c, "user@example.com")
	got := strings.Join(keys, ",")
	if !strings.Contains(got, "ajay") || !strings.Contains(got, "ajay kumar") {
		t.Fatal(keys)
	}
}

func TestFindMappedRemembersAndLookupSkipsScan(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	m := &memMap{}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "user@example.com"}
	q, _ := ParseQuery("Ajay", false)
	r, err := FindMapped(context.Background(), st, m, sess, q, 0)
	if err != nil || r.Count != 1 || r.Items[0].ID != "chat-ajay" {
		t.Fatalf("%+v %v", r, err)
	}
	if m.ids["ajay kumar"] != "chat-ajay" && m.ids["ajay"] != "chat-ajay" {
		t.Fatalf("%v", m.ids)
	}
	first := st.listCalls
	r, err = FindMapped(context.Background(), st, m, sess, q, 0)
	if err != nil || r.Count != 1 || r.Items[0].ID != "chat-ajay" {
		t.Fatalf("lookup %+v %v", r, err)
	}
	if st.listCalls != first {
		t.Fatalf("rescanned: %d -> %d", first, st.listCalls)
	}
}

func TestSendMappedUsesChatMap(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	m := &memMap{ids: map[string]string{"ajay": "chat-ajay"}}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "user@example.com"}
	out, err := SendMapped(context.Background(), st, m, sess, SendInput{To: "Ajay", Text: "ping", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	got := out.(map[string]any)
	if got["chat_id"] != "chat-ajay" {
		t.Fatal(got)
	}
	if st.listCalls != 0 {
		t.Fatalf("list %d", st.listCalls)
	}
}

func TestListMappedRemembers(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	m := &memMap{}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "user@example.com"}
	if _, err := ListMapped(context.Background(), st, m, sess, 50, "", false); err != nil {
		t.Fatal(err)
	}
	id, ok, err := m.Lookup("user@example.com", "ajay kumar")
	if err != nil || !ok || id != "chat-ajay" {
		t.Fatalf("%s %v %v", id, ok, err)
	}
}
