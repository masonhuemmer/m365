package teams

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/masonhuemmer/m365/internal/domain"
)

// match resolves the text after an "@" to one chat member and reports how many
// bytes of it the name used. A nil member means no mention here; a name that
// matches nobody is recorded as unresolved and stays plain text.
func (m *mentioner) match(rest string) (*domain.Person, int, error) {
	if p, n, err := m.byFullName(rest); p != nil || err != nil {
		return p, n, err
	}
	word := strings.TrimRight(mentionWord.FindString(rest), "._-")
	if word == "" {
		return nil, 0, nil
	}
	found := m.byWord(word)
	switch len(found) {
	case 0:
		m.noteUnresolved("@" + word)
		return nil, 0, nil
	case 1:
		return m.mentionable(found[0], len(word))
	}
	return nil, 0, domain.Usagef("@%s matches several chat members (%s); write the full name", word, names(found))
}

func (m *mentioner) mentionable(p domain.Person, n int) (*domain.Person, int, error) {
	if p.ID == "" {
		return nil, 0, domain.Usagef("cannot mention %q: no user id on the chat member", p.Name)
	}
	return &p, n, nil
}

// byFullName finds the member whose whole display name follows the "@", so
// "@Ajay Singh" wins over "@Ajay" when both Ajays are in the chat.
func (m *mentioner) byFullName(rest string) (*domain.Person, int, error) {
	best, bestLen := []domain.Person{}, 0
	for _, p := range m.members {
		n := len(p.Name)
		if n == 0 || n > len(rest) || !strings.EqualFold(rest[:n], p.Name) || wordContinues(rest[n:]) {
			continue
		}
		switch {
		case n > bestLen:
			best, bestLen = []domain.Person{p}, n
		case n == bestLen:
			best = append(best, p)
		}
	}
	switch len(best) {
	case 0:
		return nil, 0, nil
	case 1:
		return m.mentionable(best[0], bestLen)
	}
	return nil, 0, domain.Usagef("%q matches several chat members (%s); pass a chat id", best[0].Name, names(best))
}

// wordContinues is true when s carries on the same @ token: more letters or
// digits, or an alias separator (. _ -) that is followed by one, as in
// "@bo.chen". It lets the email-alias match win over a shorter full name.
func wordContinues(s string) bool {
	r, n := utf8.DecodeRuneInString(s)
	if s == "" {
		return false
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	if r == '.' || r == '_' || r == '-' {
		next, _ := utf8.DecodeRuneInString(s[n:])
		return unicode.IsLetter(next) || unicode.IsDigit(next)
	}
	return false
}

// byWord matches one word: an email name part (@ajay.mathew) first, then a
// first name (@Ajay).
func (m *mentioner) byWord(word string) []domain.Person {
	var local, first []domain.Person
	for _, p := range m.members {
		if at := strings.IndexByte(p.Address, '@'); at > 0 && strings.EqualFold(p.Address[:at], word) {
			local = appendOnce(local, p)
		}
		if f := strings.Fields(p.Name); len(f) > 0 && strings.EqualFold(f[0], word) {
			first = appendOnce(first, p)
		}
	}
	if len(local) > 0 {
		return local
	}
	return first
}

func appendOnce(list []domain.Person, p domain.Person) []domain.Person {
	for _, q := range list {
		if q.ID == p.ID && q.Name == p.Name {
			return list
		}
	}
	return append(list, p)
}

func names(list []domain.Person) string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, p.Name)
	}
	return strings.Join(out, ", ")
}

func (m *mentioner) noteUnresolved(token string) {
	for _, t := range m.unresolved {
		if t == token {
			return
		}
	}
	m.unresolved = append(m.unresolved, token)
}
