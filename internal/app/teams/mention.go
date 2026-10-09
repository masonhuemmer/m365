package teams

import (
	"context"
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Mention is one person tagged in a message: the <at id> in the body and the
// matching entry in Graph's mentions array.
type Mention struct {
	ID     int
	Name   string
	UserID string
}

var (
	htmlTag      = regexp.MustCompile(`<[^>]*>`)
	mentionStart = regexp.MustCompile(`(^|[\s>(])@[\p{L}\p{N}]`)
	mentionWord  = regexp.MustCompile(`^[\p{L}\p{N}._-]+`)
)

// applyMentions turns a typed @Name into a real mention of a member of the
// chat. Only an @ at the start of a word counts, so addresses like
// ajay@example.com are left alone, and names are matched against the chat's
// members only. A name that matches nobody stays plain text and is reported;
// one that matches several members fails before anything is sent.
func applyMentions(ctx context.Context, st Store, in *SendInput) error {
	if in.ChatID == SelfChatID || !mentionStart.MatchString(in.Rendered.Content) {
		return nil
	}
	chat, err := st.GetChat(ctx, in.ChatID)
	if err != nil {
		return err
	}
	mp := &mentioner{members: chat.Members, ids: map[string]int{}}
	content, err := mp.rewrite(in.Rendered.Content)
	if err != nil {
		return err
	}
	in.Rendered.Content, in.Mentions, in.UnresolvedMentions = content, mp.mentions, mp.unresolved
	return nil
}

type mentioner struct {
	members    []domain.Person
	ids        map[string]int
	mentions   []Mention
	unresolved []string
}

// rewrite walks the HTML and mentions people in text only: never inside tags,
// code blocks or links.
func (m *mentioner) rewrite(content string) (string, error) {
	var out strings.Builder
	skip, last := 0, 0
	for _, loc := range htmlTag.FindAllStringIndex(content, -1) {
		text, err := m.text(content[last:loc[0]], skip)
		if err != nil {
			return "", err
		}
		out.WriteString(text)
		tag := content[loc[0]:loc[1]]
		out.WriteString(tag)
		skip += skipDelta(tag)
		last = loc[1]
	}
	text, err := m.text(content[last:], skip)
	if err != nil {
		return "", err
	}
	out.WriteString(text)
	return out.String(), nil
}

// skipDelta counts how deep we are inside <code>, <pre> and <a>.
func skipDelta(tag string) int {
	name := strings.ToLower(strings.TrimLeft(strings.Trim(tag, "<>/ "), "/"))
	if i := strings.IndexAny(name, " \t\n/"); i >= 0 {
		name = name[:i]
	}
	if name != "code" && name != "pre" && name != "a" {
		return 0
	}
	if strings.HasPrefix(tag, "</") {
		return -1
	}
	return 1
}

func (m *mentioner) text(text string, skip int) (string, error) {
	if skip > 0 || !strings.Contains(text, "@") {
		return text, nil
	}
	plain := html.UnescapeString(text)
	var out strings.Builder
	last, changed := 0, false
	for i := 0; i < len(plain); i++ {
		if plain[i] != '@' || !atWordStart(plain, i) {
			continue
		}
		who, n, err := m.match(plain[i+1:])
		if err != nil {
			return "", err
		}
		if who == nil {
			continue
		}
		out.WriteString(html.EscapeString(plain[last:i]))
		out.WriteString(m.tag(*who))
		last, i, changed = i+1+n, i+n, true
	}
	if !changed {
		return text, nil
	}
	out.WriteString(html.EscapeString(plain[last:]))
	return out.String(), nil
}

func atWordStart(s string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return unicode.IsSpace(r) || r == '('
}

func (m *mentioner) tag(p domain.Person) string {
	id, seen := m.ids[p.ID]
	if !seen {
		id = len(m.mentions)
		m.ids[p.ID] = id
		m.mentions = append(m.mentions, Mention{ID: id, Name: p.Name, UserID: p.ID})
	}
	return `<at id="` + strconv.Itoa(id) + `">` + html.EscapeString(p.Name) + `</at>`
}
