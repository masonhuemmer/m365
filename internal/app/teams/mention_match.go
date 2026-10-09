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
	raw := mentionWord.FindString(rest)
	// "_" and "-" are valid at the end of an email name (ops_@example.com), so a
	// token that ends in one is an alias: it matches exactly or not at all, and
	// is never shortened to a different spelling.
	if tok := strings.TrimRight(raw, ".'\u2019"); strings.HasSuffix(tok, "_") || strings.HasSuffix(tok, "-") || strings.HasSuffix(tok, "+") {
		if found := m.byLocal(tok); len(found) > 0 {
			return m.pick(found, tok)
		}
		m.noteUnresolved("@" + tok)
		return nil, 0, nil
	}
	if p, n, err := m.byFullName(rest); p != nil || err != nil {
		return p, n, err
	}
	word := strings.TrimRight(raw, "._-'\u2019")
	if word == "" {
		return nil, 0, nil
	}
	found := m.byWord(word)
	if len(found) == 0 { // "@Ajay's": the name is Ajay
		if trimmed, ok := cutPossessiveSuffix(word); ok && trimmed != "" {
			if found = m.byWord(trimmed); len(found) > 0 {
				word = trimmed
			}
		}
	}
	if len(found) == 0 {
		m.noteUnresolved("@" + word)
		return nil, 0, nil
	}
	return m.pick(found, word)
}

// pick returns the one member found for word, or fails if there are several.
func (m *mentioner) pick(found []domain.Person, word string) (*domain.Person, int, error) {
	if len(found) == 1 {
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
		n, ok := foldPrefix(rest, p.Name)
		if !ok || wordContinues(rest[n:]) {
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
	return nil, 0, domain.Usagef("%q matches several chat members (%s); write an email name (@name.surname) instead", best[0].Name, names(best))
}

// wordContinues is true when s carries on the same @ token: more letters or
// digits, or alias separators (. _ - and apostrophes) followed by one, as in
// "@bo.chen", "@bo__chen" or "@Al'Amin". It lets the email-alias match win over a shorter full name.
// foldPrefix reports whether rest starts with name, ignoring case and treating
// any run of whitespace (spaces, newlines, non-breaking spaces) as one space,
// the way HTML shows it. It returns how many bytes of rest the name covers.
func foldPrefix(rest, name string) (int, bool) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return 0, false
	}
	i := 0
	for _, want := range name {
		if unicode.IsSpace(want) {
			j := i
			for j < len(rest) {
				r, w := utf8.DecodeRuneInString(rest[j:])
				if !unicode.IsSpace(r) {
					break
				}
				j += w
			}
			if j == i {
				return 0, false
			}
			i = j
			continue
		}
		if i >= len(rest) {
			return 0, false
		}
		r, w := utf8.DecodeRuneInString(rest[i:])
		if !strings.EqualFold(string(r), string(want)) {
			return 0, false
		}
		i += w
	}
	return i, true
}

func wordContinues(s string) bool {
	if after, ok := cutPossessivePrefix(s); ok && !startsWord(strings.TrimLeft(after, "._-+'\u2019")) {
		return false // "@Bo Chen's change": the 's is not part of the name
	}
	return startsWord(strings.TrimLeft(s, "._-+'\u2019"))
}

func startsWord(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return s != "" && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r))
}

// byWord matches one word: an email name part (@ajay.mathew) first, then a
// first name (@Ajay).
func (m *mentioner) byWord(word string) []domain.Person {
	if local := m.byLocal(word); len(local) > 0 {
		return local
	}
	var first []domain.Person
	for _, p := range m.members {
		if f := strings.Fields(p.Name); len(f) > 0 && strings.EqualFold(f[0], word) {
			first = appendOnce(first, p)
		}
	}
	return first
}

// byLocal matches the part of a member's email before the @.
func (m *mentioner) byLocal(word string) []domain.Person {
	var local []domain.Person
	for _, p := range m.members {
		if at := strings.IndexByte(p.Address, '@'); at > 0 && strings.EqualFold(p.Address[:at], word) {
			local = appendOnce(local, p)
		}
	}
	return local
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
	if m.seen[token] {
		return
	}
	if m.seen == nil {
		m.seen = map[string]bool{}
	}
	m.seen[token] = true
	m.unresolved = append(m.unresolved, token)
}

// cutPossessivePrefix strips a leading 's or ’s, in any case.
func cutPossessivePrefix(s string) (string, bool) {
	for _, apostrophe := range []string{"'", "\u2019"} {
		if after, ok := strings.CutPrefix(s, apostrophe); ok && after != "" && (after[0] == 's' || after[0] == 'S') {
			return after[1:], true
		}
	}
	return s, false
}

// cutPossessiveSuffix strips a trailing 's or ’s, in any case.
func cutPossessiveSuffix(s string) (string, bool) {
	for _, apostrophe := range []string{"'", "\u2019"} {
		n := len(apostrophe) + 1
		if len(s) >= n && strings.HasPrefix(s[len(s)-n:], apostrophe) && strings.EqualFold(s[len(s)-1:], "s") {
			return s[:len(s)-n], true
		}
	}
	return s, false
}
