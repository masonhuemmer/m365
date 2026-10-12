package cli

import (
	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/domain"
)

func runAuth(args []string, d Deps, format string, verbose bool) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return writeHelp(d.Stdout, rootHelp)
	}
	verb := args[0]
	switch verb {
	case "status":
		st, err := auth.Status(d.Store)
		if err != nil {
			return fail(d, err)
		}
		out := map[string]any{
			"signed_in":      st.SignedIn,
			"session_usable": st.SessionUsable,
			"account":        st.Account,
			"namespaces":     nsMap(st),
		}
		if verbose {
			fmtVerbose(d, "status ok")
		}
		return success(d, format, out)
	case "logout":
		if err := auth.Logout(d.Store); err != nil {
			return fail(d, err)
		}
		st := domain.SignedOut()
		return success(d, format, map[string]any{
			"signed_in": st.SignedIn, "session_usable": st.SessionUsable,
			"namespaces": nsMap(st),
		})
	case "login":
		return runLogin(args[1:], d, format)
	default:
		return fail(d, domain.Usagef("unknown auth verb %q", verb))
	}
}

func nsMap(st domain.Session) map[string]bool {
	return map[string]bool{
		"mail": st.MailConsented, "teams": st.TeamsConsented,
		"calendar": st.CalendarConsented, "files": st.FilesConsented,
	}
}

func fmtVerbose(d Deps, s string) {
	_, _ = d.Stderr.Write([]byte(redact(s) + "\n"))
}

func runLogin(flags []string, d Deps, format string) int {
	if err := d.Config.RequireApp(); err != nil {
		return fail(d, err)
	}
	fn := d.Login
	if len(flags) > 0 {
		if flags[0] != "--device-code" || len(flags) > 1 {
			return fail(d, domain.Usagef("unknown login flag %q", flags[0]))
		}
		fn = d.DeviceLogin
	}
	st, err := auth.Login(ctx(), d.Store, fn)
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, map[string]any{
		"signed_in": st.SignedIn, "session_usable": st.SessionUsable, "account": st.Account,
		"namespaces": nsMap(st),
	})
}
