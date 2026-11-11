package cli

import (
	"flag"
	"io"

	"github.com/masonhuemmer/m365/internal/adapters/fs"
	"github.com/masonhuemmer/m365/internal/app/meetings"
	"github.com/masonhuemmer/m365/internal/domain"
)

func runMeetings(args []string, d Deps, format string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return writeHelp(d.Stdout, meetingsHelp)
	}
	verb, args := args[0], args[1:]
	if hasHelp(args) {
		return writeHelp(d.Stdout, meetingsHelp)
	}
	if d.Meetings == nil {
		return fail(d, domain.Usage("meetings not available"))
	}
	sess, err := session(d)
	if err != nil {
		return fail(d, err)
	}
	switch verb {
	case "transcripts":
		return meetingsList(args, d, sess, format)
	case "transcript":
		return meetingsSave(args, d, sess, format)
	default:
		return fail(d, domain.Usagef("unknown meetings verb %q", verb))
	}
}

func meetingsList(args []string, d Deps, sess domain.Session, format string) int {
	t, err := meetingsTarget("meetings transcripts", args, false)
	if err != nil {
		return fail(d, err)
	}
	list, err := meetings.List(ctx(), d.Meetings, sess, t)
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, list)
}

func meetingsSave(args []string, d Deps, sess domain.Session, format string) int {
	t, err := meetingsTarget("meetings transcript", args, true)
	if err != nil {
		return fail(d, err)
	}
	write := d.Write
	if write == nil {
		write = fs.WriteFile
	}
	saved, err := meetings.Save(ctx(), d.Meetings, sess, t, write)
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, saved)
}

// meetingsTarget parses --event and --join-url, plus the save flags when save is set.
func meetingsTarget(name string, args []string, save bool) (meetings.Target, error) {
	var t meetings.Target
	fsset := flag.NewFlagSet(name, flag.ContinueOnError)
	fsset.SetOutput(io.Discard) // the error is reported once, redacted, by fail
	fsset.StringVar(&t.EventID, "event", "", "")
	fsset.StringVar(&t.JoinURL, "join-url", "", "")
	if save {
		fsset.StringVar(&t.ID, "id", "", "")
		fsset.StringVar(&t.Out, "out", "", "")
		fsset.BoolVar(&t.Overwrite, "overwrite", false, "")
	}
	if err := parseMixed(fsset, args); err != nil {
		return t, domain.Usage(err.Error())
	}
	return t, nil
}
