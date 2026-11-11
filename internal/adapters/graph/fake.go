package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

func NewFakeServer(mem *Memory) *httptest.Server {
	if mem == nil {
		mem = Seed()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		switch tok {
		case "fake-expired", "":
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		case "fake-mail":
			if strings.Contains(r.URL.Path, "chats") {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
		case "fake-teams":
			if strings.Contains(r.URL.Path, "/me/messages") {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
		}
		if denyWorkload(tok, r.URL.Path) {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		if mem.Throttle {
			http.Error(w, `{"error":"throttled"}`, http.StatusTooManyRequests)
			return
		}
		p := r.URL.Path
		if r.Method == http.MethodPost && (strings.HasSuffix(p, "/sendMail") || strings.Contains(p, "/reply") || strings.HasSuffix(p, "/messages")) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case p == "/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "user-1", "displayName": "Test User"})
		case p == "/me/messages":
			out := map[string]any{"value": graphMailValues(mem)}
			if r.URL.Query().Get("$filter") == "" {
				out["@odata.nextLink"] = "https://graph.microsoft.com/v1.0/me/messages?$top=10&$skip=10"
			}
			_ = json.NewEncoder(w).Encode(out)
		case strings.HasPrefix(p, "/me/messages/"):
			id := strings.TrimPrefix(p, "/me/messages/")
			if i := strings.IndexByte(id, '?'); i >= 0 {
				id = id[:i]
			}
			for _, m := range mem.Mails {
				if m.ID == id {
					_ = json.NewEncoder(w).Encode(graphMailOne(m))
					return
				}
			}
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "ErrorInvalidId"}})
		case p == "/me/chats":
			top := 50
			if t := r.URL.Query().Get("$top"); t != "" {
				if n, err := strconv.Atoi(t); err == nil && n > 0 {
					top = n
				}
			}
			skip := 0
			if s := r.URL.Query().Get("$skiptoken"); strings.HasPrefix(s, "s.") {
				skip, _ = strconv.Atoi(strings.TrimPrefix(s, "s."))
			}
			chats := mem.chatsWithHidden(strings.Contains(r.URL.Query().Get("$filter"), "viewpoint/isHidden eq false"))
			if skip > len(chats) {
				skip = len(chats)
			}
			end := skip + top
			if end > len(chats) {
				end = len(chats)
			}
			vals := []map[string]any{}
			for _, c := range chats[skip:end] {
				mems := []map[string]any{}
				for _, p := range c.Members {
					mems = append(mems, map[string]any{"displayName": p.Name, "email": p.Address})
				}
				vals = append(vals, map[string]any{
					"id": c.ID, "topic": c.Topic, "chatType": c.Type, "members": mems,
					"viewpoint": map[string]any{"isHidden": c.Hidden},
				})
			}
			out := map[string]any{"value": vals}
			if end < len(chats) {
				out["@odata.nextLink"] = "s." + strconv.Itoa(end)
			}
			_ = json.NewEncoder(w).Encode(out)
		case strings.HasSuffix(p, "/messages") && strings.Contains(p, "/me/chats/"):
			id := strings.TrimSuffix(strings.TrimPrefix(p, "/me/chats/"), "/messages")
			vals := []map[string]any{}
			for _, m := range mem.Msgs[id] {
				msg := map[string]any{
					"id": m.ID, "createdDateTime": m.Created,
					"body": map[string]string{"content": m.Text},
					"from": map[string]any{"user": map[string]string{"displayName": m.From}},
				}
				if m.ID == "cmsg-2" {
					msg["mentions"] = []map[string]any{{
						"mentioned": map[string]any{"user": map[string]string{"id": "user-1"}},
					}}
				}
				vals = append(vals, msg)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"value": vals})
		case strings.HasPrefix(p, "/me/chats/"):
			id := strings.TrimPrefix(p, "/me/chats/")
			for _, c := range mem.Chats {
				if c.ID == id {
					_ = json.NewEncoder(w).Encode(map[string]any{"id": c.ID, "topic": c.Topic, "chatType": c.Type})
					return
				}
			}
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		default:
			if handleCalendarFiles(w, r, mem) {
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
		}
	})
	return httptest.NewServer(mux)
}

func graphMailValues(mem *Memory) []map[string]any {
	out := make([]map[string]any, 0, len(mem.Mails))
	for _, m := range mem.Mails {
		out = append(out, graphMailOne(m))
	}
	return out
}

func graphMailOne(m domain.MailMessage) map[string]any {
	atts := []map[string]any{}
	for _, a := range m.Attachments {
		atts = append(atts, map[string]any{"id": a.ID, "name": a.Name, "size": a.Size, "contentType": a.ContentType})
	}
	return map[string]any{
		"id": m.ID, "subject": m.Subject, "conversationId": m.Conversation,
		"receivedDateTime": m.Received,
		"isRead":           m.IsRead,
		"hasAttachments":   m.HasAttachments,
		"body":             map[string]string{"content": m.Body},
		"from":             map[string]any{"emailAddress": map[string]string{"address": m.From.Address, "name": m.From.Name}},
		"toRecipients":     graphMailRecipients(m.To),
		"ccRecipients":     graphMailRecipients(m.CC),
		"attachments":      atts,
	}
}

func graphMailRecipients(people []domain.Person) []map[string]any {
	recipients := make([]map[string]any, 0, len(people))
	for _, person := range people {
		recipients = append(recipients, map[string]any{
			"emailAddress": map[string]string{"address": person.Address, "name": person.Name},
		})
	}
	return recipients
}
