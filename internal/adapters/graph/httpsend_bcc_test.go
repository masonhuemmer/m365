package graph

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/app/mail"
	"github.com/masonhuemmer/m365/internal/domain/msgbody"
)

func sendMessage(t *testing.T, in mail.SendInput) map[string]any {
	t.Helper()
	srv, cap := captureJSONServer(t)
	c := &HTTPClient{Base: srv.URL, Token: "t"}
	in.Rendered = msgbody.Rendered{Content: "<p>b</p>"}
	if _, err := c.Send(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	msg, _ := cap.Body["message"].(map[string]any)
	return msg
}

func TestHTTPSendPostsBccRecipients(t *testing.T) {
	msg := sendMessage(t, mail.SendInput{To: []string{"a@example.com"}, BCC: []string{"hidden@example.com", "two@example.com"}, Subject: "s"})
	bcc, _ := msg["bccRecipients"].([]any)
	if len(bcc) != 2 {
		t.Fatalf("bccRecipients: %v", msg["bccRecipients"])
	}
	first := bcc[0].(map[string]any)["emailAddress"].(map[string]any)["address"]
	if first != "hidden@example.com" {
		t.Fatal(first)
	}
}

func TestHTTPSendWithoutBccOmitsTheField(t *testing.T) {
	msg := sendMessage(t, mail.SendInput{To: []string{"a@example.com"}, CC: []string{"c@example.com"}, Subject: "s"})
	if _, ok := msg["bccRecipients"]; ok {
		t.Fatal("bccRecipients sent without --bcc")
	}
	if _, ok := msg["ccRecipients"]; !ok {
		t.Fatal("ccRecipients lost")
	}
}
