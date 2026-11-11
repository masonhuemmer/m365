package graph

import (
	"context"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/masonhuemmer/m365/internal/domain"
)

const fakeJoinURL = "https://teams.example.test/l/meetup-join/abc"

func wantMeetingsErr(t *testing.T, err error, class, msg string) {
	t.Helper()
	de, ok := err.(*domain.Error)
	if !ok || de.Class != class || de.Message != msg {
		t.Fatalf("err %#v, want %s %q", err, class, msg)
	}
	if strings.Contains(de.Message, "synthetic-token") || strings.Contains(de.Hint, "synthetic-token") {
		t.Fatal("token in error")
	}
}

func TestHTTPMeetingsEventMeeting(t *testing.T) {
	f := &meetingsFake{events: map[string]fakeResp{
		"evt-online":  onlineEvent(),
		"evt-offline": {body: []byte(`{"isOnlineMeeting":false,"start":{"dateTime":"2030-01-15T15:00:00.0000000"},"end":{"dateTime":"2030-01-15T15:30:00.0000000"}}`)},
	}}
	c := startMeetingsFake(t, f)
	occ, err := c.EventMeeting(context.Background(), "evt-online")
	if err != nil {
		t.Fatal(err)
	}
	if occ.JoinURL != fakeJoinURL || !occ.Start.Equal(time.Date(2030, 1, 15, 15, 0, 0, 0, time.UTC)) ||
		!occ.End.Equal(time.Date(2030, 1, 15, 15, 30, 0, 0, time.UTC)) || occ.Start.Location() != time.UTC {
		t.Fatalf("occurrence %#v", occ)
	}
	req := f.requests()[0]
	q, _ := url.ParseQuery(req.query)
	if req.path != "/me/events/evt-online" || q.Get("$select") != "start,end,isOnlineMeeting,onlineMeeting" || req.prefer != `outlook.timezone="UTC"` {
		t.Fatalf("request %#v", req)
	}
	_, err = c.EventMeeting(context.Background(), "evt-offline")
	wantMeetingsErr(t, err, domain.ClassNotFound, "event is not an online meeting")
	_, err = c.EventMeeting(context.Background(), "evt-missing")
	wantMeetingsErr(t, err, domain.ClassNotFound, "not found")
}

func TestHTTPMeetingsFindMeetingEscapesTheJoinURL(t *testing.T) {
	quoted := "https://teams.example.test/l/meetup-join/a'b"
	f := &meetingsFake{meetings: map[string]string{quoted: "mtg-1"}}
	c := startMeetingsFake(t, f)
	id, err := c.FindMeeting(context.Background(), quoted)
	if err != nil || id != "mtg-1" {
		t.Fatalf("id %q err %v", id, err)
	}
	q, _ := url.ParseQuery(f.requests()[0].query)
	if got := q.Get("$filter"); got != "JoinWebUrl eq 'https://teams.example.test/l/meetup-join/a''b'" {
		t.Fatalf("filter %q", got)
	}
	_, err = c.FindMeeting(context.Background(), "https://teams.example.test/other")
	wantMeetingsErr(t, err, domain.ClassNotFound, "no meeting found for that join URL")
}

func TestHTTPMeetingsTranscriptsFollowsPages(t *testing.T) {
	f := &meetingsFake{pages: [][]map[string]string{
		{transcriptJSON("t1", "2030-01-15T14:00:00Z")},
		{transcriptJSON("t2", "2030-01-15T15:05:00Z")},
		{transcriptJSON("t3", "2030-01-15T16:30:00Z")},
	}}
	c := startMeetingsFake(t, f)
	got, err := c.Transcripts(context.Background(), "mtg-1")
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.Transcript{
		{ID: "t1", Created: "2030-01-15T14:00:00Z", Ended: "2030-01-15T15:29:00Z"},
		{ID: "t2", Created: "2030-01-15T15:05:00Z", Ended: "2030-01-15T15:29:00Z"},
		{ID: "t3", Created: "2030-01-15T16:30:00Z", Ended: "2030-01-15T15:29:00Z"},
	}
	if !reflect.DeepEqual(got, want) || len(f.requests()) != 3 {
		t.Fatalf("got %#v after %d requests", got, len(f.requests()))
	}
	if f.requests()[0].path != "/me/onlineMeetings/mtg-1/transcripts" {
		t.Fatalf("path %q", f.requests()[0].path)
	}
}

func TestHTTPMeetingsTranscriptsPageCap(t *testing.T) {
	f := &meetingsFake{pages: emptyPages(11, transcriptJSON("t1", "2030-01-15T15:05:00Z"))}
	c := startMeetingsFake(t, f)
	_, err := c.Transcripts(context.Background(), "mtg-1")
	wantMeetingsErr(t, err, domain.ClassService, "more than 10 pages of transcripts")
	if n := len(f.requests()); n != 10 {
		t.Fatalf("%d list requests, want 10", n)
	}
}

func TestHTTPMeetingsTranscriptContent(t *testing.T) {
	vtt, err := os.ReadFile("../../domain/testdata/meeting.vtt")
	if err != nil {
		t.Fatal(err)
	}
	f := &meetingsFake{content: map[string]fakeResp{"t1": {body: vtt}}}
	c := startMeetingsFake(t, f)
	got, err := c.TranscriptContent(context.Background(), "mtg-1", "t1")
	if err != nil || string(got) != string(vtt) {
		t.Fatalf("content %q err %v", got, err)
	}
	req := f.requests()[0]
	q, _ := url.ParseQuery(req.query)
	if req.path != "/me/onlineMeetings/mtg-1/transcripts/t1/content" || q.Get("$format") != "text/vtt" {
		t.Fatalf("request %#v", req)
	}
}

func TestHTTPMeetingsContentCap(t *testing.T) {
	for _, size := range []int{10 << 20, 10<<20 + 2} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			body := []byte("WEBVTT\n" + strings.Repeat("x", size-7))
			c := startMeetingsFake(t, &meetingsFake{content: map[string]fakeResp{"t1": {body: body}}})
			got, err := c.TranscriptContent(context.Background(), "mtg-1", "t1")
			if size == 10<<20 {
				if err != nil || len(got) != size {
					t.Fatalf("len %d err %v", len(got), err)
				}
				return
			}
			wantMeetingsErr(t, err, domain.ClassService, "transcript is larger than the 10 MiB limit")
		})
	}
}
