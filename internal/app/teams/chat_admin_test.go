package teams

import (
	"context"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func (s *stub) CreateChat(_ context.Context, topic string, _ []string) (domain.Chat, error) {
	return domain.Chat{ID: "chat-new", Topic: topic, Type: "group"}, s.err
}
func (s *stub) RenameChat(context.Context, string, string) error { return s.err }
func (s *stub) HideChat(context.Context, string, bool) error     { return s.err }

func (m *chatMem) CreateChat(_ context.Context, topic string, _ []string) (domain.Chat, error) {
	return domain.Chat{ID: "chat-new", Topic: topic, Type: "group"}, nil
}
func (m *chatMem) RenameChat(context.Context, string, string) error { return nil }
func (m *chatMem) HideChat(context.Context, string, bool) error     { return nil }

// adminMem records the calls that change chats.
type adminMem struct {
	chatMem
	created []string
	renamed []string
	hidden  []string
}

func (m *adminMem) CreateChat(_ context.Context, topic string, members []string) (domain.Chat, error) {
	m.created = append(m.created, topic+"|"+strings.Join(members, ","))
	return domain.Chat{ID: "chat-new", Topic: topic, Type: "group"}, nil
}
func (m *adminMem) RenameChat(_ context.Context, id, topic string) error {
	m.renamed = append(m.renamed, id+"|"+topic)
	return nil
}

func (m *adminMem) HideChat(_ context.Context, id string, hidden bool) error {
	m.hidden = append(m.hidden, id+"|"+map[bool]string{true: "hide", false: "unhide"}[hidden])
	return nil
}

func adminSess() domain.Session {
	return domain.Session{SignedIn: true, SessionUsable: true, TeamsConsented: true, Account: "me@example.com"}
}

func TestCreateGroupDryRunAndSend(t *testing.T) {
	st := &adminMem{}
	in := CreateInput{Name: "  NOC Weekly ", Members: []string{"a@example.com", "01234567-89ab-cdef-0123-456789abcdef"}, DryRun: true}
	out, err := CreateGroup(context.Background(), st, adminSess(), in)
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["dry_run"] != true || m["name"] != "NOC Weekly" || len(st.created) != 0 {
		t.Fatalf("%v created=%v", m, st.created)
	}
	in.DryRun = false
	out, err = CreateGroup(context.Background(), st, adminSess(), in)
	if err != nil || out.(map[string]any)["id"] != "chat-new" || len(st.created) != 1 {
		t.Fatal(err, out, st.created)
	}
}

func TestCreateGroupRejectsBadInput(t *testing.T) {
	good := []string{"a@example.com", "b@example.com"}
	cases := map[string]CreateInput{
		"no name":         {Members: good},
		"newline in name": {Name: "a\nb", Members: good},
		"long name":       {Name: strings.Repeat("x", MaxTopicLength+1), Members: good},
		"one member":      {Name: "n", Members: good[:1]},
		"display name":    {Name: "n", Members: []string{"Ajay", "b@example.com"}},
		"named address":   {Name: "n", Members: []string{"Ajay <a@example.com>", "b@example.com"}},
		"duplicate":       {Name: "n", Members: []string{"a@example.com", "A@Example.com"}},
		"self":            {Name: "n", Members: []string{"ME@example.com", "b@example.com"}},
	}
	for label, in := range cases {
		st := &adminMem{}
		if _, err := CreateGroup(context.Background(), st, adminSess(), in); domain.ExitOf(err) != domain.ExitUsage || len(st.created) != 0 {
			t.Errorf("%s: err=%v created=%v", label, err, st.created)
		}
	}
	if _, err := CreateGroup(context.Background(), &adminMem{}, domain.SignedOut(), CreateInput{Name: "n", Members: good}); domain.ExitOf(err) != domain.ExitAuth {
		t.Fatal(err)
	}
}

func TestRenameOnlyGroupChats(t *testing.T) {
	st := &adminMem{chatMem: chatMem{chats: []domain.Chat{
		{ID: "g1", Type: "group", Topic: "Old"},
		{ID: "o1", Type: "oneOnOne"},
		{ID: "m1", Type: "meeting"},
	}}}
	out, err := Rename(context.Background(), st, adminSess(), "g1", "New", true)
	if err != nil || out.(map[string]any)["old_name"] != "Old" || len(st.renamed) != 0 {
		t.Fatal(err, out, st.renamed)
	}
	if _, err := Rename(context.Background(), st, adminSess(), "g1", "New", false); err != nil || st.renamed[0] != "g1|New" {
		t.Fatal(err, st.renamed)
	}
	for _, id := range []string{"o1", "m1"} {
		if _, err := Rename(context.Background(), st, adminSess(), id, "New", false); domain.ExitOf(err) != domain.ExitUsage {
			t.Fatalf("%s renamed: %v", id, err)
		}
	}
	if _, err := Rename(context.Background(), st, adminSess(), "nope", "New", false); domain.ExitOf(err) != domain.ExitNotFound {
		t.Fatal(err)
	}
	if _, err := Rename(context.Background(), st, adminSess(), "g1", " ", false); domain.ExitOf(err) != domain.ExitUsage {
		t.Fatal(err)
	}
	if len(st.renamed) != 1 {
		t.Fatalf("extra renames: %v", st.renamed)
	}
}

func TestSetHiddenDryRunThenApply(t *testing.T) {
	st := &adminMem{chatMem: chatMem{chats: []domain.Chat{{ID: "g1", Type: "group", Topic: "Test"}, {ID: "o1", Type: "oneOnOne"}}}}
	out, err := SetHidden(context.Background(), st, adminSess(), "g1", true, true)
	if err != nil || out.(map[string]any)["dry_run"] != true || out.(map[string]any)["name"] != "Test" || len(st.hidden) != 0 {
		t.Fatal(err, out, st.hidden)
	}
	if out, err = SetHidden(context.Background(), st, adminSess(), "g1", true, false); err != nil || out.(map[string]any)["hidden"] != true {
		t.Fatal(err, out)
	}
	if out, err = SetHidden(context.Background(), st, adminSess(), "o1", false, false); err != nil || out.(map[string]any)["unhidden"] != true {
		t.Fatal(err, out)
	}
	if len(st.hidden) != 2 || st.hidden[0] != "g1|hide" || st.hidden[1] != "o1|unhide" {
		t.Fatal(st.hidden)
	}
}

func TestSetHiddenRejectsBadInput(t *testing.T) {
	st := &adminMem{}
	if _, err := SetHidden(context.Background(), st, adminSess(), " ", true, false); domain.ExitOf(err) != domain.ExitUsage {
		t.Fatal(err)
	}
	if _, err := SetHidden(context.Background(), st, adminSess(), "missing", true, false); domain.ExitOf(err) != domain.ExitNotFound {
		t.Fatal(err)
	}
	if _, err := SetHidden(context.Background(), st, domain.SignedOut(), "g1", true, false); domain.ExitOf(err) != domain.ExitAuth {
		t.Fatal(err)
	}
	if len(st.hidden) != 0 {
		t.Fatal("hid a chat on a rejected request")
	}
}
