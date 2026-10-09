package graph

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/masonhuemmer/m365/internal/app/mail"
	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

func (c *HTTPClient) doJSON(ctx context.Context, method, path string, payload any) error {
	var body []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return domain.Usage(err.Error())
		}
		body = b
	}
	res, err := c.request(ctx, method, strings.TrimRight(c.Base, "/")+path, body, "application/json", "")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return MapStatus(res.StatusCode)
	}
	return nil
}

func (c *HTTPClient) Send(ctx context.Context, in mail.SendInput) (string, error) {
	to := make([]map[string]any, 0, len(in.To))
	for _, a := range in.To {
		to = append(to, map[string]any{"emailAddress": map[string]string{"address": a}})
	}
	cc := make([]map[string]any, 0, len(in.CC))
	for _, a := range in.CC {
		cc = append(cc, map[string]any{"emailAddress": map[string]string{"address": a}})
	}
	atts, err := fileAttachments(in.Files)
	if err != nil {
		return "", err
	}
	msg := map[string]any{
		"subject":      in.Subject,
		"body":         map[string]string{"contentType": "HTML", "content": in.Rendered.Content},
		"toRecipients": to,
	}
	if len(cc) > 0 {
		msg["ccRecipients"] = cc
	}
	if len(atts) > 0 {
		msg["attachments"] = atts
	}
	if err := c.doJSON(ctx, http.MethodPost, "/me/sendMail", map[string]any{"message": msg}); err != nil {
		return "", err
	}
	return "sent", nil
}

func (c *HTTPClient) Reply(ctx context.Context, in mail.ReplyInput) (string, error) {
	atts, err := fileAttachments(in.Files)
	if err != nil {
		return "", err
	}
	path := "/me/messages/" + url.PathEscape(in.ID) + "/reply"
	if in.All {
		path = "/me/messages/" + url.PathEscape(in.ID) + "/replyAll"
	}
	// Graph renders comment as HTML above the quoted thread. message.body
	// would replace the whole reply and drop the thread.
	payload := map[string]any{"comment": in.Rendered.Content}
	if len(atts) > 0 {
		payload["message"] = map[string]any{"attachments": atts}
	}
	if err := c.doJSON(ctx, http.MethodPost, path, payload); err != nil {
		return "", err
	}
	return "sent", nil
}

func (c *HTTPClient) Download(ctx context.Context, messageID, attach string) ([]byte, domain.Attachment, error) {
	res, err := c.do(ctx, http.MethodGet, "/me/messages/"+url.PathEscape(messageID)+"/attachments/"+url.PathEscape(attach)+"/$value")
	if err != nil {
		return nil, domain.Attachment{}, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, domain.Attachment{}, domain.Service(err.Error())
	}
	return b, domain.Attachment{ID: attach}, nil
}

func (c *HTTPTeams) Send(ctx context.Context, in teams.SendInput) (string, error) {
	// The app layer renders the body; send it exactly as dry-run showed it.
	content := in.Rendered.Content
	body := map[string]any{}
	var shared []sharedFile
	if len(in.Files) > 0 {
		var err error
		if shared, err = c.shareFiles(ctx, in); err != nil {
			return "", err
		}
		var atts []map[string]any
		content, atts = attachRefs(content, shared)
		body["attachments"] = atts
	}
	if len(in.Mentions) > 0 {
		body["mentions"] = mentionPayload(in.Mentions)
	}
	body["body"] = map[string]string{"contentType": "html", "content": content}
	if err := c.doJSON(ctx, http.MethodPost, "/me/chats/"+url.PathEscape(in.ChatID)+"/messages", body); err != nil {
		return "", partialUpload(err, shared)
	}
	return "sent", nil
}

func fileAttachments(files []domain.OutboundFile) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(files))
	for _, f := range files {
		b, err := os.ReadFile(f.Path)
		if err != nil {
			return nil, domain.Usagef("unreadable file: %s", f.Path)
		}
		out = append(out, map[string]any{
			"@odata.type":  "#microsoft.graph.fileAttachment",
			"name":         f.Name,
			"contentBytes": base64.StdEncoding.EncodeToString(b),
		})
	}
	return out, nil
}
