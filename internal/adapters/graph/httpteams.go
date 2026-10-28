package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

type HTTPTeams struct{ *HTTPClient }

func (c *HTTPTeams) ListChats(ctx context.Context, top int, page string) (domain.ChatPage, error) {
	path := "/me/chats?$expand=members&$top=" + strconv.Itoa(top)
	if page != "" {
		if next, ok := decodeNext(page); ok {
			if strings.HasPrefix(next, "http") {
				path = next
			} else {
				path += "&$skiptoken=" + url.QueryEscape(next)
			}
		} else if strings.HasPrefix(page, "http") {
			path = page
		} else {
			path += "&$skiptoken=" + url.QueryEscape(page)
		}
	}
	res, err := c.do(ctx, http.MethodGet, path)
	if err != nil {
		return domain.ChatPage{}, err
	}
	defer res.Body.Close()
	var raw graphChatList
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return domain.ChatPage{}, domain.Service("invalid chat list")
	}
	items := make([]domain.Chat, 0, len(raw.Value))
	for _, g := range raw.Value {
		items = append(items, mapGraphChat(g))
	}
	p := domain.ChatPage{Limit: top, Count: len(items), Items: items}
	if tok := encodeNext(raw.Next); tok != "" {
		p.NextPage = &tok
	}
	return p, nil
}

func (c *HTTPTeams) GetChat(ctx context.Context, id string) (domain.Chat, error) {
	res, err := c.do(ctx, http.MethodGet, "/me/chats/"+url.PathEscape(id)+"?$expand=members")
	if err != nil {
		return domain.Chat{}, err
	}
	defer res.Body.Close()
	var g graphChat
	if err := json.NewDecoder(res.Body).Decode(&g); err != nil {
		return domain.Chat{}, domain.Service("invalid chat")
	}
	return mapGraphChat(g), nil
}

func mapGraphChat(g graphChat) domain.Chat {
	c := domain.Chat{ID: g.ID, Topic: g.Topic, Type: g.ChatType}
	for _, m := range g.Members {
		c.Members = append(c.Members, domain.Person{ID: m.UserID, Name: m.DisplayName, Address: m.Email})
	}
	return c
}

func (c *HTTPTeams) Messages(ctx context.Context, q teams.MessageQuery) (domain.ChatMessagePage, error) {
	u := "/me/chats/" + url.PathEscape(q.ChatID) + "/messages?$top=" + strconv.Itoa(q.Top)
	if q.PageToken != "" {
		if next, ok := decodeNext(q.PageToken); ok {
			u = next
		} else {
			u += "&$skiptoken=" + url.QueryEscape(q.PageToken)
		}
	}
	res, err := c.do(ctx, http.MethodGet, u)
	if err != nil {
		return domain.ChatMessagePage{}, err
	}
	defer res.Body.Close()
	var raw graphMsgList
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return domain.ChatMessagePage{}, domain.Service("invalid messages")
	}
	items := make([]domain.ChatMessage, 0, len(raw.Value))
	for _, g := range raw.Value {
		m := g.toChatMessage(q.ChatID)
		if m.System && !q.IncludeSystem {
			continue
		}
		items = append(items, m)
	}
	p := domain.ChatMessagePage{Limit: q.Top, Count: len(items), Items: items}
	if tok := encodeNext(raw.Next); tok != "" {
		p.NextPage = &tok
	}
	return p, nil
}

func (c *HTTPTeams) Attachments(context.Context, string, string) ([]domain.Attachment, error) {
	return nil, domain.Service("not wired")
}
func (c *HTTPTeams) Download(context.Context, string, string, string) ([]byte, domain.Attachment, error) {
	return nil, domain.Attachment{}, domain.Service("not wired")
}

type graphChatList struct {
	Value []graphChat `json:"value"`
	Next  string      `json:"@odata.nextLink"`
}
type graphChat struct {
	ID       string        `json:"id"`
	Topic    string        `json:"topic"`
	ChatType string        `json:"chatType"`
	Members  []graphMember `json:"members"`
}
type graphMember struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}
type graphMsgList struct {
	Value []graphChatMsg `json:"value"`
	Next  string         `json:"@odata.nextLink"`
}
type graphChatMsg struct {
	ID      string `json:"id"`
	Created string `json:"createdDateTime"`
	Type    string `json:"messageType"`
	Body    struct {
		Content string `json:"content"`
	} `json:"body"`
	From *struct {
		User        *graphIdentity `json:"user"`
		Application *graphIdentity `json:"application"`
	} `json:"from"`
}
type graphIdentity struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// toChatMessage names the sender as the user, else the app (bots, webhooks).
// Graph returns from=null for system events, which report as messageType
// systemEventMessage or unknownFutureValue.
func (g graphChatMsg) toChatMessage(chatID string) domain.ChatMessage {
	m := domain.ChatMessage{ID: g.ID, ChatID: chatID, Text: g.Body.Content, Created: g.Created}
	m.System = g.Type != "" && g.Type != "message"
	if g.From != nil {
		if g.From.User != nil {
			m.From = g.From.User.DisplayName
			m.FromUserID = g.From.User.ID
			m.SenderType = "user"
		} else if g.From.Application != nil {
			m.From = g.From.Application.DisplayName
			m.FromApplicationID = g.From.Application.ID
			m.SenderType = "application"
		}
	}
	return m
}
