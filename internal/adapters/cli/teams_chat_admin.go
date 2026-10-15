package cli

import (
	"flag"

	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

func teamsCreate(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("teams create", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	name := fsset.String("name", "", "")
	dry := fsset.Bool("dry-run", false, "")
	var members []string
	fsset.Func("member", "", func(s string) error { members = append(members, s); return nil })
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	if fsset.NArg() > 0 {
		return fail(d, domain.Usage("teams create takes --name and --member, no positional arguments"))
	}
	out, err := teams.CreateGroup(ctx(), d.Teams, sess, teams.CreateInput{Name: *name, Members: members, DryRun: *dry})
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, out)
}

func teamsRename(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("teams rename", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	name := fsset.String("name", "", "")
	dry := fsset.Bool("dry-run", false, "")
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	out, err := teams.Rename(ctx(), d.Teams, sess, fsset.Arg(0), *name, *dry)
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, out)
}

// teamsChatAdmin runs the verbs that create or rename chats; ok is false for any other verb.
func teamsChatAdmin(verb string, args []string, d Deps, sess domain.Session, format string) (code int, ok bool) {
	run, help := teamsCreate, teamsCreateHelp
	switch verb {
	case "create":
	case "rename":
		run, help = teamsRename, teamsRenameHelp
	default:
		return 0, false
	}
	if hasHelp(args) {
		return writeHelp(d.Stdout, help), true
	}
	return run(args, d, sess, format), true
}
