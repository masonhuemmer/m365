---
name: mail-write
description: Write Outlook mail or replies as HTML paragraphs, lists, and links with m365 mail send/reply --html.
---

# mail-write

Write Outlook mail as real HTML, not a markdown dump. No session required to read this recipe.

Use <p> for paragraphs, <ul>/<ol> for lists, and <a href> for links. Do not send a markdown heading as the mail body.

Plain text is fine for a short reply: blank lines become paragraphs and newlines stay line breaks. Replies keep the quoted thread either way.

  mail send --to you@example.com --subject 'Status' --html --body '<p>The change is in UAT.</p><ul><li>Rollback is the previous chart.</li></ul><p>Track it in <a href="https://example.com/ticket">the ticket</a>.</p>' --dry-run
  mail reply MESSAGE_ID --html --body '<p>Agreed.</p><p>I will update the ticket.</p>' --dry-run

MCP (flags.body string, not body-file=-):

  m365_run namespace=mail verb=send flags to=you@example.com subject=Status html=true body='<p>The change is in UAT.</p><ul><li>Rollback is the previous chart.</li></ul>'
  m365_run namespace=mail verb=reply args=[MESSAGE_ID] flags html=true body='<p>Agreed.</p><p>I will update the ticket.</p>'

Attach local files with --attach PATH, repeatable, on mail send and mail reply (10 MiB per file, 10 files; missing, empty or oversize files exit 3 and send nothing). Dry-run lists each file's name and size, so check it before the real send.

  mail send --to you@example.com --subject 'Report' --body 'Report attached.' --attach ~/report.pdf --dry-run

MCP: flags attach=[/path/report.pdf]. Paths are local to the machine running m365.

Copy recipients with --cc and --bcc (repeatable, mail send only). Bcc recipients are hidden from everyone else, so confirm them in the dry-run "bcc" list before the real send. With --exact-recipients every --to, --cc and --bcc must be an email address.

Happy-path examples stay dry-run (write_opt_in false). MCP writes dry-run unless write_opt_in is true.
