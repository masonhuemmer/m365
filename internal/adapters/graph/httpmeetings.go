package graph

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/masonhuemmer/m365/internal/app/meetings"
	"github.com/masonhuemmer/m365/internal/domain"
)

const (
	maxTranscriptPages = 10
	maxTranscriptBytes = 10 << 20
	// eventTimeLayout reads Graph's zone-less dateTime; a fraction is accepted when parsing.
	eventTimeLayout = "2006-01-02T15:04:05"
	utcPrefer       = `outlook.timezone="UTC"`
)

// HTTPMeetings reads online meetings and their transcripts from Microsoft Graph.
type HTTPMeetings struct{ *HTTPClient }

type graphTranscript struct {
	ID      string `json:"id"`
	Created string `json:"createdDateTime"`
	Ended   string `json:"endDateTime"`
}

func (c *HTTPMeetings) EventMeeting(ctx context.Context, eventID string) (meetings.Occurrence, error) {
	b, err := c.get(ctx, "/me/events/"+url.PathEscape(eventID)+"?$select=start,end,isOnlineMeeting,onlineMeeting", utcPrefer)
	if err != nil {
		return meetings.Occurrence{}, err
	}
	var ev struct {
		IsOnline bool `json:"isOnlineMeeting"`
		Online   struct {
			JoinURL string `json:"joinUrl"`
		} `json:"onlineMeeting"`
		Start struct{ DateTime string } `json:"start"`
		End   struct{ DateTime string } `json:"end"`
	}
	if err := json.Unmarshal(b, &ev); err != nil {
		return meetings.Occurrence{}, domain.Service("invalid event")
	}
	if !ev.IsOnline || ev.Online.JoinURL == "" {
		return meetings.Occurrence{}, domain.NotFound("event is not an online meeting")
	}
	start, serr := time.ParseInLocation(eventTimeLayout, ev.Start.DateTime, time.UTC)
	end, eerr := time.ParseInLocation(eventTimeLayout, ev.End.DateTime, time.UTC)
	if serr != nil || eerr != nil {
		return meetings.Occurrence{}, domain.Service("invalid event")
	}
	return meetings.Occurrence{JoinURL: ev.Online.JoinURL, Start: start, End: end}, nil
}

func (c *HTTPMeetings) FindMeeting(ctx context.Context, joinURL string) (string, error) {
	filter := "JoinWebUrl eq '" + strings.ReplaceAll(joinURL, "'", "''") + "'"
	b, err := c.get(ctx, "/me/onlineMeetings?$filter="+url.QueryEscape(filter), "")
	if err != nil {
		return "", err
	}
	var list struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(b, &list); err != nil {
		return "", domain.Service("invalid meeting list")
	}
	if len(list.Value) == 0 || list.Value[0].ID == "" {
		return "", domain.NotFound("no meeting found for that join URL")
	}
	return list.Value[0].ID, nil
}

// Transcripts reads every page of the meeting's transcript list, up to
// maxTranscriptPages, following only https links on the Graph host.
func (c *HTTPMeetings) Transcripts(ctx context.Context, meetingID string) ([]domain.Transcript, error) {
	next := "/me/onlineMeetings/" + url.PathEscape(meetingID) + "/transcripts"
	items := []domain.Transcript{}
	for range maxTranscriptPages {
		page, link, err := c.transcriptPage(ctx, next)
		if err != nil {
			return nil, err
		}
		items = append(items, page...)
		if link == "" {
			return items, nil
		}
		if !c.graphLink(link) {
			return nil, domain.Service("refused a transcript list link outside Microsoft Graph")
		}
		next = link
	}
	return nil, domain.Service("more than 10 pages of transcripts")
}

func (c *HTTPMeetings) transcriptPage(ctx context.Context, next string) ([]domain.Transcript, string, error) {
	b, err := c.get(ctx, next, "")
	if err != nil {
		return nil, "", err
	}
	var page struct {
		Value []graphTranscript `json:"value"`
		Next  string            `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(b, &page); err != nil {
		return nil, "", domain.Service("invalid transcript")
	}
	items := make([]domain.Transcript, 0, len(page.Value))
	for _, g := range page.Value {
		items = append(items, domain.Transcript(g))
	}
	return items, page.Next, nil
}

// graphLink accepts a continuation link only over https to the Graph host.
func (c *HTTPMeetings) graphLink(link string) bool {
	u, err := url.Parse(link)
	base, berr := url.Parse(c.Base)
	return err == nil && berr == nil && u.Scheme == "https" && strings.EqualFold(u.Host, base.Host)
}

func (c *HTTPMeetings) TranscriptContent(ctx context.Context, meetingID, transcriptID string) ([]byte, error) {
	b, err := c.get(ctx, "/me/onlineMeetings/"+url.PathEscape(meetingID)+"/transcripts/"+url.PathEscape(transcriptID)+"/content?$format=text/vtt", "")
	if err != nil {
		return nil, err
	}
	if len(b) > maxTranscriptBytes {
		return nil, domain.Service("transcript is larger than the 10 MiB limit")
	}
	return b, nil
}

// get sends one GET (path relative to Base, or an already checked absolute
// link) and reads at most maxTranscriptBytes+1 bytes of the answer.
func (c *HTTPMeetings) get(ctx context.Context, path, prefer string) ([]byte, error) {
	raw := path
	if strings.HasPrefix(path, "/") {
		raw = strings.TrimRight(c.Base, "/") + path
	}
	res, err := c.request(ctx, http.MethodGet, raw, nil, "", prefer)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, maxTranscriptBytes+1))
	if err != nil {
		return nil, domain.Service("microsoft 365 service error")
	}
	if res.StatusCode >= 400 {
		return nil, MapGraphError(res.StatusCode, b)
	}
	return b, nil
}
