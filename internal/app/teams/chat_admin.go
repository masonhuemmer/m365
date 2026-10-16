package teams

import (
	"context"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

// MaxTopicLength is Teams' limit for a group chat name.
const MaxTopicLength = 250

// MinGroupMembers is how many people besides the signed-in user a group chat needs.
const MinGroupMembers = 2

var memberID = regexp.MustCompile(`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`)

type CreateInput struct {
	Name    string
	Members []string
	DryRun  bool
}

func CreateGroup(ctx context.Context, st Store, sess domain.Session, in CreateInput) (any, error) {
	if err := auth.Require(sess, false, true); err != nil {
		return nil, err
	}
	name, err := validTopic(in.Name)
	if err != nil {
		return nil, err
	}
	members, err := validMembers(in.Members, sess.Account)
	if err != nil {
		return nil, err
	}
	if in.DryRun {
		return map[string]any{"dry_run": true, "name": name, "chat_type": "group", "members": members}, nil
	}
	chat, err := st.CreateChat(ctx, name, members)
	if err != nil {
		return nil, err
	}
	return map[string]any{"created": true, "id": chat.ID, "name": chat.Topic, "members": members}, nil
}

func Rename(ctx context.Context, st Store, sess domain.Session, chatID, name string, dryRun bool) (any, error) {
	if err := auth.Require(sess, false, true); err != nil {
		return nil, err
	}
	if strings.TrimSpace(chatID) == "" {
		return nil, domain.Usage("chat id is required")
	}
	name, err := validTopic(name)
	if err != nil {
		return nil, err
	}
	chat, err := st.GetChat(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(chat.Type, "group") {
		return nil, domain.Usage("only group chats can be renamed")
	}
	if dryRun {
		return map[string]any{"dry_run": true, "chat_id": chatID, "old_name": chat.Topic, "name": name}, nil
	}
	if err := st.RenameChat(ctx, chatID, name); err != nil {
		return nil, err
	}
	return map[string]any{"renamed": true, "chat_id": chatID, "old_name": chat.Topic, "name": name}, nil
}

// SetHidden hides or unhides a chat in the signed-in user's own list. Nothing is deleted
// and other members still see the chat; a new message in it unhides it again.
func SetHidden(ctx context.Context, st Store, sess domain.Session, chatID string, hidden, dryRun bool) (any, error) {
	if err := auth.Require(sess, false, true); err != nil {
		return nil, err
	}
	if strings.TrimSpace(chatID) == "" {
		return nil, domain.Usage("chat id is required")
	}
	chat, err := st.GetChat(ctx, chatID)
	if err != nil {
		return nil, err
	}
	action, done := "hide", "hidden"
	if !hidden {
		action, done = "unhide", "unhidden"
	}
	out := map[string]any{"chat_id": chatID, "name": chat.Topic, "chat_type": chat.Type, "action": action}
	if dryRun {
		out["dry_run"] = true
		return out, nil
	}
	if err := st.HideChat(ctx, chatID, hidden); err != nil {
		return nil, err
	}
	out[done] = true
	return out, nil
}

func validTopic(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", domain.Usage("--name is required")
	}
	if len([]rune(name)) > MaxTopicLength {
		return "", domain.Usagef("--name is longer than %d characters", MaxTopicLength)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", domain.Usage("--name must be one line of plain text")
		}
	}
	return name, nil
}

// validMembers accepts only exact emails or user IDs, never names, and rejects
// duplicates and the signed-in user (who is always added as an owner).
func validMembers(raw []string, self string) ([]string, error) {
	seen := map[string]bool{strings.ToLower(self): self != ""}
	var out []string
	for _, m := range raw {
		m = strings.TrimSpace(m)
		if !memberID.MatchString(m) {
			addr, err := mail.ParseAddress(m)
			if err != nil || addr.Address != m {
				return nil, domain.Usagef("--member %q must be an exact email or user ID", m)
			}
		}
		key := strings.ToLower(m)
		if seen[key] {
			return nil, domain.Usagef("--member %q is listed twice or is you; you are added automatically", m)
		}
		seen[key] = true
		out = append(out, m)
	}
	if len(out) < MinGroupMembers {
		return nil, domain.Usagef("a group chat needs at least %d --member values besides you", MinGroupMembers)
	}
	return out, nil
}
