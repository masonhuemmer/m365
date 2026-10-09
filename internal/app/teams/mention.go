package teams

import (
	"context"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Mention is one person tagged in a message: the <at id> in the body and the
// matching entry in Graph's mentions array.
type Mention struct {
	ID     int
	Name   string
	UserID string
}

var mentionWord = regexp.MustCompile(`^[\p{L}\p{N}._-]+`)

// resolver maps the text after an "@" to a chat member and the bytes it used.
type resolver func(rest string) (*domain.Person, int, error)

// applyMentions turns a typed @Name into a real mention of a member of the
// chat. Only an @ at the start of a word counts, so addresses like
// ajay@example.com are left alone, and names are matched against the chat's
// members only. A name that matches nobody stays plain text and is reported;
// one that matches several members fails before anything is sent.
func applyMentions(ctx context.Context, st Store, in *SendInput) error {
	if in.ChatID == SelfChatID {
		return nil
	}
	// First pass with no members: only to learn whether the text has a typed
	// @name at all, so a chat is never read for an ordinary message.
	probe := &mentioner{}
	found := false
	if _, err := probe.rewrite(in.Rendered.Content, func(string) (*domain.Person, int, error) {
		found = true
		return nil, 0, nil
	}); err != nil || !found {
		return err
	}
	chat, err := st.GetChat(ctx, in.ChatID)
	if err != nil {
		return err
	}
	mp := &mentioner{members: chat.Members, ids: map[string]int{}}
	content, err := mp.rewrite(in.Rendered.Content, mp.match)
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
	prev       rune // last visible character, carried across inline tags
}

// rewrite tokenizes the HTML and mentions people in text only: never inside
// tags, comments, code blocks or links.
func (m *mentioner) rewrite(content string, match resolver) (string, error) {
	z := xhtml.NewTokenizer(strings.NewReader(content))
	var out strings.Builder
	skip := 0
	m.prev = '\n'
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			if z.Err() != io.EOF {
				return "", domain.Usage("could not read the message HTML")
			}
			return out.String(), nil
		}
		raw := string(z.Raw())
		switch tt {
		case xhtml.StartTagToken, xhtml.EndTagToken, xhtml.SelfClosingTagToken:
			name, _ := z.TagName()
			skip = max(0, skip+skipDelta(tt, string(name)))
			if breaksLine(string(name)) {
				m.prev = '\n'
			}
			out.WriteString(raw)
		case xhtml.TextToken:
			text, err := m.text(raw, skip > 0, match)
			if err != nil {
				return "", err
			}
			out.WriteString(text)
		default:
			out.WriteString(raw)
		}
	}
}

// skipDelta counts how deep we are inside <code>, <pre> and <a>.
func skipDelta(tt xhtml.TokenType, name string) int {
	switch name {
	case "code", "pre", "a":
	default:
		return 0
	}
	switch tt {
	case xhtml.StartTagToken:
		return 1
	case xhtml.EndTagToken:
		return -1
	}
	return 0
}

// breaksLine is true for tags after which a new word starts, so "@" right
// after one is at a word start. Inline tags like <strong> do not break.
func breaksLine(name string) bool {
	switch name {
	case "p", "br", "div", "li", "ul", "ol", "tr", "td", "th", "blockquote", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

func (m *mentioner) text(raw string, skipped bool, match resolver) (string, error) {
	plain := html.UnescapeString(raw)
	defer func() {
		if r, _ := utf8.DecodeLastRuneInString(plain); r != utf8.RuneError && plain != "" {
			m.prev = r
		}
	}()
	if skipped || !strings.Contains(plain, "@") {
		return raw, nil
	}
	var out strings.Builder
	last, changed := 0, false
	for i := 0; i < len(plain); i++ {
		if plain[i] != '@' || !m.atWordStart(plain, i) {
			continue
		}
		who, n, err := match(plain[i+1:])
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
		return raw, nil
	}
	out.WriteString(html.EscapeString(plain[last:]))
	return out.String(), nil
}

func (m *mentioner) atWordStart(s string, i int) bool {
	r := m.prev
	if i > 0 {
		r, _ = utf8.DecodeLastRuneInString(s[:i])
	}
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
