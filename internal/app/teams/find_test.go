package teams

import (
	"context"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

type chatMem struct {
	chats        []domain.Chat
	listCalls    int
	visibleCalls int
}

func (m *chatMem) ListVisibleChats(ctx context.Context, top int, page string) (domain.ChatPage, error) {
	m.visibleCalls++
	return m.ListChats(ctx, top, page)
}

func (m *chatMem) ListChats(_ context.Context, top int, page string) (domain.ChatPage, error) {
	m.listCalls++
	skip := 0
	if strings.HasPrefix(page, "s.") {
		skip = atoi(strings.TrimPrefix(page, "s."))
	}
	if top <= 0 {
		top = len(m.chats)
	}
	if skip > len(m.chats) {
		skip = len(m.chats)
	}
	end := skip + top
	if end > len(m.chats) {
		end = len(m.chats)
	}
	p := domain.ChatPage{Limit: top, Items: m.chats[skip:end], Count: end - skip}
	if end < len(m.chats) {
		n := "s." + itoa(end)
		p.NextPage = &n
	}
	return p, nil
}
func (m *chatMem) GetChat(_ context.Context, id string) (domain.Chat, error) {
	for _, c := range m.chats {
		if c.ID == id {
			return c, nil
		}
	}
	return domain.Chat{}, domain.NotFound("no")
}
func (m *chatMem) Messages(context.Context, MessageQuery) (domain.ChatMessagePage, error) {
	return domain.ChatMessagePage{}, nil
}
func (m *chatMem) Send(context.Context, SendInput) (string, error) { return "id", nil }
func (m *chatMem) Watch(context.Context, WatchQuery) ([]domain.WatchEvent, error) {
	return nil, nil
}
func (m *chatMem) Attachments(context.Context, string, string) ([]domain.Attachment, error) {
	return nil, nil
}
func (m *chatMem) Download(context.Context, string, string, string) ([]byte, domain.Attachment, error) {
	return nil, domain.Attachment{}, nil
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func sampleChats() []domain.Chat {
	chats := []domain.Chat{{ID: "chat-1", Type: "oneOnOne", Members: []domain.Person{{Name: "Alice"}}}}
	for i := 0; i < 20; i++ {
		chats = append(chats, domain.Chat{ID: "pad", Type: "oneOnOne", Members: []domain.Person{{Name: "Pad"}}})
		chats[len(chats)-1].ID = "pad-" + itoa(i)
	}
	return append(chats,
		domain.Chat{ID: "chat-ajay", Type: "oneOnOne", Members: []domain.Person{{Name: "Ajay Kumar", Address: "ajay@example.com"}}},
		domain.Chat{ID: "chat-group-ajay", Type: "group", Topic: "Project", Members: []domain.Person{{Name: "Ajay Kumar"}}},
		domain.Chat{ID: "chat-noc-dev", Type: "group", Topic: "NOC-Dev"},
		domain.Chat{ID: "chat-2", Type: "group", Topic: "NOC"},
	)
}

func TestFindPersonSkipsGroupsAndOffPage(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "user@example.com"}
	q, _ := ParseQuery("Ajay", false)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Count != 1 || r.Items[0].ID != "chat-ajay" || r.Incomplete {
		t.Fatalf("%+v", r)
	}
	for _, c := range r.Items {
		if c.ID == "chat-group-ajay" {
			t.Fatal("group included")
		}
	}
}

func TestFindSeveralAjays(t *testing.T) {
	chats := append(sampleChats(), domain.Chat{
		ID: "chat-ajay-b", Type: "oneOnOne",
		Members: []domain.Person{{Name: "Ajay Singh", Address: "ajay.singh@example.com"}},
	})
	st := &chatMem{chats: chats}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true}
	q, _ := ParseQuery("Ajay", false)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil || r.Count != 2 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestFindSelfReturnsNotes(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "Mason.Huemmer@Sesami.io"}
	q, _ := ParseQuery("Mason Huemmer", false)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Count != 1 || r.Items[0].ID != SelfChatID || r.Incomplete {
		t.Fatalf("%+v", r)
	}
}

func TestFindSelfByLocalPart(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "user@example.com"}
	q, _ := ParseQuery("user", false)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil || r.Count != 1 || r.Items[0].ID != SelfChatID {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestFindNone(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true}
	q, _ := ParseQuery("Nobody", false)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil || r.Count != 0 || r.Incomplete {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestFindGroupNOC(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true}
	q, _ := ParseQuery("NOC", true)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, c := range r.Items {
		if strings.EqualFold(c.Type, "oneOnOne") {
			t.Fatal("1:1 in group find")
		}
		ids[c.ID] = true
	}
	if !ids["chat-noc-dev"] && !ids["chat-2"] {
		t.Fatalf("%+v", r)
	}
}

func TestFindGroupWithAjay(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	sess := domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true}
	q, _ := ParseQuery("group with Ajay", false)
	r, err := Find(context.Background(), st, sess, q, 0)
	if err != nil || r.Intent != IntentGroup {
		t.Fatal(r, err)
	}
	found := false
	for _, c := range r.Items {
		if c.ID == "chat-group-ajay" {
			found = true
		}
		if c.ID == "chat-ajay" {
			t.Fatal("1:1")
		}
	}
	if !found {
		t.Fatalf("%+v", r)
	}
}
