package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

func (c *HTTPTeams) CreateChat(ctx context.Context, topic string, members []string) (domain.Chat, error) {
	res, err := c.do(ctx, http.MethodGet, "/me?$select=id")
	if err != nil {
		return domain.Chat{}, err
	}
	var me struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(res.Body).Decode(&me)
	_ = res.Body.Close()
	if err != nil || me.ID == "" {
		return domain.Chat{}, domain.Service("could not read the signed-in user id")
	}
	ids := append([]string{me.ID}, members...)
	bound := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		bound = append(bound, map[string]any{
			"@odata.type":     "#microsoft.graph.aadUserConversationMember",
			"roles":           []string{"owner"},
			"user@odata.bind": "https://graph.microsoft.com/v1.0/users('" + strings.ReplaceAll(id, "'", "''") + "')",
		})
	}
	body, err := json.Marshal(map[string]any{"chatType": "group", "topic": topic, "members": bound})
	if err != nil {
		return domain.Chat{}, domain.Usage(err.Error())
	}
	res, err = c.doBody(ctx, http.MethodPost, "/chats", body)
	if err != nil {
		return domain.Chat{}, err
	}
	defer res.Body.Close()
	var g graphChat
	if err := json.NewDecoder(res.Body).Decode(&g); err != nil || g.ID == "" {
		return domain.Chat{}, domain.Service("invalid create chat response")
	}
	return mapGraphChat(g), nil
}

func (c *HTTPTeams) RenameChat(ctx context.Context, chatID, topic string) error {
	return c.doJSON(ctx, http.MethodPatch, "/chats/"+url.PathEscape(chatID), map[string]string{"topic": topic})
}
