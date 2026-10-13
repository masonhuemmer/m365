package cli

import (
	"flag"

	"github.com/masonhuemmer/m365/internal/adapters/fs"
	"github.com/masonhuemmer/m365/internal/app/mail"
	"github.com/masonhuemmer/m365/internal/domain"
)

func mailSend(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("mail send", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	subject := fsset.String("subject", "", "")
	body := fsset.String("body", "", "")
	bodyFile := fsset.String("body-file", "", "")
	html := fsset.Bool("html", false, "")
	formatmd := fsset.String("format", "", "")
	dry := fsset.Bool("dry-run", false, "")
	preview := fsset.Bool("preview", false, "")
	note := fsset.Bool("note-to-self", false, "")
	var to, cc, bcc, attach []string
	fsset.Func("to", "", func(s string) error { to = append(to, s); return nil })
	fsset.Func("cc", "", func(s string) error { cc = append(cc, s); return nil })
	fsset.Func("bcc", "", func(s string) error { bcc = append(bcc, s); return nil })
	fsset.Func("attach", "", func(s string) error { attach = append(attach, s); return nil })
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	md, err := markdownFlag(*html, *formatmd)
	if err != nil {
		return fail(d, err)
	}
	b, err := readBody(*body, *bodyFile, d.Stdin)
	if err != nil {
		return fail(d, err)
	}
	files, err := loadAttach(attach)
	if err != nil {
		return fail(d, err)
	}
	out, err := mail.Send(ctx(), d.Mail, sess, mail.SendInput{
		To: to, CC: cc, BCC: bcc, Subject: *subject, Body: b, HTML: *html, MD: md, DryRun: *dry || *preview, NoteToSelf: *note, Files: files,
	})
	if err != nil {
		return fail(d, err)
	}
	if *preview {
		return writePreview(d, mailPreviewHeaders(out, cc, bcc, files), out)
	}
	return success(d, format, out)
}

func mailReply(args []string, d Deps, sess domain.Session, format string) int {
	fsset := flag.NewFlagSet("mail reply", flag.ContinueOnError)
	fsset.SetOutput(d.Stderr)
	body := fsset.String("body", "", "")
	bodyFile := fsset.String("body-file", "", "")
	all := fsset.Bool("all", false, "")
	html := fsset.Bool("html", false, "")
	formatmd := fsset.String("format", "", "")
	dry := fsset.Bool("dry-run", false, "")
	preview := fsset.Bool("preview", false, "")
	var attach []string
	fsset.Func("attach", "", func(s string) error { attach = append(attach, s); return nil })
	if err := parseMixed(fsset, args); err != nil {
		return fail(d, domain.Usage(err.Error()))
	}
	id := fsset.Arg(0)
	md, err := markdownFlag(*html, *formatmd)
	if err != nil {
		return fail(d, err)
	}
	b, err := readBody(*body, *bodyFile, d.Stdin)
	if err != nil {
		return fail(d, err)
	}
	files, err := loadAttach(attach)
	if err != nil {
		return fail(d, err)
	}
	out, err := mail.Reply(ctx(), d.Mail, sess, mail.ReplyInput{
		ID: id, Body: b, All: *all, HTML: *html, MD: md, DryRun: *dry || *preview, Files: files,
	})
	if err != nil {
		return fail(d, err)
	}
	if *preview {
		return writePreview(d, replyPreviewHeaders(id, *all, files), out)
	}
	return success(d, format, out)
}

func loadAttach(paths []string) ([]domain.OutboundFile, error) {
	var files []domain.OutboundFile
	for _, p := range paths {
		f, err := fs.ReadAttach(p)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}
