package teams

import (
	"context"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

// prepareFiles validates --attach files and works out who must be able to open
// them: every chat member except the signed-in user. Notes is private to the
// user, so it has nobody to share with. A member without an email address
// cannot be granted access, so it fails here, before anything is uploaded.
func prepareFiles(ctx context.Context, st Store, sess domain.Session, in *SendInput) error {
	if err := domain.ValidateOutbound(in.Files); err != nil {
		return err
	}
	if len(in.Files) == 0 || in.ChatID == SelfChatID {
		return nil
	}
	chat, err := st.GetChat(ctx, in.ChatID)
	if err != nil {
		return err
	}
	if len(chat.Members) == 0 {
		return domain.Usage("could not read the chat members, so nobody could open the files; nothing was uploaded or sent")
	}
	for _, member := range chat.Members {
		if sess.Account != "" && strings.EqualFold(member.Address, sess.Account) {
			continue
		}
		if member.Address == "" {
			return domain.Usagef("cannot share files with %q: no email address on the chat member; nothing was uploaded or sent", member.Name)
		}
		in.ShareWith = append(in.ShareWith, member.Address)
	}
	return nil
}

func dryRunResult(in SendInput, problems any) map[string]any {
	atts := []map[string]any{}
	for _, f := range in.Files {
		atts = append(atts, map[string]any{"name": f.Name, "size": f.Size})
	}
	out := map[string]any{
		"dry_run": true, "chat_id": in.ChatID, "text": in.Text, "attachments": atts,
		"rendered": in.Rendered, "format_problems": problems,
	}
	if in.To != "" {
		out["to"] = in.To
	}
	if len(in.ShareWith) > 0 {
		out["share_with"] = in.ShareWith
	}
	if len(in.Mentions) > 0 {
		who := make([]string, 0, len(in.Mentions))
		for _, m := range in.Mentions {
			who = append(who, m.Name)
		}
		out["mentions"] = who
	}
	if len(in.UnresolvedMentions) > 0 {
		out["unresolved_mentions"] = in.UnresolvedMentions
	}
	return out
}
