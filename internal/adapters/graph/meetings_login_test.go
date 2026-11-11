package graph

import (
	"context"
	"reflect"
	"testing"

	"golang.org/x/oauth2"
)

var plainScopes = []string{"openid", "offline_access", "profile", "Mail.Read", "Mail.Send", "Chat.Read", "Chat.ReadWrite", "ChatMessage.Send", "Calendars.ReadWrite", "Files.ReadWrite"}

func TestPlainLoginScopesAreUnchanged(t *testing.T) {
	if got := PKCEConfig("c", "t", "").Scopes; !reflect.DeepEqual(got, plainScopes) {
		t.Fatalf("scopes %q", got)
	}
}

func TestMeetingsLoginAddsTwoScopes(t *testing.T) {
	want := []string{"OnlineMeetings.Read", "OnlineMeetingTranscript.Read.All"}
	if !reflect.DeepEqual(MeetingsScopes, want) {
		t.Fatalf("MeetingsScopes %q", MeetingsScopes)
	}
	got := PKCEConfig("c", "t", "", MeetingsScopes...).Scopes
	if !reflect.DeepEqual(got, append(append([]string{}, plainScopes...), want...)) {
		t.Fatalf("scopes %q", got)
	}
	if again := PKCEConfig("c", "t", "").Scopes; !reflect.DeepEqual(again, plainScopes) {
		t.Fatalf("a meetings config changed the plain list: %q", again)
	}
}

func TestBlobMeetingsFlag(t *testing.T) {
	meetingsRequest := append(append([]string{}, plainScopes...), "OnlineMeetings.Read", "OnlineMeetingTranscript.Read.All")
	cases := []struct {
		name, scope string
		requested   []string
		want        bool
	}{
		{"granted, meetings request", "Mail.Read OnlineMeetings.Read OnlineMeetingTranscript.Read.All", meetingsRequest, true},
		{"granted, plain request", "Mail.Read OnlineMeetings.Read OnlineMeetingTranscript.Read.All", plainScopes, true},
		{"one scope only", "OnlineMeetings.Read", meetingsRequest, false},
		{"omitted, meetings request", "", meetingsRequest, true},
		{"omitted, plain request", "", plainScopes, false},
		{"resource URIs", "https://graph.microsoft.com/OnlineMeetings.Read https://graph.microsoft.com/OnlineMeetingTranscript.Read.All", plainScopes, true},
	}
	for _, c := range cases {
		// An empty access token keeps blobFromToken from calling Graph's /me.
		tok := (&oauth2.Token{}).WithExtra(map[string]any{"scope": c.scope})
		if got := blobFromToken(context.Background(), tok, c.requested).Meetings; got != c.want {
			t.Errorf("%s: Meetings = %v, want %v", c.name, got, c.want)
		}
	}
}
