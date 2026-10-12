package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/masonhuemmer/m365/internal/app/auth"
	"github.com/masonhuemmer/m365/internal/app/calendar"
	"github.com/masonhuemmer/m365/internal/app/files"
	"github.com/masonhuemmer/m365/internal/app/mail"
	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/config"
	"github.com/masonhuemmer/m365/internal/domain"
	"github.com/masonhuemmer/m365/internal/version"
)

type Deps struct {
	Config              config.Config
	ConfigError         error
	Store               auth.Store
	Mail                mail.Store
	MailChanges         mail.ChangeStore
	MailThreads         mail.ThreadStore
	MailClassifier      mail.ResponseClassifier
	MailClassifierError error
	MailWatchState      mail.WatchStateStore
	Teams               teams.Store
	ChatMap             teams.ChatMap
	Calendar            calendar.Store
	Files               files.Store
	Login               auth.LoginFn
	DeviceLogin         auth.LoginFn
	Write               func(string, []byte, bool) error
	Watch               WatchStore
	Stdin               io.Reader
	Stdout              io.Writer
	Stderr              io.Writer
}

type WatchStore interface {
	Load() (domain.WatchCheckpoint, error)
	Save(domain.WatchCheckpoint) error
}

func Run(args []string, d Deps) int {
	if d.Stdin == nil {
		d.Stdin = os.Stdin
	}
	if d.Stdout == nil {
		d.Stdout = os.Stdout
	}
	if d.Stderr == nil {
		d.Stderr = os.Stderr
	}
	human, jsonOn, verbose, showVersion, rest := peelGlobals(args)
	if showVersion {
		fmt.Fprintln(d.Stdout, version.Version)
		return domain.ExitOK
	}
	if human && jsonOn {
		return fail(d, domain.Usage("use only one of --json or --human"))
	}
	if jsonOn && previewRequested(rest) {
		return fail(d, domain.Usage("use only one of --preview or --json"))
	}
	format := "json"
	if human {
		format = "human"
	}
	if len(rest) == 0 || rest[0] == "help" {
		return writeHelp(d.Stdout, rootHelp)
	}
	ns, rest := rest[0], rest[1:]
	if ns == "chat" {
		ns = "teams"
	}
	switch ns {
	case "auth":
		return runAuth(rest, d, format, verbose)
	case "mail":
		return runMail(rest, d, format)
	case "teams":
		return runTeams(rest, d, format)
	case "calendar":
		return runCalendar(rest, d, format)
	case "files":
		return runFiles(rest, d, format)
	case "mcp":
		return runMCP(rest, d, format)
	default:
		if strings.HasPrefix(ns, "-") {
			return writeHelp(d.Stdout, rootHelp)
		}
		return fail(d, domain.Usagef("unknown namespace %q", ns))
	}
}

func peelGlobals(args []string) (human, jsonOn, verbose, showVersion bool, rest []string) {
	if len(args) > 0 {
		base := args[0]
		if strings.HasSuffix(base, "m365") || strings.HasSuffix(base, "m365.exe") || strings.Contains(base, "/") {
			args = args[1:]
		}
	}
	sawCmd := false
	for _, a := range args {
		switch a {
		case "--human":
			human = true
		case "--json":
			jsonOn = true
		case "--verbose", "--debug":
			verbose = true
		case "--version":
			showVersion = true
		case "--help", "-h":
			if !sawCmd {
				rest = append(rest, "help")
			} else {
				rest = append(rest, a)
			}
		default:
			if !strings.HasPrefix(a, "-") {
				sawCmd = true
			}
			rest = append(rest, a)
		}
	}
	return
}

func fail(d Deps, err error) int {
	code := domain.ExitOf(err)
	cls := domain.ClassOf(err)
	msg := err.Error()
	hint := ""
	if de, ok := err.(*domain.Error); ok {
		hint = de.Hint
	}
	enc := json.NewEncoder(d.Stderr)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(errObj{Class: cls, Message: redact(msg), Hint: redact(hint)})
	return code
}

type errObj struct {
	Class   string `json:"class"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

func success(d Deps, format string, v any) int {
	if format == "human" {
		fmt.Fprintln(d.Stdout, humanize(v))
		return domain.ExitOK
	}
	enc := json.NewEncoder(d.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	return domain.ExitOK
}

func writeHelp(w io.Writer, s string) int {
	fmt.Fprint(w, s)
	if !strings.HasSuffix(s, "\n") {
		fmt.Fprintln(w)
	}
	return domain.ExitOK
}

func readBody(flagVal, fileVal string, stdin io.Reader) (string, error) {
	if fileVal != "" {
		if fileVal == "-" {
			b, err := io.ReadAll(stdin)
			return string(b), err
		}
		b, err := os.ReadFile(fileVal) // #nosec G304 -- caller --body-file/--text-file path
		if err != nil {
			return "", domain.Usagef("cannot read body file: %s", fileVal)
		}
		return string(b), nil
	}
	return flagVal, nil
}

func humanize(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return strings.TrimSpace(buf.String())
}

func session(d Deps) (domain.Session, error) {
	return auth.Status(d.Store)
}

func ctx() context.Context { return context.Background() }

var boolFlags = map[string]bool{
	"--dry-run": true, "--html": true, "--overwrite": true, "--unread": true,
	"--all": true, "--include-system": true, "--help": true, "-h": true,
	"--json": true, "--human": true, "--verbose": true, "--debug": true,
	"--bodies": true, "--group": true, "--include-existing": true, "--classify": true,
	"--version": true, "--note-to-self": true, "--preview": true, "--exact-recipient": true,
}

func parseMixed(fsset *flag.FlagSet, args []string) error {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if strings.Contains(a, "=") || boolFlags[a] {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return fsset.Parse(append(flags, pos...))
}
