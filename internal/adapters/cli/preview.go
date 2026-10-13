package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/term"

	"github.com/masonhuemmer/m365/internal/domain"
	"github.com/masonhuemmer/m365/internal/domain/msgbody"
)

const minPreviewWidth = 20

// writePreview draws the dry-run payload of a send as the preview box.
func writePreview(d Deps, headers []string, out any) int {
	m, _ := out.(map[string]any)
	r, _ := m["rendered"].(msgbody.Rendered)
	problems, _ := m["format_problems"].(msgbody.Problems)
	width := max(termWidth(d.Stdout), minPreviewWidth)
	fmt.Fprint(d.Stdout, drawPreview(headers, msgbody.Text(r.Content, width-4), problems, width))
	return domain.ExitOK
}

// drawPreview draws the preview box: header lines, a rule, the body text,
// then any format problems below the box (contracts/preview.md).
func drawPreview(headers []string, body string, problems msgbody.Problems, width int) string {
	width = max(width, minPreviewWidth)
	inner := width - 4
	rule := strings.Repeat("─", width-2)
	var b strings.Builder
	row := func(s string) {
		for _, piece := range chunk(visible(s), inner) {
			b.WriteString("│ " + piece + strings.Repeat(" ", inner-msgbody.StringWidth(piece)) + " │\n")
		}
	}
	b.WriteString("┌" + rule + "┐\n")
	for _, h := range headers {
		row(h)
	}
	b.WriteString("├" + rule + "┤\n")
	for _, l := range strings.Split(body, "\n") {
		row(l)
	}
	b.WriteString("└" + rule + "┘\n")
	if len(problems) > 0 {
		noun := "format problems"
		if len(problems) == 1 {
			noun = "format problem"
		}
		fmt.Fprintf(&b, "%d %s:\n", len(problems), noun)
		for _, p := range problems {
			b.WriteString("- " + visible(p.Rule+": "+p.Detail) + "\n")
		}
	}
	return b.String()
}

// chunk splits a line into pieces of at most n terminal columns; the box
// never grows past its width, even for an unwrapped code line.
func chunk(s string, n int) []string {
	var out []string
	for msgbody.StringWidth(s) > n {
		head, rest := msgbody.SplitWidth(s, n)
		out = append(out, head)
		s = rest
	}
	return append(out, s)
}

// visible shows control characters (ESC as \x1b) and format characters
// (bidi controls, zero-width spaces and joiners, BOM) as escapes, so
// message text cannot recolour the terminal, reorder a line or hide in
// it; a tab becomes a space.
func visible(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\t':
			b.WriteByte(' ')
		case unicode.IsControl(r) && r < 0x100:
			fmt.Fprintf(&b, `\x%02x`, r)
		case unicode.IsControl(r) || unicode.In(r, unicode.Cf, unicode.Bidi_Control):
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// termWidth is the terminal's column count, or 80 when w is not a terminal.
func termWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		fd := int(f.Fd()) // #nosec G115 -- file descriptors fit in int
		if term.IsTerminal(fd) {
			if cols, _, err := term.GetSize(fd); err == nil && cols > 0 {
				return cols
			}
		}
	}
	return 80
}

func mailPreviewHeaders(out any, cc, bcc []string, files []domain.OutboundFile) []string {
	m, _ := out.(map[string]any)
	to, _ := m["to"].([]string)
	subject, _ := m["subject"].(string)
	h := []string{"To: " + strings.Join(to, ", ")}
	if len(cc) > 0 {
		h = append(h, "Cc: "+strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		h = append(h, "Bcc: "+strings.Join(bcc, ", "))
	}
	h = append(h, "Subject: "+subject)
	return appendAttachments(h, files)
}

func replyPreviewHeaders(id string, all bool, files []domain.OutboundFile) []string {
	h := "Reply to message " + id
	if all {
		h += " (reply all)"
	}
	return appendAttachments([]string{h}, files)
}

func teamsPreviewHeaders(out any, files []domain.OutboundFile) []string {
	m, _ := out.(map[string]any)
	chat, _ := m["chat_id"].(string)
	h := "Chat: " + chat
	if to, ok := m["to"].(string); ok && to != "" {
		h = "To: " + to + " (chat " + chat + ")"
	}
	return appendAttachments([]string{h}, files)
}

func appendAttachments(h []string, files []domain.OutboundFile) []string {
	if len(files) == 0 {
		return h
	}
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = fmt.Sprintf("%s (%d B)", f.Name, f.Size)
	}
	return append(h, "Attachments: "+strings.Join(names, ", "))
}
