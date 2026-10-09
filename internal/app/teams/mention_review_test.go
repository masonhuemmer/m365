package teams

import (
	"strings"
	"testing"
	"time"

	"github.com/masonhuemmer/m365/internal/domain"
)

// Review findings: each case below mentioned the wrong person, corrupted HTML,
// or missed a valid mention before the HTML was tokenized properly.

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

// Second review round.

func TestMentionAliasWithRepeatedSeparatorsBeatsShorterName(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo__chen@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@bo__chen look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bc" {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

// A name that visibly carries on after an inline tag is not the shorter name.
func TestMentionNeverGuessesWhenFormattingSplitsTheName(t *testing.T) {
	members := []domain.Person{person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com")}
	for _, body := range []string{`<p>@Bo<strong>.chen</strong> hi</p>`, `<p>@Bo<em>b</em> hi</p>`} {
		st := chatOf(members...)
		sendHTML(t, st, body)
		if len(st.last.Mentions) != 0 {
			t.Fatalf("%s mentioned %+v", body, st.last.Mentions)
		}
	}
	st := chatOf(members...)
	sendHTML(t, st, `<p>@Bo<strong> please</strong></p>`)
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bo" {
		t.Fatalf("a space after the tag ends the name: %+v", st.last.Mentions)
	}
}

func TestMentionAfterHrAndPre(t *testing.T) {
	for _, body := range []string{`status<hr>@Ajay`, `<pre>command</pre>@Ajay`} {
		st := group()
		sendHTML(t, st, body)
		if len(st.last.Mentions) != 1 {
			t.Fatalf("%s: %q", body, st.last.Rendered.Content)
		}
	}
}

// Third review round.

func TestMentionNeverGuessesWhenASeparatorEndsTheTextNode(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	sendHTML(t, st, `<p>@Bo.<strong>chen</strong> please look</p>`)
	if len(st.last.Mentions) != 0 {
		t.Fatalf("mentioned %+v", st.last.Mentions)
	}
	st = chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-me", "Me", "self@example.com"))
	sendHTML(t, st, `<p>Thanks @Bo. Please look</p>`)
	if len(st.last.Mentions) != 1 {
		t.Fatalf("a sentence-ending period must not block the mention: %q", st.last.Rendered.Content)
	}
}

func TestMentionFirstNamesWithApostropheAndCombiningMarks(t *testing.T) {
	st := chatOf(person("u-d", "D'Arcy Smith", ""), person("u-k", "दीपक शर्मा", ""), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@D'Arcy and @दीपक please look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 2 || st.last.Mentions[0].UserID != "u-d" || st.last.Mentions[1].UserID != "u-k" {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

func TestMentionPossessiveStillMentionsThePerson(t *testing.T) {
	for _, text := range []string{"@Ajay's change is in", "@Ajay’s change is in"} {
		st := group()
		if _, err := send(t, st, text, func(in *SendInput) { in.DryRun = false }); err != nil {
			t.Fatal(err)
		}
		if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-ajay" {
			t.Fatalf("%q: %+v", text, st.last.Mentions)
		}
	}
}

// Fourth review round.

func TestMentionNeverGuessesWhenFormattingSplitsAFullName(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	sendHTML(t, st, `<p>@Bo<strong> Chen</strong> please look</p>`)
	if len(st.last.Mentions) != 0 {
		t.Fatalf("mentioned %+v", st.last.Mentions)
	}
}

func TestMentionApostropheInsideANameBeatsShorterName(t *testing.T) {
	st := chatOf(person("u-al", "Al", "al@example.com"), person("u-am", "Al'Amin Khan", "al.amin@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@Al'Amin look", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-am" {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

// Fifth review round.

func TestMentionSpaceBeforeInlineTagCannotPickTheShorterName(t *testing.T) {
	members := []domain.Person{person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com")}
	st := chatOf(members...)
	if _, err := send(t, st, "@Bo **Chen** please look", func(in *SendInput) { in.DryRun = false; in.MD = true }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 0 {
		t.Fatalf("guessed %+v from %q", st.last.Mentions, st.last.Rendered.Content)
	}
	st = chatOf(members...)
	if _, err := send(t, st, "@Bo **please** look", func(in *SendInput) { in.DryRun = false; in.MD = true }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bo" {
		t.Fatalf("a later word is not part of the name: %+v", st.last.Mentions)
	}
}

func TestMentionPossessiveAfterAFullName(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@Bo Chen's change is in", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bc" {
		t.Fatalf("%+v %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

// Sixth review round: a later segment of the name crosses an inline tag.
func TestMentionNeverGuessesWhenALaterSegmentCrossesATag(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@Bo Ch**en** please look", func(in *SendInput) { in.DryRun = false; in.MD = true }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 0 {
		t.Fatalf("guessed %+v from %q", st.last.Mentions, st.last.Rendered.Content)
	}
}

// Seventh review round: HTML collapses whitespace, so the name must too.
func TestMentionFullNameAcrossCollapsedWhitespace(t *testing.T) {
	members := []domain.Person{person("u-a", "Ann Marie", ""), person("u-aj", "Ann Marie Jones", ""), person("u-me", "Me", "self@example.com")}
	for _, body := range []string{"<p>@Ann Marie  Jones please look</p>", "<p>@Ann Marie\nJones please look</p>", "<p>@Ann Marie&nbsp;Jones please look</p>"} {
		st := chatOf(members...)
		sendHTML(t, st, body)
		if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-aj" {
			t.Fatalf("%q: %+v", body, st.last.Mentions)
		}
	}
}

// Eighth review round: capitalization must not change who is notified.
func TestMentionUppercasePossessives(t *testing.T) {
	st := chatOf(person("u-bo", "Bo", "bo@example.com"), person("u-bc", "Bo Chen", "bo.chen@example.com"), person("u-me", "Me", "self@example.com"))
	if _, err := send(t, st, "@BO CHEN'S update is in", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-bc" {
		t.Fatalf("full name: %+v", st.last.Mentions)
	}
	st = group()
	if _, err := send(t, st, "@AJAY’S change is in", func(in *SendInput) { in.DryRun = false }); err != nil {
		t.Fatal(err)
	}
	if len(st.last.Mentions) != 1 || st.last.Mentions[0].UserID != "u-ajay" {
		t.Fatalf("first name: %+v", st.last.Mentions)
	}
}

// Ninth review round: a unique name split by formatting is not an ambiguity.
func TestMentionSplitUniqueNameIsNotReportedAsAmbiguous(t *testing.T) {
	st := chatOf(person("u-am", "Ajay Mathew", ""), person("u-as", "Ajay Singh", ""), person("u-me", "Me", "self@example.com"))
	out, err := send(t, st, "@Ajay **Singh** please look", func(in *SendInput) { in.MD = true })
	if err != nil {
		t.Fatalf("blocked the message: %v", err)
	}
	m := out.(map[string]any)
	if _, ok := m["mentions"]; ok {
		t.Fatalf("mentioned %v", m["mentions"])
	}
	if got := m["unresolved_mentions"]; !equalStrings(got, []string{"@Ajay Singh"}) {
		t.Fatalf("unresolved %v", got)
	}
}

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

// Many mentions in one long paragraph must stay linear.
func TestMentionManyInOneParagraphIsFast(t *testing.T) {
	st := group()
	body := "<p>" + strings.Repeat("@Bo <strong>x</strong> ", 30000) + "</p>"
	start := time.Now()
	sendHTML(t, st, body)
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("30000 mentions took %v", d)
	}
	if len(st.last.Mentions) != 1 {
		t.Fatalf("mentions %d", len(st.last.Mentions))
	}
}
