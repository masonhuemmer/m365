package graph

import (
	"context"
	"testing"
)

func TestListVisibleChatsFiltersHiddenOnTheServer(t *testing.T) {
	mem := Seed()
	srv := NewFakeServer(mem)
	defer srv.Close()
	c := &HTTPTeams{HTTPClient: &HTTPClient{Base: srv.URL, Token: "fake-both", Client: srv.Client()}}
	if err := mem.HideChat(context.Background(), "chat-2", true); err != nil {
		t.Fatal(err)
	}
	all, err := c.ListChats(context.Background(), 50, "")
	if err != nil {
		t.Fatal(err)
	}
	visible, err := c.ListVisibleChats(context.Background(), 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if visible.Count != all.Count-1 {
		t.Fatalf("all %d visible %d", all.Count, visible.Count)
	}
	for _, ch := range visible.Items {
		if ch.ID == "chat-2" || ch.Hidden {
			t.Fatalf("hidden chat listed: %+v", ch)
		}
	}
	found := false
	for _, ch := range all.Items {
		found = found || (ch.ID == "chat-2" && ch.Hidden)
	}
	if !found {
		t.Fatal("hidden chat not marked hidden in the full list")
	}
}

func TestListVisibleChatsKeepsFilterOnLaterPages(t *testing.T) {
	mem := Seed()
	srv := NewFakeServer(mem)
	defer srv.Close()
	c := &HTTPTeams{HTTPClient: &HTTPClient{Base: srv.URL, Token: "fake-both", Client: srv.Client()}}
	if err := mem.HideChat(context.Background(), "chat-ajay", true); err != nil {
		t.Fatal(err)
	}
	p, err := c.ListVisibleChats(context.Background(), 20, "")
	if err != nil || p.NextPage == nil {
		t.Fatal(err, p)
	}
	p2, err := c.ListVisibleChats(context.Background(), 20, *p.NextPage)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range p2.Items {
		if ch.ID == "chat-ajay" {
			t.Fatal("hidden chat on page 2")
		}
	}
	if p2.Count == 0 {
		t.Fatal("page 2 empty")
	}
}
