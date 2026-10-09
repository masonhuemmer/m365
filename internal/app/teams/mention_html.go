package teams

import (
	"html"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"

	"github.com/masonhuemmer/m365/internal/domain"
)

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

// aheadLimit bounds how far past a name we read: display names and aliases are
// short, and reading the rest of a long line for every "@" would be quadratic.
const aheadLimit = 256

// aheadText is the visible text that follows token k on the same line, read
// through inline tags, so a name split by <strong> is seen whole. Runs of
// whitespace collapse to one space, as HTML shows them. The bool is true when
// the limit cut the text short, so the caller cannot treat its end as the end
// of the line.
func aheadText(toks []htmlToken, k int) (string, bool) {
	var b strings.Builder
	space := false
	for visited, t := range toks[k+1:] {
		if visited >= aheadLimit || b.Len() >= aheadLimit {
			return b.String(), true
		}
		switch t.tt {
		case xhtml.StartTagToken, xhtml.EndTagToken, xhtml.SelfClosingTagToken:
			if breaksLine(t.name) {
				return b.String(), false
			}
		case xhtml.TextToken:
			// Decode only as much of a long text token as the budget can use.
			raw := t.raw
			if len(raw) > 4*aheadLimit {
				raw = raw[:runeStart(raw, 4*aheadLimit)]
			}
			for _, r := range html.UnescapeString(raw) {
				if b.Len() >= aheadLimit {
					return b.String(), true
				}
				if unicode.IsSpace(r) {
					if !space {
						b.WriteByte(' ')
					}
					space = true
					continue
				}
				space = false
				b.WriteRune(r)
			}
			if len(raw) < len(t.raw) {
				return b.String(), true
			}
		}
	}
	return b.String(), false
}

// window is the text a name is resolved against: the rest of this text node
// plus what follows it on the line, both bounded.
func window(rest string, ahead func() (string, bool)) (string, bool) {
	if len(rest) > aheadLimit {
		return rest[:runeStart(rest, aheadLimit)], true
	}
	more, truncated := ahead()
	return rest + more, truncated
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

// runeStart moves n back to the start of a rune, so a cut never splits one.
func runeStart(s string, n int) int {
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return n
}
