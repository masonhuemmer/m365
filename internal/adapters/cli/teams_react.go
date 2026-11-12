package cli

import (
	"flag"

	"github.com/masonhuemmer/m365/internal/app/teams"
	"github.com/masonhuemmer/m365/internal/domain"
)

// teamsReact runs teams react: add or remove an emoji reaction on one chat message.
func teamsReact(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("teams react", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	to := fsset.String("to", "", "")
	exact := fsset.Bool("exact-recipient", false, "")
	last := fsset.Bool("last", false, "")
	remove := fsset.Bool("remove", false, "")
	dry := fsset.Bool("dry-run", false, "")
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	in, err := reactArgs(fsset.Args())
	if err != nil {
		return fail(d, err)
	}
	in.To, in.ExactRecipient, in.Last, in.Remove, in.DryRun = *to, *exact, *last, *remove, *dry
	r, ok := d.Teams.(teams.Reactor)
	if !ok {
		return fail(d, domain.Service("the Teams client cannot send reactions"))
	}
	out, err := teams.React(ctx(), d.Teams, r, d.ChatMap, sess, in)
	if err != nil {
		return fail(d, err)
	}
	return success(d, format, out)
}

// reactArgs reads [CHAT_ID [MESSAGE_ID]] REACTION: the last argument is the reaction.
func reactArgs(pos []string) (teams.ReactInput, error) {
	var in teams.ReactInput
	switch len(pos) {
	case 0:
	case 1:
		in.Reaction = pos[0]
	case 2:
		in.ChatID, in.Reaction = pos[0], pos[1]
	case 3:
		in.ChatID, in.MessageID, in.Reaction = pos[0], pos[1], pos[2]
	default:
		return in, domain.Usage("teams react takes at most CHAT_ID MESSAGE_ID REACTION")
	}
	return in, nil
}
