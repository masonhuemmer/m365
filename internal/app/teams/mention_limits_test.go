package teams

import (
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Tenth review round.

func TestMentionAliasEndingInUnderscoreOrDash(t *testing.T) {
	st := chatOf(person("u-o", "Olivia Smith", "ops_@example.com"), person("u-ops", "Ops", "ops@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@ops_ please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-o" {
		t.Fatalf("alias: %+v", st.last.Mentions)
	}
	st = chatOf(person("u-o", "Olivia Smith", "ops_@example.com"), person("u-ops", "Ops", "ops@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@Ops please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-ops" {
		t.Fatalf("plain name: %+v", st.last.Mentions)
	}
}

// The lookahead is bounded, so a window that is cut off must not be read as
// "the name ended".
func TestMentionNeverGuessesWhenTheLookaheadIsTruncated(t *testing.T) {
	members := []domain.Person{person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com")}
	for name, body := range map[string]string{
		"padding":    "<p>@Bo<strong>" + strings.Repeat(" ", 400) + "</strong>Chen</p>",
		"empty tags": "<p>@Bo" + strings.Repeat("<i></i>", 400) + "Chen</p>",
	} {
		st := chatOf(members...)
		sendHTML(t, st, body)
		if len(st.last.Mentions) != 0 {
			t.Fatalf("%s: guessed %+v", name, st.last.Mentions)
		}
	}
}

// Twelfth review round.

func TestMentionUnknownAliasEndingInUnderscoreStaysText(t *testing.T) {
	st := chatOf(person("u-ops", "Ops", "ops@example.com"), person("u-me", "Me", "self@example.com"))
	out, err := send(t, st, "@ops_ please look")
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if _, ok := m["mentions"]; ok {
		t.Fatalf("notified someone for a different spelling: %v", m["mentions"])
	}
	if got := m["unresolved_mentions"]; !equalStrings(got, []string{"@ops_"}) {
		t.Fatalf("unresolved %v", got)
	}
}

func TestMentionNeverGuessesWhenPaddingHidesALongerName(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	sendHTML(t, st, "<p>@Bo"+strings.Repeat(" ", 251)+"Chen</p>")
	if len(st.last.Mentions) != 0 {
		t.Fatalf("guessed %+v", st.last.Mentions)
	}
}

// Thirteenth review round.

func TestMentionAliasWithApostropheSDoesNotLookLikeAPossessive(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Boris Chen", "bo's-team@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@bo's-team please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bc" {
		t.Fatalf("%+v", st.last.Mentions)
	}
}

// Fourteenth review round.

func TestMentionPlusAddressedAlias(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Boris Chen", "bo+ops@partner.example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@bo+ops please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bc" {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

func TestMentionDuplicateNameErrorSuggestsAnAlias(t *testing.T) {
	st := chatOf(person("u1", "Ajay Mathew", "ajay.m1@example.com"), person("u2", "Ajay Mathew", "ajay.m2@example.com"), person("u-me", "Me", "self@example.com"))
	_, err := send(t, st, "@Ajay Mathew look", func(in *SendInput) { in.DryRun = false })
	if err == nil || strings.Contains(err.Error(), "chat id") || !strings.Contains(err.Error(), "email name") {
		t.Fatalf("error should suggest an email name, not a chat id: %v", err)
	}
}

func TestMentionAliasEndingInPlus(t *testing.T) {
	st := chatOf(person("u-ops", "Ops", "ops@example.com"), person("u-o", "Olivia Smith", "ops+@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@ops+ please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-o" {
		t.Fatalf("%+v", st.last.Mentions)
	}
}
