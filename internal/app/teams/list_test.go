package teams

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

type stub struct {
	chats domain.ChatPage
	chat  domain.Chat
	msgs  domain.ChatMessagePage
	err   error
	sent  int
	last  SendInput
}

func (s *stub) ListChats(context.Context, int, string) (domain.ChatPage, error) {
	return s.chats, s.err
}
func (s *stub) ListVisibleChats(ctx context.Context, top int, page string) (domain.ChatPage, error) {
	return s.ListChats(ctx, top, page)
}
func (s *stub) GetChat(context.Context, string) (domain.Chat, error) { return s.chat, s.err }
func (s *stub) Messages(context.Context, MessageQuery) (domain.ChatMessagePage, error) {
	return s.msgs, s.err
}
func (s *stub) Send(_ context.Context, in SendInput) (string, error) {
	s.sent++
	s.last = in
	return "m1", s.err
}
func (s *stub) Watch(context.Context, WatchQuery) ([]domain.WatchEvent, error) {
	return nil, s.err
}
func (s *stub) Attachments(context.Context, string, string) ([]domain.Attachment, error) {
	return nil, s.err
}
func (s *stub) Download(context.Context, string, string, string) ([]byte, domain.Attachment, error) {
	return []byte("synthetic-ok"), domain.Attachment{ID: "att-1", Name: "note.txt", Size: 12}, s.err
}

func sess() domain.Session {
	return domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true}
}

func TestListTopAndAuth(t *testing.T) {
	st := &stub{}
	_, err := List(context.Background(), st, domain.SignedOut(), 0, "")
	if domain.ExitOf(err) != domain.ExitAuth {
		t.Fatal(err)
	}
	_, err = List(context.Background(), st, sess(), 51, "")
	if domain.ExitOf(err) != domain.ExitUsage {
		t.Fatal(err)
	}
	_, err = Get(context.Background(), st, sess(), "")
	if domain.ExitOf(err) != domain.ExitUsage {
		t.Fatal(err)
	}
}
