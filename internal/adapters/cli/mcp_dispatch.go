package cli

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/masonhuemmer/m365/internal/domain"
)

func FlagMapToArgs(ns, verb string, pos []string, flags map[string]any) ([]string, error) {
	out := []string{"m365", ns, verb}
	for _, arg := range pos {
		if strings.HasPrefix(arg, "-") {
			return nil, domain.Usagef("positional argument cannot be a flag: %q", arg)
		}
	}
	out = append(out, pos...)
	if len(flags) == 0 {
		return out, nil
	}
	keys := make([]string, 0, len(flags))
	for k := range flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !validMCPFlagKey(k) {
			return nil, domain.Usagef("unknown flag %q", k)
		}
		name := "--" + k
		switch v := flags[k].(type) {
		case nil:
		case bool:
			if v {
				if !boolFlags[name] {
					return nil, domain.Usagef("flag %s does not take a boolean value", k)
				}
				out = append(out, name)
			}
		case string:
			out = append(out, name+"="+v)
		case float64:
			if v == float64(int64(v)) {
				out = append(out, name+"="+strconv.FormatInt(int64(v), 10))
			} else {
				out = append(out, name+"="+strconv.FormatFloat(v, 'f', -1, 64))
			}
		case json.Number:
			out = append(out, name+"="+v.String())
		case []string:
			for _, s := range v {
				out = append(out, name+"="+s)
			}
		case []any:
			for _, e := range v {
				s, ok := e.(string)
				if !ok {
					return nil, domain.Usagef("flag %s values must be strings", k)
				}
				out = append(out, name+"="+s)
			}
		default:
			return nil, domain.Usagef("unsupported flag type for %s", k)
		}
	}
	return out, nil
}

func validMCPFlagKey(k string) bool {
	if k == "" || !lowerAlphaNum(k[0]) {
		return false
	}
	for i := 1; i < len(k); i++ {
		c := k[i]
		if !lowerAlphaNum(c) && c != '-' {
			return false
		}
	}
	return true
}

func lowerAlphaNum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
}

func applyWriteGate(ns, verb string, flags map[string]any, optIn bool) map[string]any {
	if !isWrite(ns, verb) || optIn {
		return flags
	}
	n := map[string]any{"dry-run": true}
	for k, v := range flags {
		n[k] = v
	}
	n["dry-run"] = true
	return n
}

func isWrite(ns, verb string) bool {
	switch ns + " " + verb {
	case "mail send", "mail reply", "teams send", "teams create", "teams rename",
		"calendar create", "calendar update", "calendar delete",
		"files upload", "files create-folder", "files delete", "files move":
		return true
	}
	return false
}

func runForbidden(ns, verb string) error {
	if ns == "mcp" {
		return &domain.Error{Class: domain.ClassUsage, Message: "mcp is not a run namespace", Hint: "use m365_status, m365_help, or m365_run"}
	}
	if ns == "auth" && (verb == "login" || verb == "logout") {
		return &domain.Error{Class: domain.ClassUsage, Message: "auth login is a human terminal command", Hint: "run m365 auth login in a terminal"}
	}
	return nil
}

func rejectStdinFiles(flags map[string]any) error {
	for _, k := range []string{"body-file", "text-file"} {
		s, _ := flags[k].(string)
		if s == "-" {
			return domain.Usagef("%s cannot be - over MCP", k)
		}
	}
	return nil
}

func buildRunArgs(ns, verb string, pos []string, flags map[string]any, optIn bool) ([]string, error) {
	if ns == "chat" {
		ns = "teams"
	}
	if ns == "" || verb == "" {
		return nil, domain.Usage("namespace and verb are required")
	}
	if err := runForbidden(ns, verb); err != nil {
		return nil, err
	}
	if err := rejectStdinFiles(flags); err != nil {
		return nil, err
	}
	if _, ok := flags["preview"]; ok {
		return nil, &domain.Error{Class: domain.ClassUsage, Message: "preview is a terminal flag", Hint: "use dry-run; its JSON has rendered and format_problems"}
	}
	return FlagMapToArgs(ns, verb, pos, applyWriteGate(ns, verb, flags, optIn))
}
