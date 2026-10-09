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

// Letters, combining marks, digits, apostrophes and alias separators: D'Arcy, दीपक, bo.chen.
var mentionWord = regexp.MustCompile(`^[\p{L}\p{M}\p{N}._'\x{2019}-]+`)

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

type htmlToken struct {
	tt   xhtml.TokenType
	raw  string
	name string
}

func tokenize(content string) ([]htmlToken, error) {
	z := xhtml.NewTokenizer(strings.NewReader(content))
	var toks []htmlToken
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			if z.Err() != io.EOF {
				return nil, domain.Usage("could not read the message HTML")
			}
			return toks, nil
		}
		t := htmlToken{tt: tt, raw: string(z.Raw())}
		switch tt {
		case xhtml.StartTagToken, xhtml.EndTagToken, xhtml.SelfClosingTagToken:
			name, _ := z.TagName()
			t.name = string(name)
		}
		toks = append(toks, t)
	}
}

// aheadText is the visible text that follows token k on the same line, read
// through inline tags, so a name split by <strong> is seen whole.
func aheadText(toks []htmlToken, k int) string {
	var b strings.Builder
	for _, t := range toks[k+1:] {
		switch t.tt {
		case xhtml.StartTagToken, xhtml.EndTagToken, xhtml.SelfClosingTagToken:
			if breaksLine(t.name) {
				return b.String()
			}
		case xhtml.TextToken:
			b.WriteString(html.UnescapeString(t.raw))
		}
	}
	return b.String()
}

// rewrite mentions people in text only: never inside tags, comments, code
// blocks or links.
func (m *mentioner) rewrite(content string, match resolver) (string, error) {
	toks, err := tokenize(content)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	skip := 0
	m.prev = '\n'
	for k, t := range toks {
		switch t.tt {
		case xhtml.StartTagToken, xhtml.EndTagToken, xhtml.SelfClosingTagToken:
			skip = max(0, skip+skipDelta(t.tt, t.name))
			if breaksLine(t.name) {
				m.prev = '\n'
			}
			out.WriteString(t.raw)
		case xhtml.TextToken:
			text, err := m.text(t.raw, skip > 0, match, func() string { return aheadText(toks, k) })
			if err != nil {
				return "", err
			}
			out.WriteString(text)
		default:
			out.WriteString(t.raw)
		}
	}
	return out.String(), nil
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
	case "p", "br", "div", "li", "ul", "ol", "tr", "td", "th", "blockquote", "pre", "hr", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

func (m *mentioner) text(raw string, skipped bool, match resolver, ahead func() string) (string, error) {
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
		// A name that runs to the end of this text may continue past an inline tag
		// ("@Bo<strong> Chen</strong>"). Resolve the whole visible token and
		// only mention the person if it is the same one; never guess.
		if ok, err := m.sameWhenWhole(plain[i+1:], who, n, match, ahead); err != nil {
			return "", err
		} else if !ok {
			m.noteUnresolved("@" + plain[i+1:i+1+n])
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

// sameWhenWhole re-resolves a name that reaches the end of its text against
// the visible text that follows it on the line. It is the same mention only if
// the longer text still names the same member in the same number of bytes.
func (m *mentioner) sameWhenWhole(rest string, who *domain.Person, n int, match resolver, ahead func() string) (bool, error) {
	// The token ends inside this node when a space follows it before the node
	// ends: "@Bo, hi", "@Bo. please" or "@Bo please". Trailing space alone, or
	// no space at all ("@Bo." / "@Bo"), leaves it open to the next node.
	tail := rest[n:]
	if sp := strings.IndexFunc(tail, unicode.IsSpace); sp > 0 || (sp == 0 && strings.TrimSpace(tail) != "") {
		return true, nil
	}
	more := ahead()
	if more == "" {
		return true, nil
	}
	whole, wn, err := match(rest + more)
	if err != nil {
		return false, err
	}
	return whole != nil && whole.ID == who.ID && whole.Name == who.Name && wn == n, nil
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
