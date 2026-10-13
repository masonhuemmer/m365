package mail

import (
	"context"
	"testing"

	"github.com/masonhuemmer/m365/internal/domain"
)

func TestDryRunListsAllRecipientsAndSendKeepsBcc(t *testing.T) {
	st := &stub{}
	in := SendInput{To: []string{"a@example.com"}, CC: []string{"c@example.com"}, BCC: []string{"hidden@example.com"}, Subject: "t", Body: "b", DryRun: true}
	out, err := Send(context.Background(), st, sessMail(), in)
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["cc"].([]string)[0] != "c@example.com" || m["bcc"].([]string)[0] != "hidden@example.com" || st.sent != 0 {
		t.Fatalf("dry run: %v sent=%d", out, st.sent)
	}
	in.DryRun = false
	if _, err := Send(context.Background(), st, sessMail(), in); err != nil || st.sent != 1 {
		t.Fatal(err, st.sent)
	}
}

func TestDryRunUnsetRecipientListsAreEmptyArrays(t *testing.T) {
	out, err := Send(context.Background(), &stub{}, sessMail(), SendInput{To: []string{"a@example.com"}, Subject: "t", Body: "b", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	for _, key := range []string{"cc", "bcc"} {
		if list, ok := m[key].([]string); !ok || list == nil || len(list) != 0 {
			t.Fatalf("%s: %#v", key, m[key])
		}
	}
}

func TestSendAllowsBccWithoutTo(t *testing.T) {
	st := &stub{}
	in := SendInput{BCC: []string{"hidden@example.com"}, Subject: "t", Body: "b"}
	if _, err := Send(context.Background(), st, sessMail(), in); err != nil || st.sent != 1 {
		t.Fatal(err, st.sent)
	}
	in.BCC = nil
	if _, err := Send(context.Background(), st, sessMail(), in); domain.ExitOf(err) != domain.ExitUsage {
		t.Fatalf("no recipients accepted: %v", err)
	}
}

func TestReplyDryRunShowsBcc(t *testing.T) {
	out, err := Reply(context.Background(), &stub{}, sessMail(), ReplyInput{ID: "m1", BCC: []string{"hidden@example.com"}, Body: "ok", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.(map[string]any)["bcc"].([]string)[0] != "hidden@example.com" {
		t.Fatal(out)
	}
}
