package graph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

func (c *HTTPTeams) CreateChat(ctx context.Context, topic string, members []string) (domain.Chat, error) {
	self, err := c.selfID(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	ids := append([]string{self}, members...)
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
	res, err := c.doBody(ctx, http.MethodPost, "/chats", body)
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

func (c *HTTPTeams) selfID(ctx context.Context) (string, error) {
	res, err := c.do(ctx, http.MethodGet, "/me?$select=id")
	if err != nil {
		return "", err
	}
	var me struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(res.Body).Decode(&me)
	_ = res.Body.Close()
	if err != nil || me.ID == "" {
		return "", domain.Service("could not read the signed-in user id")
	}
	return me.ID, nil
}

// tenantID reads the tid claim of the access token, so no extra permission is needed.
func (c *HTTPTeams) tenantID() (string, error) {
	parts := strings.Split(c.bearer(false), ".")
	if len(parts) == 3 {
		if raw, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				TID string `json:"tid"`
			}
			if json.Unmarshal(raw, &claims) == nil && claims.TID != "" {
				return claims.TID, nil
			}
		}
	}
	return "", domain.Service("could not read the tenant id from the access token")
}

// HideChat calls hideForUser or unhideForUser for the signed-in user.
func (c *HTTPTeams) HideChat(ctx context.Context, chatID string, hidden bool) error {
	self, err := c.selfID(ctx)
	if err != nil {
		return err
	}
	tenant, err := c.tenantID()
	if err != nil {
		return err
	}
	action := "unhideForUser"
	if hidden {
		action = "hideForUser"
	}
	body := map[string]any{"user": map[string]string{"id": self, "tenantId": tenant}}
	return c.doJSON(ctx, http.MethodPost, "/chats/"+url.PathEscape(chatID)+"/"+action, body)
}

func (c *HTTPTeams) RenameChat(ctx context.Context, chatID, topic string) error {
	return c.doJSON(ctx, http.MethodPatch, "/chats/"+url.PathEscape(chatID), map[string]string{"topic": topic})
}
