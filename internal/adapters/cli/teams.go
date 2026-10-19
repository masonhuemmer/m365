package cli

import (
	"encoding/json"
	"flag"
	"strings"

	"github.com/masonhuemmer/m365/internal/adapters/fs"
	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

func runTeams(args []string, d Deps, format string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return writeHelp(d.Stdout, teamsListHelp+teamsMessagesHelp+teamsFindHelp+teamsSendHelp+teamsCreateHelp+teamsRenameHelp+teamsHideHelp+teamsUnhideHelp)
	}
	verb, args := args[0], args[1:]
	sess, err := session(d)
	if err != nil {
		return fail(d, err)
	}
	if code, ok := teamsChatAdmin(verb, args, d, sess, format); ok {
		return code
	}
	switch verb {
	case "list":
		if hasHelp(args) {
			return writeHelp(d.Stdout, teamsListHelp)
		}
		fsset := flag.NewFlagSet("teams list", flag.ContinueOnError)
		fsset.SetOutput(d.Stderr)
		top := fsset.Int("top", 0, "")
		page := fsset.String("page-token", "", "")
		hidden := fsset.Bool("include-hidden", false, "")
		if err := parseMixed(fsset, args); err != nil {
			return fail(d, domain.Usage(err.Error()))
		}
		p, err := teams.ListMapped(ctx(), d.Teams, d.ChatMap, sess, *top, *page, *hidden)
		if err != nil {
			return fail(d, err)
		}
		return success(d, format, p)
	case "get":
		return teamsGet(args, d, sess, format)
	case "messages":
		if hasHelp(args) {
			return writeHelp(d.Stdout, teamsMessagesHelp)
		}
		fsset := flag.NewFlagSet("teams messages", flag.ContinueOnError)
		fsset.SetOutput(d.Stderr)
		top := fsset.Int("top", 0, "")
		page := fsset.String("page-token", "", "")
		sys := fsset.Bool("include-system", false, "")
		if err := parseMixed(fsset, args); err != nil {
			return fail(d, domain.Usage(err.Error()))
		}
		id := fsset.Arg(0)
		p, err := teams.Messages(ctx(), d.Teams, sess, teams.MessageQuery{
			ChatID: id, Top: *top, PageToken: *page, IncludeSystem: *sys,
		})
		if err != nil {
			return fail(d, err)
		}
		return success(d, format, p)
	case "find":
		if hasHelp(args) {
			return writeHelp(d.Stdout, teamsFindHelp)
		}
		return teamsFind(args, d, sess, format)
	case "send":
		if hasHelp(args) {
			return writeHelp(d.Stdout, teamsSendHelp)
		}
		return teamsSend(args, d, sess, format)
	case "watch":
		return teamsWatch(args, d, sess, format)
	case "attachments":
		if len(args) < 2 {
			return fail(d, domain.Usage("chat id and message id are required"))
		}
		atts, err := teams.ListAttachments(ctx(), d.Teams, sess, args[0], args[1])
		if err != nil {
			return fail(d, err)
		}
		return success(d, format, map[string]any{"limit": len(atts), "count": len(atts), "items": atts})
	case "save-attachment":
		return teamsSave(args, d, sess, format)
	default:
		return fail(d, domain.Usagef("unknown teams verb %q", verb))
	}
}

func teamsGet(args []string, d Deps, sess domain.Session, format string) int {
	if len(args) < 1 {
		return fail(d, domain.Usage("chat id is required"))
	}
	c, err := teams.Get(ctx(), d.Teams, sess, args[0])
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, c)
}

func teamsSend(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("teams send", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	text := fsset.String("text", "", "")
	textFile := fsset.String("text-file", "", "")
	to := fsset.String("to", "", "")
	exactRecipient := fsset.Bool("exact-recipient", false, "")
	html := fsset.Bool("html", false, "")
	formatmd := fsset.String("format", "", "")
	dry := fsset.Bool("dry-run", false, "")
	preview := fsset.Bool("preview", false, "")
	note := fsset.Bool("note-to-self", false, "")
	var attach []string
	fsset.Func("attach", "", func(s string) error { attach = append(attach, s); return nil })
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	md, err := markdownFlag(*html, *formatmd)
	if err != nil {
		return fail(d, err)
	}
	body, err := readBody(*text, *textFile, d.Stdin)
	if err != nil {
		return fail(d, err)
	}
	files, err := loadAttach(attach)
	if err != nil {
		return fail(d, err)
	}
	out, err := teams.SendMapped(ctx(), d.Teams, d.ChatMap, sess, teams.SendInput{
		ChatID: fsset.Arg(0), To: *to, Text: body, HTML: *html, MD: md, DryRun: *dry || *preview, NoteToSelf: *note, ExactRecipient: *exactRecipient, Files: files,
	})
	if err != nil {
		return fail(d, err)
	}
	if *preview {
		return writePreview(d, teamsPreviewHeaders(out, files), out)
	}
	return success(d, format, out)
}

func teamsWatch(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("teams watch", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	since := fsset.String("since", "", "")
	top := fsset.Int("top", 0, "")
	var chats []string
	fsset.Func("chat", "", func(s string) error { chats = append(chats, s); return nil })
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	ev, err := teams.Watch(ctx(), d.Teams, sess, teams.WatchQuery{Since: *since, Chats: chats, Top: *top})
	if err != nil {
		return fail(d, err)
	}
	if d.Watch != nil && len(ev) > 0 {
		cp, _ := d.Watch.Load()
		if cp.Marks == nil {
			cp.Marks = map[string]domain.WatchMark{}
		}
		for _, e := range ev {
			cp.Marks[e.ChatID] = domain.WatchMark{MessageID: e.MessageID, Created: e.Created}
		}
		_ = d.Watch.Save(cp)
	}
	if format != "human" {
		enc := json.NewEncoder(d.Stdout)
		enc.SetEscapeHTML(false)
		for _, e := range ev {
			_ = enc.Encode(e)
		}
		return domain.ExitOK
	}
	for _, e := range ev {
		_, _ = d.Stdout.Write([]byte(e.Text + "\n"))
	}
	return domain.ExitOK
}

func teamsSave(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("teams save-attachment", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	outp := fsset.String("out", "", "")
	ov := fsset.Bool("overwrite", false, "")
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	write := d.Write
	if write == nil {
		write = fs.WriteFile
	}
	path, err := teams.Save(ctx(), d.Teams, sess, fsset.Arg(0), fsset.Arg(1), fsset.Arg(2), *outp, *ov, write)
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, map[string]any{"path": path})
}

func splitPos(args []string) (flags []string, pos []string) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
		} else {
			pos = append(pos, a)
		}
	}
	return
}
