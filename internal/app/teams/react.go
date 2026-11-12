package teams

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

// Reactor adds or removes the signed-in user's reaction on one chat message.
type Reactor interface {
	React(ctx context.Context, chatID, messageID, emoji string, remove bool) error
}

// ReactInput addresses one message by chat and message ID, or by --last in a chat or a --to match.
type ReactInput struct {
	ChatID         string
	MessageID      string
	To             string
	Reaction       string
	Last           bool
	Remove         bool
	DryRun         bool
	ExactRecipient bool
}

// LastMessageScan is how many messages --last looks at: the first page Graph returns.
const LastMessageScan = 50

// React adds or removes a reaction on one chat message.
func React(ctx context.Context, st Store, r Reactor, m ChatMap, sess domain.Session, in ReactInput) (any, error) {
	if err := auth.Require(sess, false, true); err != nil {
		return nil, err
	}
	if err := checkReactForm(in); err != nil {
		return nil, err
	}
	emoji, err := domain.ParseReaction(in.Reaction)
	if err != nil {
		return nil, err
	}
	if in.To != "" {
		if in.ChatID, err = resolveRecipient(ctx, st, m, sess, SendInput{To: in.To, ExactRecipient: in.ExactRecipient}); err != nil {
			return nil, err
		}
	}
	out := map[string]any{"chat_id": in.ChatID, "reaction": emoji}
	if in.Last {
		msg, err := lastFromOthers(ctx, st, sess, in.ChatID)
		if err != nil {
			return nil, err
		}
		in.MessageID = msg.ID
		out["from"], out["created"], out["text"] = msg.From, msg.Created, msg.Text
	}
	out["message_id"] = in.MessageID
	if in.DryRun {
		out["dry_run"] = true
		out["action"] = map[bool]string{false: "react", true: "remove"}[in.Remove]
		return out, nil
	}
	if err := r.React(ctx, in.ChatID, in.MessageID, emoji, in.Remove); err != nil {
		return nil, err
	}
	out[map[bool]string{false: "reacted", true: "removed"}[in.Remove]] = true
	return out, nil
}

func checkReactForm(in ReactInput) error {
	switch {
	case in.To != "" && in.ChatID != "":
		return domain.Usage("use --to or a chat id, not both")
	case in.To != "" && !in.Last:
		return domain.Usage("--to needs --last; to pick another message, pass CHAT_ID MESSAGE_ID")
	case in.Last && in.MessageID != "":
		return domain.Usage("use --last or a message id, not both")
	case in.To == "" && in.ChatID == "":
		return domain.Usage("chat id is required (or --to NAME --last)")
	case !in.Last && in.MessageID == "":
		return domain.Usage("message id is required (or --last)")
	}
	return nil
}

// lastFromOthers picks the newest message, by created time, that has a sender other than
// you, among the first LastMessageScan messages. Graph lists them by last change.
func lastFromOthers(ctx context.Context, st Store, sess domain.Session, chatID string) (domain.ChatMessage, error) {
	chat, err := st.GetChat(ctx, chatID)
	if err != nil {
		return domain.ChatMessage{}, err
	}
	self := memberIDFor(chat, sess.Account)
	if self == "" {
		return domain.ChatMessage{}, domain.Usage("cannot tell which messages in this chat are yours; pass CHAT_ID MESSAGE_ID")
	}
	p, err := st.Messages(ctx, MessageQuery{ChatID: chatID, Top: LastMessageScan})
	if err != nil {
		return domain.ChatMessage{}, err
	}
	var best domain.ChatMessage
	var bestAt time.Time
	found := false
	for _, msg := range p.Items {
		if msg.System || msg.FromUserID == self || (msg.FromUserID == "" && msg.FromApplicationID == "") {
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, msg.Created)
		if !found || at.After(bestAt) {
			best, bestAt, found = msg, at, true
		}
	}
	if !found {
		return domain.ChatMessage{}, domain.NotFound(fmt.Sprintf("no message from someone else among the %d most recently changed messages", LastMessageScan))
	}
	return best, nil
}

func memberIDFor(chat domain.Chat, account string) string {
	if account == "" {
		return ""
	}
	for _, member := range chat.Members {
		if strings.EqualFold(member.Address, account) {
			return member.ID
		}
	}
	return ""
}
