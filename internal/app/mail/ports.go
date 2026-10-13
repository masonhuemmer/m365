package mail

import (
	"context"
	"strings"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
	"github.com/masonhuemmer/m365/internal/domain/msgbody"
)

type ListQuery struct {
	Folder    string
	Unread    bool
	Search    string
	Top       int
	PageToken string
}

type SendInput struct {
	To, CC, BCC []string
	Subject     string
	Body        string
	HTML        bool
	MD          bool
	DryRun      bool
	NoteToSelf  bool
	Files       []domain.OutboundFile
	// Rendered is the delivered body, set by Send. Stores send it as-is.
	Rendered msgbody.Rendered
}

type ReplyInput struct {
	ID     string
	BCC    []string
	Body   string
	All    bool
	HTML   bool
	MD     bool
	DryRun bool
	Files  []domain.OutboundFile
	// Rendered is the delivered comment, set by Reply. Stores send it as-is.
	Rendered msgbody.Rendered
}

type Store interface {
	List(ctx context.Context, q ListQuery) (domain.MailPage, error)
	Get(ctx context.Context, id string) (domain.MailMessage, error)
	Thread(ctx context.Context, id string, bodies bool) (domain.MailThread, error)
	Send(ctx context.Context, in SendInput) (string, error)
	Reply(ctx context.Context, in ReplyInput) (string, error)
	Attachments(ctx context.Context, messageID string) ([]domain.Attachment, error)
	Download(ctx context.Context, messageID, attach string) ([]byte, domain.Attachment, error)
}

func List(ctx context.Context, st Store, sess domain.Session, q ListQuery) (domain.MailPage, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return domain.MailPage{}, err
	}
	if q.Folder == "" {
		q.Folder = "inbox"
	}
	top, err := domain.NormalizeTop(q.Top, domain.DefaultMailTop)
	if err != nil {
		return domain.MailPage{}, err
	}
	q.Top = top
	return st.List(ctx, q)
}

func Get(ctx context.Context, st Store, sess domain.Session, id string) (domain.MailMessage, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return domain.MailMessage{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.MailMessage{}, domain.Usage("message id is required")
	}
	return st.Get(ctx, id)
}

func Thread(ctx context.Context, st Store, sess domain.Session, id string, bodies bool) (domain.MailThread, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return domain.MailThread{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.MailThread{}, domain.Usage("message id is required")
	}
	return st.Thread(ctx, id, bodies)
}

func Send(ctx context.Context, st Store, sess domain.Session, in SendInput) (any, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return nil, err
	}
	if in.NoteToSelf {
		if len(in.To) > 0 {
			return nil, domain.Usage("use --note-to-self or --to, not both")
		}
		if strings.TrimSpace(sess.Account) == "" {
			return nil, domain.Usage("signed-in account is required")
		}
		in.To = []string{sess.Account}
		if in.Subject == "" {
			in.Subject = "Note to self"
		}
	}
	if len(in.To)+len(in.BCC) == 0 || in.Subject == "" || in.Body == "" {
		return nil, domain.Usage("a recipient (--to or --bcc), subject, and body are required")
	}
	if err := domain.ValidateOutbound(in.Files); err != nil {
		return nil, err
	}
	in.Rendered = msgbody.Render(msgbody.ModeFor(in.HTML, in.MD), msgbody.Mail, in.Body)
	if strings.TrimSpace(in.Rendered.Content) == "" {
		return nil, domain.Usage("a recipient (--to or --bcc), subject, and body are required")
	}
	problems := append(msgbody.LintSubject(in.Subject), msgbody.Lint(in.Rendered.Content)...)
	if in.DryRun {
		return dryPayload(in.To, in.CC, in.BCC, in.Subject, in.Body, in.Files, in.Rendered, problems), nil
	}
	if err := problems.Err(); err != nil {
		return nil, err
	}
	id, err := st.Send(ctx, in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "sent": true}, nil
}

func Reply(ctx context.Context, st Store, sess domain.Session, in ReplyInput) (any, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return nil, err
	}
	if in.ID == "" || in.Body == "" {
		return nil, domain.Usage("message id and body are required")
	}
	if err := domain.ValidateOutbound(in.Files); err != nil {
		return nil, err
	}
	in.Rendered = msgbody.Render(msgbody.ModeFor(in.HTML, in.MD), msgbody.Mail, in.Body)
	if strings.TrimSpace(in.Rendered.Content) == "" {
		return nil, domain.Usage("message id and body are required")
	}
	problems := msgbody.Lint(in.Rendered.Content)
	if in.DryRun {
		return dryPayload(nil, nil, in.BCC, "", in.Body, in.Files, in.Rendered, problems), nil
	}
	if err := problems.Err(); err != nil {
		return nil, err
	}
	id, err := st.Reply(ctx, in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "sent": true}, nil
}

func ListAttachments(ctx context.Context, st Store, sess domain.Session, id string) ([]domain.Attachment, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, domain.Usage("message id is required")
	}
	return st.Attachments(ctx, id)
}

func Save(ctx context.Context, st Store, sess domain.Session, messageID, attach, dest string, overwrite bool, write func(string, []byte, bool) error) (string, error) {
	if err := auth.Require(sess, true, false); err != nil {
		return "", err
	}
	if messageID == "" || attach == "" {
		return "", domain.Usage("message id and attachment are required")
	}
	if dest == "" {
		return "", domain.Usage("destination path is required")
	}
	data, meta, err := st.Download(ctx, messageID, attach)
	if err != nil {
		return "", err
	}
	_ = meta
	if err := write(dest, data, overwrite); err != nil {
		return "", err
	}
	return dest, nil
}

func dryPayload(to, cc, bcc []string, subject, body string, files []domain.OutboundFile, r msgbody.Rendered, problems msgbody.Problems) map[string]any {
	atts := make([]map[string]any, 0, len(files))
	for _, f := range files {
		atts = append(atts, map[string]any{"name": f.Name, "size": f.Size})
	}
	return map[string]any{
		"dry_run":         true,
		"to":              orEmpty(to),
		"cc":              orEmpty(cc),
		"bcc":             orEmpty(bcc),
		"subject":         subject,
		"body":            body,
		"attachments":     atts,
		"rendered":        r,
		"format_problems": problems,
	}
}

// orEmpty keeps unset recipient lists as [] in dry-run JSON, never null.
func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
