package teams

import (
	"context"
	"testing"
)

func TestListMappedSkipsHiddenUnlessAsked(t *testing.T) {
	st := &chatMem{chats: sampleChats()}
	if _, err := List(context.Background(), st, sess(), 50, ""); err != nil {
		t.Fatal(err)
	}
	if st.visibleCalls != 1 {
		t.Fatalf("visible %d", st.visibleCalls)
	}
	before := st.listCalls
	if _, err := ListMapped(context.Background(), st, nil, sess(), 50, "", true); err != nil {
		t.Fatal(err)
	}
	if st.visibleCalls != 1 || st.listCalls != before+1 {
		t.Fatalf("visible %d all %d (was %d)", st.visibleCalls, st.listCalls, before)
	}
}
