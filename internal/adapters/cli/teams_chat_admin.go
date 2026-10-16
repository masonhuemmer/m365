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

func teamsHide(hidden bool) func([]string, Deps, domain.Session, string) int {
	return func(args []string, d Deps, sess domain.Session, format string) int {
		fsset := flag.NewFlagSet("teams hide", flag.ContinueOnError)
		fsset.SetOutput(d.Stderr)
		dry := fsset.Bool("dry-run", false, "")
		if err := parseMixed(fsset, args); err != nil {
			return fail(d, domain.Usage(err.Error()))
		}
		out, err := teams.SetHidden(ctx(), d.Teams, sess, fsset.Arg(0), hidden, *dry)
		if err != nil {
			return fail(d, err)
		}
		return success(d, format, out)
	}
}

type chatAdminVerb struct {
	run  func([]string, Deps, domain.Session, string) int
	help string
}

var chatAdminVerbs = map[string]chatAdminVerb{
	"create": {teamsCreate, teamsCreateHelp},
	"rename": {teamsRename, teamsRenameHelp},
	"hide":   {teamsHide(true), teamsHideHelp},
	"unhide": {teamsHide(false), teamsUnhideHelp},
}

// teamsChatAdmin runs the verbs that create, rename or hide chats; ok is false for any other verb.
func teamsChatAdmin(verb string, args []string, d Deps, sess domain.Session, format string) (code int, ok bool) {
	v, found := chatAdminVerbs[verb]
	if !found {
		return 0, false
	}
	if hasHelp(args) {
		return writeHelp(d.Stdout, v.help), true
	}
	return v.run(args, d, sess, format), true
}
