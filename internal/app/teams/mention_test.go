package teams

import (
	"context"
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func person(id, name, email string) domain.Person {
	return domain.Person{ID: id, Name: name, Address: email}
}

func group() *stub {
	return chatOf(
		person("u-me", "Mason Huemmer", "self@example.com"),
		person("u-ajay", "Ajay Mathew", "ajay.mathew@example.com"),
		person("u-bo", "Bo Chen", "bo@example.com"),
	)
}

func send(t *testing.T, st Store, text string, opts ...func(*SendInput)) (any, error) {
	t.Helper()
	in := SendInput{ChatID: "c", Text: text, DryRun: true}
	for _, o := range opts {
		o(&in)
	}
	return Send(context.Background(), st, selfSess(), in)
}

func TestMentionResolvesTypedNameToMember(t *testing.T) {
	st := group()
	out, err := send(t, st, "@Ajay can you check the deploy?", func(in *SendInput) { in.DryRun = false })
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	got := st.last
	if !strings.Contains(got.Rendered.Content, `<at id="0">Ajay Mathew</at> can you check the deploy?`) {
		t.Fatalf("body %q", got.Rendered.Content)
	}
	if len(got.Mentions) != 1 || got.Mentions[0] != (Mention{ID: 0, Name: "Ajay Mathew", UserID: "u-ajay"}) {
		t.Fatalf("mentions %+v", got.Mentions)
	}
}

func TestMentionFullNameBeatsFirstName(t *testing.T) {
	st := chatOf(person("u1", "Ajay Mathew", ""), person("u2", "Ajay Singh", ""), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@Ajay Singh ok?", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if st.last.Mentions[0].UserID != "u2" || !strings.Contains(st.last.Rendered.Content, `<at id="0">Ajay Singh</at> ok?`) {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

func TestMentionAmbiguousFirstNameFailsBeforeSending(t *testing.T) {
	st := chatOf(person("u1", "Ajay Mathew", ""), person("u2", "Ajay Singh", ""), person("u-me", "Me", "self@example.com"))
	_, err := send(t, st, "@Ajay ok?", func(in *SendInput) { in.DryRun = false })
	if domain.ExitOf(err) != domain.ExitUsage || st.sent != 0 || !strings.Contains(err.Error(), "Ajay Mathew") || !strings.Contains(err.Error(), "Ajay Singh") {
		t.Fatalf("err=%v sent=%d", err, st.sent)
	}
}

func TestMentionUnknownNameStaysPlainTextAndIsReported(t *testing.T) {
	st := group()
	out, err := send(t, st, "@everyone heads up")
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if got := m["unresolved_mentions"]; !equalStrings(got, []string{"@everyone"}) {
		t.Fatalf("unresolved %v", got)
	}
	if _, ok := m["mentions"]; ok {
		t.Fatalf("unexpected mentions %v", m["mentions"])
	}
}

func TestMentionIgnoresEmailsAndDoesNotReadTheChat(t *testing.T) {
	st := &failingChat{stub: group()}
	if _, err := send(t, st, "mail ajay@example.com about it", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatalf("chat should not be read for a message without an @name: %v", err)
	}
}

func TestMentionSkipsCodeAndLinks(t *testing.T) {
	st := group()
	body := `<p>run <code>@Ajay</code> and see <a href="https://x.test/@Bo">@Bo</a></p>`
	if _, err := send(t, st, body, func(in *SendInput) { in.DryRun = false; in.HTML = true }); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(st.last.Rendered.Content, "<at ") || len(st.last.Mentions) != 0 {
		t.Fatalf("mentioned inside code or a link: %q", st.last.Rendered.Content)
	}
}

func TestMentionSamePersonTwiceSharesOneEntry(t *testing.T) {
	st := group()
	if _, err := send(t, st, "@Ajay first, then @Ajay again", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || strings.Count(st.last.Rendered.Content, `<at id="0">Ajay Mathew</at>`) != 2 {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

func TestMentionNotesIsLeftAlone(t *testing.T) {
	st := &failingChat{stub: group()}
	if _, err := send(t, st, "@Ajay note to self", func(in *SendInput) { in.DryRun = false; in.ChatID = SelfChatID }); err != nil {
		t.Fatalf("Notes has no members to read: %v", err)
	}
}

func TestMentionDryRunListsWhoIsNotified(t *testing.T) {
	out, err := send(t, group(), "@Bo and @Ajay Mathew please look")
	if err != nil {
		t.Fatal(err)
	}
	if got := out.(map[string]any)["mentions"]; !equalStrings(got, []string{"Bo Chen", "Ajay Mathew"}) {
		t.Fatalf("mentions %v", got)
	}
}

// failingChat fails the test run if the chat is read.
type failingChat struct{ *stub }

func (f *failingChat) GetChat(context.Context, string) (domain.Chat, error) {
	return domain.Chat{}, domain.Service("chat must not be read")
}

func equalStrings(v any, want []string) bool {
	got, ok := v.([]string)
	if !ok || len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// Review findings: each case below mentioned the wrong person, corrupted HTML,
// or missed a valid mention before the HTML was tokenized properly.

func sendHTML(t *testing.T, st *stub, body string) {
	t.Helper()
	if _, err := send(t, st, body, func(in *SendInput) { in.DryRun = false; in.HTML = true }); err != nil {
		t.Fatal(err)
	}
}

func TestMentionEmailAliasBeatsShorterFullName(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@bo.chen please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bc" {
		t.Fatalf("mentions %+v body %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

func TestMentionLeavesHTMLCommentsAlone(t *testing.T) {
	st := group()
	body := `<!-- x > @Ajay --><p>Hello</p>`
	sendHTML(t, st, body)
	if st.last.Rendered.Content != body || len(st.last.Mentions) != 0 {
		t.Fatalf("comment changed: %q %+v", st.last.Rendered.Content, st.last.Mentions)
	}
}

func TestMentionWordBoundaryCarriesAcrossInlineTags(t *testing.T) {
	st := group()
	sendHTML(t, st, `<p>mail user<strong>@Ajay</strong>.example</p>`)
	if len(st.last.Mentions) != 0 {
		t.Fatalf("mentioned inside an email address: %q", st.last.Rendered.Content)
	}
	st = group()
	sendHTML(t, st, `<p>hi <strong>@Ajay</strong></p><p>@Bo</p>`)
	if len(st.last.Mentions) != 2 {
		t.Fatalf("missed mentions after a space or a block tag: %q", st.last.Rendered.Content)
	}
}

func TestMentionSkipsLinksWithWhitespaceInTheTag(t *testing.T) {
	st := group()
	sendHTML(t, st, "<p><a\r\nhref=\"https://x.test\">@Ajay</a> and @Bo</p>")
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bo" {
		t.Fatalf("mentions %+v body %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

func TestMentionAfterNonBreakingSpace(t *testing.T) {
	st := group()
	sendHTML(t, st, `<p>Hello&nbsp;@Ajay</p>`)
	if len(st.last.Mentions) != 1 {
		t.Fatalf("html nbsp: %q", st.last.Rendered.Content)
	}
	st = group()
	if _, err := send(t, st, "Hello @Ajay", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 {
		t.Fatalf("plain nbsp: %q", st.last.Rendered.Content)
	}
}
