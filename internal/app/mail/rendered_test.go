package mail

import (
	"context"
	"errors"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
	"github.com/masonhuemmer/m365/internal/domain/msgbody"
)

// replyRec records the reply input; the shared stub only counts replies.
type replyRec struct {
	*stub
	lastReply ReplyInput
}

func (r *replyRec) Reply(_ context.Context, in ReplyInput) (string, error) {
	r.sent++
	r.lastReply = in
	return "r-1", nil
}

func wantDryRendered(t *testing.T, out any, raw string, want msgbody.Rendered) {
	t.Helper()
	m := out.(map[string]any)
	if got := m["rendered"]; got != want {
		t.Fatalf("rendered %#v want %#v", got, want)
	}
	fp, ok := m["format_problems"].(msgbody.Problems)
	if !ok || len(fp) != 0 {
		t.Fatalf("format_problems %#v", m["format_problems"])
	}
	for _, k := range []string{"dry_run", "to", "subject", "attachments"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing existing key %q in %v", k, m)
		}
	}
	if m["body"] != raw {
		t.Fatalf("body %v want raw %q", m["body"], raw)
	}
}

func wantUsage(t *testing.T, err error, msg string) {
	t.Helper()
	var de *domain.Error
	if !errors.As(err, &de) || de.Class != domain.ClassUsage || de.Message != msg {
		t.Fatalf("got %v want usage %q", err, msg)
	}
}

func TestSendRendersBeforeDryRun(t *testing.T) {
	want := msgbody.Rendered{ContentType: "html", Content: "<p>b</p>"}
	st := &stub{}
	in := SendInput{To: []string{"user@example.com"}, Subject: "t", Body: "b"}
	dry := in
	dry.DryRun = true
	out, err := Send(context.Background(), st, sessMail(), dry)
	if err != nil {
		t.Fatal(err)
	}
	wantDryRendered(t, out, "b", want)
	if _, err := Send(context.Background(), st, sessMail(), in); err != nil {
		t.Fatal(err)
	}
	if st.last.Rendered != want {
		t.Fatalf("store got %#v want %#v", st.last.Rendered, want)
	}
}

func TestReplyRendersBeforeDryRun(t *testing.T) {
	want := msgbody.Rendered{ContentType: "html", Content: "<p>b</p>"}
	st := &replyRec{stub: &stub{}}
	out, err := Reply(context.Background(), st, sessMail(), ReplyInput{ID: "msg-1", Body: "b", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	wantDryRendered(t, out, "b", want)
	if _, err := Reply(context.Background(), st, sessMail(), ReplyInput{ID: "msg-1", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if st.lastReply.Rendered != want {
		t.Fatalf("store got %#v want %#v", st.lastReply.Rendered, want)
	}
}

func TestWhitespaceBodyIsMissing(t *testing.T) {
	st := &replyRec{stub: &stub{}}
	_, err := Send(context.Background(), st, sessMail(), SendInput{To: []string{"user@example.com"}, Subject: "t", Body: " \n\t "})
	wantUsage(t, err, "a recipient (--to or --bcc), subject, and body are required")
	_, err = Reply(context.Background(), st, sessMail(), ReplyInput{ID: "msg-1", Body: "   ", DryRun: true})
	wantUsage(t, err, "message id and body are required")
	if st.sent != 0 {
		t.Fatalf("sent %d", st.sent)
	}
}
