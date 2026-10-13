package cli

import (
	"strings"
	"testing"

	"github.com/masonhuemmer/m365/internal/adapters/graph"
	"github.com/masonhuemmer/m365/internal/app/mail"
	"github.com/masonhuemmer/m365/internal/domain"
)

func TestMailSendBccFlagReachesStoreAndPreview(t *testing.T) {
	d, out, errw := testDeps()
	mem := graph.Seed()
	d.Mail = graph.MailAPI{Memory: mem}
	d.Login = graph.FakeLoginAll(true, true, true, true)
	if code := Run([]string{"m365", "auth", "login"}, d); code != 0 {
		t.Fatal(errw.String())
	}
	args := []string{"m365", "mail", "send", "--to", "a@example.com", "--bcc", "x@example.com", "--bcc", "y@example.com", "--subject", "s", "--body", "b"}
	if code := Run(append(args, "--preview"), d); code != 0 || !strings.Contains(out.String(), "Bcc: x@example.com, y@example.com") {
		t.Fatal(code, out.String(), errw.String())
	}
	if len(mem.Sent) != 0 {
		t.Fatal("preview sent mail")
	}
	if code := Run(args, d); code != 0 {
		t.Fatal(errw.String())
	}
	sent, ok := mem.Sent[0].(mail.SendInput)
	if !ok || len(sent.BCC) != 2 || sent.BCC[1] != "y@example.com" {
		t.Fatalf("%+v", mem.Sent)
	}
}

func TestMCPExactRecipientsChecksBcc(t *testing.T) {
	p := MCPPolicy{ExactRecipients: true}
	bad := runIn{Namespace: "mail", Verb: "send", Flags: map[string]any{"to": "a@example.com", "bcc": "Some Name"}}
	if err := p.check(&bad); err == nil || domain.ClassOf(err) != domain.ClassUsage {
		t.Fatalf("name-based bcc passed: %v", err)
	}
	ok := runIn{Namespace: "mail", Verb: "send", Flags: map[string]any{"to": "a@example.com", "bcc": []string{"b@example.com"}}}
	if err := p.check(&ok); err != nil {
		t.Fatal(err)
	}
}

func TestMailReplyBccFlagReachesStore(t *testing.T) {
	d, out, errw := testDeps()
	mem := graph.Seed()
	d.Mail = graph.MailAPI{Memory: mem}
	d.Login = graph.FakeLoginAll(true, true, true, true)
	if code := Run([]string{"m365", "auth", "login"}, d); code != 0 {
		t.Fatal(errw.String())
	}
	args := []string{"m365", "mail", "reply", "msg-1", "--bcc", "x@example.com", "--body", "ok"}
	if code := Run(append(args, "--preview"), d); code != 0 || !strings.Contains(out.String(), "Bcc: x@example.com") {
		t.Fatal(code, out.String(), errw.String())
	}
	if code := Run(args, d); code != 0 {
		t.Fatal(errw.String())
	}
	sent, ok := mem.Sent[len(mem.Sent)-1].(mail.ReplyInput)
	if !ok || len(sent.BCC) != 1 {
		t.Fatalf("%+v", mem.Sent)
	}
}

func TestMailSendBccOnlyPreviewHasNoToLine(t *testing.T) {
	d, out, errw := testDeps()
	d.Mail = graph.MailAPI{Memory: graph.Seed()}
	d.Login = graph.FakeLoginAll(true, true, true, true)
	_ = Run([]string{"m365", "auth", "login"}, d)
	if code := Run([]string{"m365", "mail", "send", "--bcc", "x@example.com", "--subject", "s", "--body", "b", "--preview"}, d); code != 0 || strings.Contains(out.String(), "To:") {
		t.Fatal(code, out.String(), errw.String())
	}
}

func TestMCPExactRecipientsChecksReplyBcc(t *testing.T) {
	p := MCPPolicy{ExactRecipients: true}
	bad := runIn{Namespace: "mail", Verb: "reply", Flags: map[string]any{"bcc": "Some Name"}}
	if err := p.check(&bad); err == nil {
		t.Fatal("name-based reply bcc passed")
	}
	ok := runIn{Namespace: "mail", Verb: "reply", Flags: map[string]any{"bcc": "b@example.com"}}
	if err := p.check(&ok); err != nil {
		t.Fatal(err)
	}
}
