package cli

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

var userIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func newMCPPolicy(readOnly bool, rawAllow string, exactRecipients, allowSet bool) (MCPPolicy, error) {
	p := MCPPolicy{ReadOnly: readOnly, ExactRecipients: exactRecipients}
	if rawAllow == "" {
		if allowSet {
			p.Allow = map[string]bool{}
		}
		return p, nil
	}
	p.Allow = map[string]bool{}
	for _, entry := range strings.Split(rawAllow, ",") {
		entry = strings.TrimSpace(entry)
		parts := strings.Split(entry, ".")
		if len(parts) != 2 || !isWrite(parts[0], parts[1]) {
			return MCPPolicy{}, domain.Usagef("invalid allowed write verb %q", entry)
		}
		p.Allow[entry] = true
	}
	return p, nil
}

func (p MCPPolicy) check(in *runIn) error {
	if p.ReadOnly && in.WriteOptIn {
		return domain.Usage("write_opt_in is disabled by --read-only")
	}
	ns := in.Namespace
	if ns == "chat" {
		ns = "teams"
	}
	if in.WriteOptIn && isWrite(ns, in.Verb) && p.Allow != nil && !p.Allow[ns+"."+in.Verb] {
		return domain.Usagef("%s.%s is not permitted by --allow", ns, in.Verb)
	}
	if !p.ExactRecipients || !isWrite(ns, in.Verb) {
		return nil
	}
	switch {
	case ns == "mail" && in.Verb == "reply":
		return checkReply(in.Flags)
	case in.Verb != "send":
		return nil
	case ns == "mail":
		return checkMailSend(in.Flags)
	}
	return checkTeamsSend(ns, in)
}

func checkReplyAll(flags map[string]any) error {
	if all, exists := flags["all"]; exists && all != nil && all != false && all != "false" {
		return domain.Usage("--all is not permitted with --exact-recipients")
	}
	return nil
}

func checkReply(flags map[string]any) error {
	if err := checkReplyAll(flags); err != nil {
		return err
	}
	if bcc, exists := flags["bcc"]; exists && !allEmails(bcc) {
		return domain.Usage("--bcc requires an exact email")
	}
	return nil
}

func checkMailSend(flags map[string]any) error {
	for flag, value := range flags {
		if flag == "to" || flag == "cc" || flag == "bcc" {
			if !allEmails(value) {
				return domain.Usagef("--%s requires an exact email", flag)
			}
		} else if recipientShapedFlag(flag) {
			return domain.Usagef("unknown recipient flag --%s", flag)
		}
	}
	return nil
}

func allEmails(value any) bool {
	values, ok := recipientValues(value)
	if !ok {
		return false
	}
	for _, recipient := range values {
		if !isEmail(recipient) {
			return false
		}
	}
	return true
}

func checkTeamsSend(ns string, in *runIn) error {
	to, exists := in.Flags["to"]
	if !exists {
		return nil
	}
	values, ok := recipientValues(to)
	if !ok {
		return domain.Usage("--to requires an exact email, user ID, or chat ID")
	}
	for _, value := range values {
		if ns == "teams" && isChatID(value) {
			if len(values) != 1 || len(in.Args) != 0 {
				return domain.Usage("use one chat ID or --to, not both")
			}
			in.Args = []string{value}
			delete(in.Flags, "to")
			return nil
		}
		if !isEmail(value) && !(ns == "teams" && userIDPattern.MatchString(value)) {
			return domain.Usage("--to requires an exact email, user ID, or chat ID")
		}
	}
	if ns == "teams" {
		in.Flags["exact-recipient"] = true
	}
	return nil
}

func recipientShapedFlag(flag string) bool {
	return strings.Contains(flag, "recipient") || strings.Contains(flag, "bcc") ||
		strings.HasPrefix(flag, "cc-") || strings.HasSuffix(flag, "-cc") ||
		strings.HasPrefix(flag, "to-") || strings.HasSuffix(flag, "-to")
}

func recipientValues(value any) ([]string, bool) {
	switch v := value.(type) {
	case string:
		return []string{v}, true
	case []string:
		return v, len(v) > 0
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, len(out) > 0
	}
	return nil, false
}

func isEmail(s string) bool {
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s && strings.Count(s, "@") == 1
}

func isChatID(s string) bool {
	return (strings.HasPrefix(s, "19:") || strings.HasPrefix(s, "48:")) && !strings.ContainsAny(s, " \t\n")
}
