package mail

import (
	"context"
	"testing"
)

func TestDryRunListsBccAndSendKeepsIt(t *testing.T) {
	st := &stub{}
	in := SendInput{To: []string{"a@example.com"}, BCC: []string{"hidden@example.com"}, Subject: "t", Body: "b", DryRun: true}
	out, err := Send(context.Background(), st, sessMail(), in)
	if err != nil {
		t.Fatal(err)
	}
	bcc, _ := out.(map[string]any)["bcc"].([]string)
	if len(bcc) != 1 || bcc[0] != "hidden@example.com" || st.sent != 0 {
		t.Fatalf("dry run: %v sent=%d", out, st.sent)
	}
	in.DryRun = false
	if _, err := Send(context.Background(), st, sessMail(), in); err != nil || st.sent != 1 {
		t.Fatal(err, st.sent)
	}
	out, _ = Send(context.Background(), st, sessMail(), SendInput{To: []string{"a@example.com"}, Subject: "t", Body: "b", DryRun: true})
	if _, ok := out.(map[string]any)["bcc"]; ok {
		t.Fatal("bcc key present without recipients")
	}
}
